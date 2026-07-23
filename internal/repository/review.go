package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/dapi/code-converge/internal/runner"
)

var errNoProviderRemote = errors.New("no provider remote is configured")

// ghPRListLimit makes gh paginate through a high ceiling of candidates instead
// of deciding uniqueness from its default first page of 30 pull requests.
// gh stops earlier when the candidate collection is exhausted.
const ghPRListLimit = "1000000"

const disableSplitIndexConfig = "core.splitIndex=false"

// ReviewTarget is the resolved base and scoped Git environment used for one review.
type ReviewTarget struct {
	Base       string
	BaseCommit string
	MergeBase  string
	Source     string
	Env        []string
	UnsetEnv   []string
}

var gitTransportEnvironment = []string{
	"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE",
	"GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE", "GIT_CEILING_DIRECTORIES", "GIT_EXEC_PATH",
	"GIT_DISCOVERY_ACROSS_FILESYSTEM", "GIT_IMPLICIT_WORK_TREE",
}

// ReviewScope discovers a base once and refreshes a private index before each review.
// It never changes the caller's real Git index or worktree.
type ReviewScope struct {
	Runner runner.Runner
	Base   string
	Root   string

	base, baseCommit, mergeBase, source  string
	tempDir, gitWrapperDir, gitHelperDir string
	copyIndex                            func(context.Context, string) error
}

type scopedGitConfiguration struct {
	Executable string `json:"executable"`
	Root       string `json:"root"`
	Index      string `json:"index"`
	WrapperDir string `json:"wrapper_dir"`
	HelperDir  string `json:"helper_dir"`
}

func (s *ReviewScope) Prepare(ctx context.Context) (ReviewTarget, error) {
	if s.base == "" {
		base, source, err := s.discover(ctx)
		if err != nil {
			return ReviewTarget{}, err
		}
		baseCommit, err := s.git(ctx, nil, "rev-parse", "--verify", base+"^{commit}")
		if err != nil {
			return ReviewTarget{}, fmt.Errorf("resolve selected base %q: %w", base, err)
		}
		mergeBase, err := s.git(ctx, nil, "merge-base", "HEAD", baseCommit)
		if err != nil {
			return ReviewTarget{}, fmt.Errorf("find merge-base for %q: %w", base, err)
		}
		s.base, s.baseCommit, s.mergeBase, s.source = base, baseCommit, mergeBase, source
	}
	if s.tempDir == "" {
		dir, err := s.createReviewTempDir()
		if err != nil {
			return ReviewTarget{}, err
		}
		if err := validatePathListEntry(filepath.Join(dir, "bin")); err != nil {
			_ = os.RemoveAll(dir)
			return ReviewTarget{}, err
		}
		if err := validatePathListEntry(filepath.Join(dir, "git-exec")); err != nil {
			_ = os.RemoveAll(dir)
			return ReviewTarget{}, err
		}
		s.tempDir = dir
	}
	env, err := s.reviewEnvironment()
	if err != nil {
		return ReviewTarget{}, err
	}
	if err := s.copyRealIndex(ctx, filepath.Join(s.tempDir, "index")); err != nil {
		return ReviewTarget{}, fmt.Errorf("prepare review index: %w", err)
	}
	if err := s.clearHiddenSnapshotIndexFlags(ctx); err != nil {
		return ReviewTarget{}, fmt.Errorf("prepare review index flags: %w", err)
	}
	if err := s.snapshotWorktree(ctx); err != nil {
		return ReviewTarget{}, fmt.Errorf("snapshot worktree for review: %w", err)
	}
	return ReviewTarget{
		Base: s.base, BaseCommit: s.baseCommit, MergeBase: s.mergeBase, Source: s.source,
		Env: env, UnsetEnv: append([]string(nil), gitTransportEnvironment...),
	}, nil
}

func (s *ReviewScope) snapshotWorktree(ctx context.Context) error {
	environment := s.snapshotEnvironment()
	if _, err := s.git(ctx, environment, s.rootGitArgs("add", "--sparse", "-A")...); err == nil {
		return nil
	} else {
		sparse, detectionErr := s.sparseCheckoutEnabled(ctx)
		if detectionErr != nil {
			return fmt.Errorf("git add --sparse failed: %v; determine sparse-checkout state: %w", err, detectionErr)
		}
		if sparse {
			return fmt.Errorf("this sparse checkout requires Git with git add --sparse support: %w", err)
		}
		// --sparse is newer than the oldest Git installations supported by the
		// general Linux contract. Outside a sparse checkout, plain add -A has
		// equivalent snapshot semantics and is the compatible fallback.
		if _, fallbackErr := s.git(ctx, environment, s.rootGitArgs("add", "-A")...); fallbackErr != nil {
			return fmt.Errorf("git add --sparse failed: %v; compatible git add -A fallback failed: %w", err, fallbackErr)
		}
		return nil
	}
}

func (s *ReviewScope) sparseCheckoutEnabled(ctx context.Context) (bool, error) {
	result, err := s.runGit(ctx, nil, s.rootGitArgs("config", "--bool", "core.sparseCheckout")...)
	value := strings.TrimSpace(result.Stdout)
	if err == nil {
		return value == "true", nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return false, ctxErr
	}
	// Git exits 1 with no diagnostic when the key is unset. That is the normal
	// non-sparse repository state, not a discovery failure.
	if result.ExitCode == 1 && value == "" && strings.TrimSpace(result.Stderr) == "" {
		return false, nil
	}
	return false, err
}

func (s *ReviewScope) Close() error {
	if s.tempDir == "" {
		return nil
	}
	err := os.RemoveAll(s.tempDir)
	s.tempDir = ""
	s.gitWrapperDir = ""
	s.gitHelperDir = ""
	return err
}

func (s *ReviewScope) discover(ctx context.Context) (string, string, error) {
	if strings.TrimSpace(s.Base) != "" {
		base, err := s.resolve(ctx, s.Base)
		if err != nil {
			return "", "", fmt.Errorf("resolve explicit review base %q: %w", s.Base, err)
		}
		return base, "explicit", nil
	}
	branch, err := s.git(ctx, nil, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return "", "", fmt.Errorf("discover review base: detached HEAD requires --review-base")
	}
	if candidate, found, err := s.openPRBase(ctx, branch); err != nil {
		return "", "", err
	} else if found {
		base, err := s.resolveOpenPRBase(ctx, candidate.Name, candidate.Commit, candidate.Target)
		if err != nil {
			return "", "", fmt.Errorf("resolve open PR base %q: %w", candidate.Name, err)
		}
		return base, "open_pr", nil
	}
	if candidate, err := s.git(ctx, nil, "config", "--get", "branch."+branch+".gh-merge-base"); err == nil && candidate != "" {
		base, err := s.resolve(ctx, candidate)
		if err != nil {
			return "", "", fmt.Errorf("resolve branch merge base %q: %w", candidate, err)
		}
		return base, "branch_merge_base", nil
	}
	remotes, err := s.git(ctx, nil, "remote")
	if err != nil {
		return "", "", fmt.Errorf("list remotes: %w", err)
	}
	var candidates []string
	for _, remote := range strings.Fields(remotes) {
		candidate, err := s.git(ctx, nil, "symbolic-ref", "--quiet", "refs/remotes/"+remote+"/HEAD")
		if err == nil && candidate != "" {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) != 1 {
		return "", "", fmt.Errorf("discover review base: found %d remote default branches; set --review-base", len(candidates))
	}
	base, err := s.resolve(ctx, candidates[0])
	if err != nil {
		return "", "", fmt.Errorf("resolve remote default %q: %w", candidates[0], err)
	}
	return base, "remote_default", nil
}

type providerBase struct {
	Name           string             `json:"baseRefName"`
	Commit         string             `json:"baseRefOid"`
	HeadName       string             `json:"headRefName"`
	HeadRepository providerRepository `json:"headRepository"`
	Target         string             `json:"-"`
}

type providerRepository struct {
	NameWithOwner string `json:"nameWithOwner"`
}

func (s *ReviewScope) openPRBase(ctx context.Context, branch string) (providerBase, bool, error) {
	identity, err := s.currentBranchProvider(ctx, branch)
	if err != nil {
		if errors.Is(err, errNoProviderRemote) {
			return providerBase{}, false, nil // Provider discovery is optional without a provider remote.
		}
		return providerBase{}, false, fmt.Errorf("verify open PR head for branch %q: %w", branch, err)
	}
	targets, err := s.providerTargets(ctx, identity)
	if err != nil {
		return providerBase{}, false, fmt.Errorf("discover possible open PR targets: %w", err)
	}
	var matches []providerBase
	sawCandidate := false
	var unavailableTarget string
	var unavailableError error
	for _, target := range targets {
		output, err := s.Runner.Run(ctx, runner.Invocation{Executable: "gh", Args: []string{"pr", "list", "--repo", target, "--head", branch, "--state", "open", "--json", "baseRefName,baseRefOid,headRefName,headRepository", "--limit", ghPRListLimit}})
		if err != nil {
			// Provider discovery is optional only when it produces no candidate.
			// Once another target returns a matching PR, every target is needed to
			// establish that the match is unique.
			if unavailableError == nil {
				unavailableTarget, unavailableError = target, err
			}
			continue
		}
		var prs []providerBase
		if err := json.Unmarshal([]byte(output.Stdout), &prs); err != nil {
			return providerBase{}, false, fmt.Errorf("parse open PR candidates from %q: %w", target, err)
		}
		if len(prs) > 0 {
			sawCandidate = true
		}
		for _, pr := range prs {
			if strings.TrimSpace(pr.Name) == "" || strings.TrimSpace(pr.Commit) == "" ||
				strings.TrimSpace(pr.HeadName) == "" || strings.TrimSpace(pr.HeadRepository.NameWithOwner) == "" {
				return providerBase{}, false, fmt.Errorf("open PR has incomplete head or base metadata")
			}
			if pr.HeadName == branch && strings.EqualFold(pr.HeadRepository.NameWithOwner, repositoryNameWithOwner(identity)) {
				pr.Target = target
				matches = append(matches, pr)
			}
		}
	}
	if len(matches) > 0 && unavailableError != nil {
		return providerBase{}, false, fmt.Errorf("discover review base: query open pull requests from %q while another provider target returned a match: %w; set --review-base", unavailableTarget, unavailableError)
	}
	switch len(matches) {
	case 0:
		if !sawCandidate {
			return providerBase{}, false, nil
		}
		return providerBase{}, false, fmt.Errorf("discover review base: open pull request head does not match current branch provider %q; set --review-base", identity)
	case 1:
		return matches[0], true, nil
	default:
		return providerBase{}, false, fmt.Errorf("discover review base: found %d open pull requests for current branch provider; set --review-base", len(matches))
	}
}

// createReviewTempDir keeps the private index transport outside the worktree.
// os.MkdirTemp honors TMPDIR, which may point into the repository being
// snapshotted. In that case, its wrapper and index files would otherwise be
// picked up by git add --sparse -A.
func (s *ReviewScope) createReviewTempDir() (string, error) {
	dir, err := os.MkdirTemp("", "code-converge-review-index-")
	if err != nil {
		return "", fmt.Errorf("create review index: %w", err)
	}
	canonicalDir, err := canonicalPath(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("resolve review index path: %w", err)
	}
	reviewRoot := s.Root
	if reviewRoot == "" {
		reviewRoot, err = os.Getwd()
		if err != nil {
			_ = os.RemoveAll(dir)
			return "", fmt.Errorf("resolve reviewed worktree: %w", err)
		}
	}
	canonicalRoot, err := canonicalPath(reviewRoot)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("resolve reviewed worktree: %w", err)
	}
	if !pathWithin(canonicalDir, canonicalRoot) {
		return canonicalDir, nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("remove unsafe review index: %w", err)
	}
	parent := filepath.Dir(canonicalRoot)
	if parent == canonicalRoot {
		return "", fmt.Errorf("create review index: reviewed worktree %q has no parent outside the worktree", canonicalRoot)
	}
	dir, err = os.MkdirTemp(parent, "code-converge-review-index-")
	if err != nil {
		return "", fmt.Errorf("create review index outside reviewed worktree: %w", err)
	}
	canonicalDir, err = canonicalPath(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("resolve review index path: %w", err)
	}
	if pathWithin(canonicalDir, canonicalRoot) {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("create review index: temporary directory %q is inside reviewed worktree %q", canonicalDir, canonicalRoot)
	}
	return canonicalDir, nil
}

func pathWithin(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

// providerTargets returns every configured provider repository that could own
// the PR. In the common fork workflow, the push remote identifies the PR head,
// while an upstream remote owns the PR itself.
func (s *ReviewScope) providerTargets(ctx context.Context, head string) ([]string, error) {
	remotes, err := s.git(ctx, nil, "remote")
	if err != nil {
		return nil, err
	}
	targets := []string{head}
	seen := map[string]struct{}{head: {}}
	for _, remote := range strings.Fields(remotes) {
		urls, err := s.git(ctx, nil, "remote", "get-url", "--all", remote)
		if err != nil {
			continue
		}
		for _, remoteURL := range strings.Fields(urls) {
			identity, err := repositoryIdentity(remoteURL)
			if err != nil {
				continue
			}
			if providerHost(identity) == providerHost(head) {
				if _, exists := seen[identity]; !exists {
					seen[identity] = struct{}{}
					targets = append(targets, identity)
				}
			}
		}
	}
	return targets, nil
}

func (s *ReviewScope) currentBranchProvider(ctx context.Context, branch string) (string, error) {
	remote := ""
	for _, key := range []string{"branch." + branch + ".pushRemote", "remote.pushDefault", "branch." + branch + ".remote"} {
		if candidate, err := s.git(ctx, nil, "config", "--get", key); err == nil && candidate != "" {
			remote = candidate
			break
		}
	}
	if remote == "." {
		return "", errNoProviderRemote
	}
	if remote == "" {
		remote = "origin"
	}
	// Git's missing-remote diagnostics are localized. Pin this one diagnostic-
	// bearing call to the C locale so the narrow optional-provider fallback below
	// is deterministic across user environments.
	remoteURLResult, err := s.Runner.Run(ctx, runner.Invocation{
		Executable: "git",
		Args:       []string{"remote", "get-url", "--push", "--all", remote},
		Env:        []string{"LC_ALL=C"},
		UnsetEnv:   gitTransportEnvironment,
	})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// A branch may retain a push-remote setting after the remote has
		// been removed. This is equivalent to having no provider remote:
		// local base discovery must remain available. Restrict this exception
		// to Git's expected missing-remote/no-URL diagnostics so transport,
		// configuration, and other command failures still surface.
		if isMissingProviderRemoteDiagnostic(remoteURLResult.Stderr) {
			return "", errNoProviderRemote
		}
		return "", fmt.Errorf("read push URL for remote %q: %w", remote, err)
	}
	remoteURLs := strings.TrimSpace(remoteURLResult.Stdout)
	identities := make(map[string]struct{})
	unsupportedURL := false
	for _, remoteURL := range strings.Fields(remoteURLs) {
		identity, err := repositoryIdentity(remoteURL)
		if err != nil {
			unsupportedURL = true
			continue
		}
		identities[identity] = struct{}{}
	}
	// Git pushes to every configured pushurl. A provider identity is therefore
	// trustworthy only when every destination identifies that same provider;
	// mixed local/unsupported destinations leave provider discovery unavailable.
	if unsupportedURL && len(identities) > 0 {
		return "", errNoProviderRemote
	}
	if len(identities) == 0 {
		return "", errNoProviderRemote
	}
	if len(identities) > 1 {
		return "", fmt.Errorf("remote %q has %d conflicting push repository identities", remote, len(identities))
	}
	for identity := range identities {
		return identity, nil
	}
	return "", errors.New("unreachable provider identity")
}

func isMissingProviderRemoteDiagnostic(stderr string) bool {
	diagnostic := strings.ToLower(strings.TrimSpace(stderr))
	return strings.HasPrefix(diagnostic, "error: no such remote ") ||
		strings.HasPrefix(diagnostic, "fatal: no such remote ") ||
		strings.Contains(diagnostic, "no such url found")
}

func repositoryIdentity(remoteURL string) (string, error) {
	value := strings.TrimSuffix(strings.TrimSpace(remoteURL), "/")
	var host string
	var path string
	if scpHost, repositoryPath, ok := splitSCPRemote(value); ok {
		host, path = scpHost, repositoryPath
	} else if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" {
		if !isProviderURLScheme(parsed.Scheme) {
			return "", fmt.Errorf("remote URL uses unsupported provider scheme %q", parsed.Scheme)
		}
		host, path = parsed.Hostname(), parsed.Path
		if port := parsed.Port(); port != "" && !isDefaultPort(parsed.Scheme, port) {
			host = net.JoinHostPort(host, port)
		} else if strings.Contains(host, ":") {
			// url.Hostname removes the brackets required to delimit an IPv6
			// literal from the following repository path. Restore them even
			// when url.Parse normalized away a default port.
			host = "[" + host + "]"
		}
	} else {
		return "", fmt.Errorf("remote URL does not identify a provider host")
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if host == "" || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("remote URL does not identify exactly one owner and repository")
	}
	repository := strings.TrimSuffix(parts[1], ".git")
	if repository == "" {
		return "", fmt.Errorf("remote URL has an empty repository name")
	}
	// GitHub repository identities are case-insensitive. Use one canonical
	// representation before identities are used as map keys or query targets so
	// equivalent push URLs cannot appear to be conflicting repositories.
	return strings.ToLower(host) + "/" + strings.ToLower(parts[0]) + "/" + strings.ToLower(repository), nil
}

func splitSCPRemote(value string) (string, string, bool) {
	if strings.Contains(value, "://") {
		return "", "", false
	}
	hostStart := 0
	if userSeparator := strings.LastIndex(value, "@"); userSeparator >= 0 {
		hostStart = userSeparator + 1
	}
	remaining := value[hostStart:]
	if strings.HasPrefix(remaining, "[") {
		if separator := strings.Index(remaining, "]:"); separator >= 0 {
			return remaining[:separator+1], remaining[separator+2:], true
		}
		return "", "", false
	}
	userHost, repositoryPath, ok := strings.Cut(value, ":")
	if !ok {
		return "", "", false
	}
	if userSeparator := strings.LastIndex(userHost, "@"); userSeparator >= 0 {
		userHost = userHost[userSeparator+1:]
	}
	return userHost, repositoryPath, true
}

func isProviderURLScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "ssh", "http", "https":
		return true
	default:
		return false
	}
}

func isDefaultPort(scheme, port string) bool {
	return (strings.EqualFold(scheme, "ssh") && port == "22") ||
		(strings.EqualFold(scheme, "https") && port == "443") ||
		(strings.EqualFold(scheme, "http") && port == "80")
}

func repositoryNameWithOwner(identity string) string {
	_, nameWithOwner, _ := strings.Cut(identity, "/")
	return nameWithOwner
}

func providerHost(identity string) string {
	host, _, _ := strings.Cut(identity, "/")
	return host
}

func (s *ReviewScope) resolve(ctx context.Context, candidate string) (string, error) {
	if _, err := s.git(ctx, nil, "rev-parse", "--verify", candidate+"^{commit}"); err == nil {
		return candidate, nil
	}
	remotes, err := s.git(ctx, nil, "remote")
	if err != nil {
		return "", err
	}
	var matches []string
	for _, remote := range strings.Fields(remotes) {
		ref := "refs/remotes/" + remote + "/" + candidate
		if _, err := s.git(ctx, nil, "rev-parse", "--verify", ref+"^{commit}"); err == nil {
			matches = append(matches, ref)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("reference is unavailable or ambiguous across %d remote refs", len(matches))
	}
	return matches[0], nil
}

// resolveOpenPRBase prefers a unique remote-tracking ref over a same-named local
// branch, because provider metadata names the remote pull-request target.

func (s *ReviewScope) resolveOpenPRBase(ctx context.Context, candidate, providerCommit, target string) (string, error) {
	matches, err := s.remoteTrackingRefs(ctx, candidate, target)
	if err != nil {
		return "", err
	}
	var base string
	switch len(matches) {
	case 1:
		base = matches[0]
	case 0:
		// A local-only target remains usable only when it is verified against
		// the provider's advertised base commit.
		base, err = s.resolve(ctx, candidate)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("reference is ambiguous across %d remote refs", len(matches))
	}
	actual, err := s.git(ctx, nil, "rev-parse", "--verify", base+"^{commit}")
	if err != nil {
		return "", err
	}
	if actual != providerCommit {
		return "", fmt.Errorf("local base %q is stale (have %s, provider has %s); fetch the target remote or set --review-base", base, actual, providerCommit)
	}
	return base, nil
}

func (s *ReviewScope) remoteTrackingRefs(ctx context.Context, candidate, target string) ([]string, error) {
	remotes, err := s.git(ctx, nil, "remote")
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, remote := range strings.Fields(remotes) {
		if target != "" {
			urls, err := s.git(ctx, nil, "remote", "get-url", "--all", remote)
			if err != nil {
				continue
			}
			foundTarget := false
			for _, remoteURL := range strings.Fields(urls) {
				identity, err := repositoryIdentity(remoteURL)
				if err == nil && identity == target {
					foundTarget = true
					break
				}
			}
			if !foundTarget {
				continue
			}
		}
		ref := "refs/remotes/" + remote + "/" + candidate
		if _, err := s.git(ctx, nil, "rev-parse", "--verify", ref+"^{commit}"); err == nil {
			matches = append(matches, ref)
		}
	}
	return matches, nil
}

func (s *ReviewScope) copyRealIndex(ctx context.Context, target string) error {
	if s.copyIndex != nil {
		return s.copyIndex(ctx, target)
	}
	index, err := s.git(ctx, nil, s.rootGitArgs("rev-parse", "--git-path", "index")...)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(index) {
		index = filepath.Join(s.Root, index)
	}
	data, err := os.ReadFile(index)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o600)
}

// clearHiddenSnapshotIndexFlags makes assume-unchanged entries visible to the
// disposable index without changing the caller's index. git add --sparse then
// includes materialized skip-worktree paths without staging absent paths outside
// the sparse cone.
func (s *ReviewScope) clearHiddenSnapshotIndexFlags(ctx context.Context) error {
	result, err := s.Runner.Run(ctx, runner.Invocation{
		Executable: "git",
		Args:       s.rootGitArgs("-c", disableSplitIndexConfig, "ls-files", "-v", "-z"),
		Env:        s.snapshotEnvironment(),
		UnsetEnv:   gitTransportEnvironmentExcept("GIT_INDEX_FILE"),
	})
	if err != nil {
		return err
	}
	var assumeUnchanged []string
	for _, entry := range strings.Split(result.Stdout, "\x00") {
		marker, path, found := strings.Cut(entry, " ")
		if !found || len(marker) != 1 || path == "" {
			continue
		}
		if unicode.IsLower(rune(marker[0])) {
			assumeUnchanged = append(assumeUnchanged, path)
		}
	}
	return s.updateSnapshotIndexFlags(ctx, "--no-assume-unchanged", assumeUnchanged)
}

func (s *ReviewScope) updateSnapshotIndexFlags(ctx context.Context, flag string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	_, err := s.Runner.Run(ctx, runner.Invocation{
		Executable: "git",
		Args:       s.rootGitArgs("-c", disableSplitIndexConfig, "update-index", flag, "-z", "--stdin"),
		Stdin:      strings.Join(paths, "\x00") + "\x00",
		Env:        s.snapshotEnvironment(),
		UnsetEnv:   gitTransportEnvironmentExcept("GIT_INDEX_FILE"),
	})
	if err != nil {
		return fmt.Errorf("clear %s: %w", flag, err)
	}
	return nil
}

// rootGitArgs makes snapshot preparation independent of the directory from
// which code-converge was launched. Git's --git-path output and worktree/index
// operations are otherwise relative to the runner's current directory.
func (s *ReviewScope) rootGitArgs(args ...string) []string {
	if s.Root == "" {
		return args
	}
	return append([]string{"-C", s.Root}, args...)
}

func (s *ReviewScope) reviewEnvironment() ([]string, error) {
	if s.gitWrapperDir == "" {
		gitExecutable, err := exec.LookPath("git")
		if err != nil && (gitExecutable == "" || !errors.Is(err, exec.ErrDot)) {
			return nil, fmt.Errorf("locate git for scoped review: %w", err)
		}
		gitExecutable, err = filepath.Abs(gitExecutable)
		if err != nil {
			return nil, fmt.Errorf("resolve git for scoped review: %w", err)
		}
		helper, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("locate scoped git helper: %w", err)
		}
		wrapperDir := filepath.Join(s.tempDir, "bin")
		helperDir := filepath.Join(s.tempDir, "git-exec")
		if err := validatePathListEntry(wrapperDir); err != nil {
			return nil, err
		}
		if err := validatePathListEntry(helperDir); err != nil {
			return nil, err
		}
		if err := os.Mkdir(wrapperDir, 0o700); err != nil {
			return nil, fmt.Errorf("create scoped git wrapper: %w", err)
		}
		if err := os.Mkdir(helperDir, 0o700); err != nil {
			return nil, fmt.Errorf("create scoped git helper directory: %w", err)
		}
		wrapper := filepath.Join(wrapperDir, "git")
		// The wrapper is a symlink to this executable, not a script written to the
		// temporary directory. This keeps the helper executable when TMPDIR is on
		// a noexec mount; only the symlink itself lives in the temporary directory.
		if err := os.Symlink(helper, wrapper); err != nil {
			return nil, fmt.Errorf("link scoped git helper: %w", err)
		}
		gitExecPath, err := gitExecutablePath(gitExecutable)
		if err != nil {
			return nil, err
		}
		if err := linkGitHelpers(gitExecPath, helperDir); err != nil {
			return nil, err
		}
		s.gitWrapperDir = wrapperDir
		s.gitHelperDir = helperDir
		configuration, err := json.Marshal(s.scopedGitConfiguration(gitExecutable))
		if err != nil {
			return nil, fmt.Errorf("encode scoped git configuration: %w", err)
		}
		if err := os.WriteFile(filepath.Join(wrapperDir, scopedGitConfigurationFile), configuration, 0o600); err != nil {
			return nil, fmt.Errorf("write scoped git configuration: %w", err)
		}
		if err := os.Chmod(wrapperDir, 0o700); err != nil {
			return nil, err
		}
		if err := os.Chmod(helperDir, 0o700); err != nil {
			return nil, err
		}
		return s.scopedGitEnvironment(), nil
	}
	return s.scopedGitEnvironment(), nil
}

func gitExecutablePath(gitExecutable string) (string, error) {
	command := exec.Command(gitExecutable, "--exec-path")
	command.Env = sanitizedGitProcessEnvironment()
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("locate git exec path for scoped review: %w", err)
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", errors.New("locate git exec path for scoped review: git returned an empty exec path")
	}
	return path, nil
}

// linkGitHelpers makes helperDir a complete GIT_EXEC_PATH. It must remain
// distinct from the PATH directory, which exposes only the scoped git wrapper:
// otherwise a direct git-* command would bypass review-index selection.
func linkGitHelpers(gitExecPath, helperDir string) error {
	entries, err := os.ReadDir(gitExecPath)
	if err != nil {
		return fmt.Errorf("read git exec path for scoped review: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == "git" || entry.Name() == scopedGitConfigurationFile {
			continue
		}
		if err := os.Symlink(filepath.Join(gitExecPath, entry.Name()), filepath.Join(helperDir, entry.Name())); err != nil {
			return fmt.Errorf("link git helper %q for scoped review: %w", entry.Name(), err)
		}
	}
	return nil
}

func validatePathListEntry(path string) error {
	if strings.ContainsRune(path, os.PathListSeparator) {
		return fmt.Errorf("create scoped git wrapper: temporary path %q contains the PATH list separator %q", path, os.PathListSeparator)
	}
	return nil
}

const scopedGitConfigurationFile = ".code-converge-scoped-git.json"

func (s *ReviewScope) scopedGitConfiguration(gitExecutable string) scopedGitConfiguration {
	reviewRoot := s.Root
	if reviewRoot == "" {
		reviewRoot, _ = os.Getwd()
	}
	return scopedGitConfiguration{
		Executable: gitExecutable,
		Root:       reviewRoot,
		Index:      filepath.Join(s.tempDir, "index"),
		WrapperDir: s.gitWrapperDir,
		HelperDir:  s.gitHelperDir,
	}
}

func (s *ReviewScope) scopedGitEnvironment() []string {
	return []string{
		"PATH=" + s.gitWrapperDir + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
}

func (s *ReviewScope) snapshotEnvironment() []string {
	return []string{"GIT_INDEX_FILE=" + filepath.Join(s.tempDir, "index")}
}

// IsScopedGitWrapperInvocation reports whether this process was reached through
// the review-only git symlink. The wrapper's adjacent private configuration
// keeps this transport independent of Codex's child-process environment policy.
func IsScopedGitWrapperInvocation(argv0 string) bool {
	_, ok := scopedGitConfigurationForInvocation(argv0)
	return ok
}

// RunScopedGitWrapper runs Git with the review index only after the requested
// repository has been resolved to the reviewed worktree. It deliberately does
// not export GIT_INDEX_FILE to the Codex shell: absolute Git paths and xcrun
// therefore use their normal repository index.
func RunScopedGitWrapper(args []string) int {
	configuration, ok := scopedGitConfigurationForInvocation(os.Args[0])
	if !ok {
		fmt.Fprintln(os.Stderr, "code-converge scoped git helper: missing review configuration")
		return 125
	}
	if _, _, _, ok := splitGitGlobalOptions(args); !ok {
		// A parse failure must not be treated as a command for another
		// repository. Doing so would run a valid, newly introduced global Git
		// option against the real index and silently omit the review snapshot.
		fmt.Fprintln(os.Stderr, "code-converge scoped git helper: unsupported or malformed Git global option")
		return 125
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "code-converge scoped git helper: determine working directory: %v\n", err)
		return 125
	}
	if !gitCreatesRepository(args, configuration.Executable, workingDirectory) && gitTargetsReviewRoot(args, configuration.Executable, configuration.Root) {
		return runScopedGit(configuration.Executable, args, configuration.Index, configuration.HelperDir)
	}
	return runScopedGit(configuration.Executable, args, "", configuration.HelperDir)
}

func scopedGitConfigurationForInvocation(argv0 string) (scopedGitConfiguration, bool) {
	if filepath.Base(argv0) != "git" {
		return scopedGitConfiguration{}, false
	}
	wrapper := argv0
	if !filepath.IsAbs(wrapper) && !strings.ContainsRune(wrapper, filepath.Separator) {
		var err error
		wrapper, err = exec.LookPath(wrapper)
		if err != nil {
			return scopedGitConfiguration{}, false
		}
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(wrapper), scopedGitConfigurationFile))
	if err != nil {
		return scopedGitConfiguration{}, false
	}
	var configuration scopedGitConfiguration
	if err := json.Unmarshal(data, &configuration); err != nil || configuration.Executable == "" || configuration.Root == "" || configuration.Index == "" || configuration.WrapperDir != filepath.Dir(wrapper) || configuration.HelperDir == "" {
		return scopedGitConfiguration{}, false
	}
	return configuration, true
}

func runScopedGit(gitExecutable string, args []string, indexPath, wrapperDir string) int {
	commandIndex := indexPath
	if indexPath != "" {
		var err error
		commandIndex, err = copyScopedGitIndex(indexPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "code-converge scoped git helper: prepare command index: %v\n", err)
			return 125
		}
		defer os.Remove(commandIndex)
		defer os.Remove(commandIndex + ".lock")
	}
	commandArgs := args
	if indexPath != "" {
		// A private index must never participate in split-index maintenance. Git
		// stores sharedindex.* beside the real repository index even when the
		// alternate index is elsewhere; expiry could otherwise remove the shared
		// index used by ordinary commands.
		commandArgs = privateIndexGitArgs(args)
	}
	command := exec.Command(gitExecutable, commandArgs...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	command.Env = scopedGitProcessEnvironment(commandIndex, wrapperDir)
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return exitError.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "code-converge scoped git helper: %v\n", err)
		return 125
	}
	return 0
}

// copyScopedGitIndex gives one wrapper invocation an isolated writable copy.
// A Git alias, hook or helper can inherit GIT_INDEX_FILE and bypass PATH with
// an absolute Git executable; it can then modify only this command's copy, not
// the stable review snapshot used by later review commands.
func copyScopedGitIndex(indexPath string) (string, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return "", err
	}
	copy, err := os.CreateTemp(filepath.Dir(indexPath), "command-index-")
	if err != nil {
		return "", err
	}
	copyPath := copy.Name()
	if err := copy.Chmod(0o600); err == nil {
		_, err = copy.Write(data)
	}
	if closeErr := copy.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(copyPath)
		return "", err
	}
	return copyPath, nil
}

func scopedGitProcessEnvironment(indexPath, wrapperDir string) []string {
	// The Codex invocation and top-level review Git calls already remove
	// caller-inherited repository selectors. Preserve Git variables established
	// by a running parent Git process for aliases, hooks and worktree helpers;
	// replace only the two transports owned by this wrapper.
	remove := map[string]bool{"GIT_INDEX_FILE": true, "GIT_EXEC_PATH": true}
	var environment []string
	for _, value := range os.Environ() {
		name, _, ok := strings.Cut(value, "=")
		if !ok || remove[name] {
			continue
		}
		environment = append(environment, value)
	}
	if indexPath != "" {
		environment = append(environment, "GIT_INDEX_FILE="+indexPath)
	}
	if wrapperDir != "" {
		environment = append(environment, "GIT_EXEC_PATH="+wrapperDir)
	}
	return environment
}

func gitTargetsReviewRoot(args []string, gitExecutable, reviewRoot string) bool {
	prefix, _, _, ok := splitGitGlobalOptions(args)
	if !ok {
		return false
	}
	command := exec.Command(gitExecutable, append(prefix, "rev-parse", "--show-toplevel")...)
	command.Env = scopedGitProcessEnvironment("", "")
	command.Stderr = io.Discard
	output, err := command.Output()
	if err != nil {
		return false
	}
	targetRoot, err := canonicalPath(strings.TrimSpace(string(output)))
	if err != nil {
		return false
	}
	expectedRoot, err := canonicalPath(reviewRoot)
	return err == nil && targetRoot == expectedRoot
}

func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

func gitCreatesRepository(args []string, gitExecutable, directory string) bool {
	prefix, subcommand, remainder, ok := splitGitGlobalOptions(args)
	if !ok {
		return false
	}
	return gitCommandCreatesRepository(prefix, subcommand, remainder, gitExecutable, directory, map[string]bool{})
}

func gitCommandCreatesRepository(prefix []string, subcommand string, remainder []string, gitExecutable, directory string, visitedAliases map[string]bool) bool {
	if subcommand == "clone" {
		return true
	}
	if subcommand == "worktree" {
		for index := 0; index < len(remainder); index++ {
			argument := remainder[index]
			if argument == "-b" || argument == "-B" || argument == "--orphan" {
				index++
				continue
			}
			if strings.HasPrefix(argument, "-") {
				continue
			}
			return argument == "add"
		}
		return false
	}
	// Git resolves built-ins before aliases. Checking this before consulting
	// alias.<subcommand> ensures a user alias named after a built-in cannot
	// cause the reviewed repository to lose its private snapshot.
	if gitCommandIsBuiltIn(gitExecutable, directory, subcommand) {
		return false
	}
	if gitExecutable == "" || visitedAliases[subcommand] {
		return false
	}
	visitedAliases[subcommand] = true
	expansion, found := gitAliasExpansion(gitExecutable, directory, prefix, subcommand)
	if !found {
		// External git-* commands are arbitrary executables. Without an alias
		// expansion, the wrapper cannot prove that the command stays in the
		// reviewed repository, so it must not inherit the disposable index.
		return true
	}
	if strings.HasPrefix(strings.TrimSpace(expansion), "!") {
		return shellAliasCreatesRepository(expansion)
	}
	expandedArgs, ok := splitGitAliasArguments(expansion)
	if !ok || len(expandedArgs) == 0 {
		// Git may still accept an alias syntax this classifier does not. Exempt it
		// rather than risk initializing another repository with the review index.
		return true
	}
	expandedPrefix, expandedSubcommand, expandedRemainder, ok := splitGitGlobalOptions(expandedArgs)
	if !ok {
		return true
	}
	// A regular alias may select a different repository before dispatching to
	// its command. The outer target check cannot see that expansion, so retain
	// the review index only when the expansion is target-neutral.
	if gitPrefixSelectsRepository(expandedPrefix) {
		return true
	}
	expandedPrefix = append(append([]string{}, prefix...), expandedPrefix...)
	expandedRemainder = append(expandedRemainder, remainder...)
	return gitCommandCreatesRepository(expandedPrefix, expandedSubcommand, expandedRemainder, gitExecutable, directory, visitedAliases)
}

func gitCommandIsBuiltIn(gitExecutable, directory, subcommand string) bool {
	if gitExecutable == "" {
		return false
	}
	command := exec.Command(gitExecutable, "--list-cmds=builtins")
	command.Dir = directory
	command.Env = scopedGitProcessEnvironment("", "")
	output, err := command.Output()
	if err != nil {
		// --list-cmds is unavailable in older Git versions. Preserve command
		// precedence for the long-standing built-ins that matter to scoped
		// review rather than treating their ignored aliases as executable.
		return legacyGitBuiltInCommands[subcommand]
	}
	for _, command := range strings.Fields(string(output)) {
		if command == subcommand {
			return true
		}
	}
	return false
}

var legacyGitBuiltInCommands = map[string]bool{
	"add": true, "am": true, "annotate": true, "apply": true, "archive": true,
	"bisect": true, "blame": true, "branch": true, "bundle": true,
	"cat-file": true, "check-attr": true, "check-ignore": true, "check-ref-format": true,
	"checkout": true, "checkout-index": true, "cherry": true, "cherry-pick": true,
	"clean": true, "clone": true, "commit": true, "commit-tree": true, "config": true,
	"count-objects": true, "describe": true, "diff": true, "diff-files": true,
	"diff-index": true, "diff-tree": true, "fast-export": true, "fast-import": true,
	"fetch": true, "fetch-pack": true, "fmt-merge-msg": true, "for-each-ref": true,
	"format-patch": true, "fsck": true, "gc": true, "get-tar-commit-id": true,
	"grep": true, "hash-object": true, "help": true, "index-pack": true, "init": true,
	"init-db": true, "interpret-trailers": true, "log": true, "ls-files": true,
	"ls-remote": true, "ls-tree": true, "mailinfo": true, "mailsplit": true,
	"merge": true, "merge-base": true, "merge-file": true, "merge-index": true,
	"merge-ours": true, "merge-tree": true, "mktag": true, "mktree": true, "mv": true,
	"name-rev": true, "notes": true, "pack-objects": true, "pack-refs": true,
	"patch-id": true, "prune": true, "pull": true, "push": true, "read-tree": true,
	"receive-pack": true, "reflog": true, "remote": true, "repack": true, "replace": true,
	"request-pull": true, "reset": true, "rev-list": true, "rev-parse": true,
	"revert": true, "rm": true, "send-pack": true, "shortlog": true, "show": true,
	"show-branch": true, "show-index": true, "show-ref": true, "status": true,
	"stripspace": true, "symbolic-ref": true, "tag": true, "unpack-objects": true,
	"update-index": true, "update-ref": true, "update-server-info": true,
	"upload-archive": true, "upload-pack": true, "var": true, "verify-commit": true,
	"verify-pack": true, "verify-tag": true, "version": true, "whatchanged": true,
	"worktree": true, "write-tree": true,
}

func shellAliasCreatesRepository(expansion string) bool {
	command := strings.TrimSpace(strings.TrimPrefix(expansion, "!"))
	// Shell aliases are arbitrary programs. Only a plain Git invocation without
	// shell syntax can be classified safely; everything else must run without
	// the disposable review index.
	if command == "" || strings.ContainsAny(command, "|&;<>($`\\\"'*?[]{}~\r\n") {
		return true
	}
	fields := strings.Fields(command)
	if len(fields) < 2 || filepath.Base(fields[0]) != "git" {
		return true
	}
	prefix, subcommand, _, ok := splitGitGlobalOptions(fields[1:])
	if !ok {
		return true
	}
	// A target-selection option could redirect an absolute Git descendant to
	// another repository. Resolving shell syntax and relative paths here would
	// be unreliable, so withhold the review index instead.
	if gitPrefixSelectsRepository(prefix) {
		return true
	}
	// Any subcommand that can create a repository, mutate it, or dispatch to
	// another alias is conservatively treated as unclassifiable. The reviewed
	// index is retained only for known read-only commands.
	switch subcommand {
	case "status", "diff", "log", "show", "grep", "ls-files", "ls-tree", "rev-parse", "describe", "cat-file", "for-each-ref", "name-rev", "shortlog", "var", "check-ignore", "count-objects", "fsck", "help", "version":
		return false
	default:
		return true
	}
}

func gitPrefixSelectsRepository(prefix []string) bool {
	for _, argument := range prefix {
		if argument == "-C" || strings.HasPrefix(argument, "-C") && len(argument) > 2 || argument == "--git-dir" || argument == "--work-tree" || strings.HasPrefix(argument, "--git-dir=") || strings.HasPrefix(argument, "--work-tree=") {
			return true
		}
	}
	return false
}

func gitAliasExpansion(gitExecutable, directory string, prefix []string, name string) (string, bool) {
	args := append(append([]string{}, prefix...), "config", "--get", "alias."+name)
	command := exec.Command(gitExecutable, args...)
	command.Dir = directory
	command.Env = scopedGitProcessEnvironment("", "")
	output, err := command.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSuffix(string(output), "\n"), true
}

// splitGitAliasArguments implements the quoting Git accepts for regular
// aliases. Shell aliases begin with ! and are handled before this function.
func splitGitAliasArguments(input string) ([]string, bool) {
	var arguments []string
	var argument strings.Builder
	var quote rune
	started := false
	characters := []rune(input)
	for index := 0; index < len(characters); index++ {
		character := characters[index]
		switch {
		case quote != 0:
			if character == quote {
				quote = 0
				continue
			}
			if character == '\\' && quote == '"' {
				if index+1 == len(characters) {
					return nil, false
				}
				index++
				argument.WriteRune(characters[index])
				started = true
				continue
			}
			argument.WriteRune(character)
			started = true
		case character == '\\':
			if index+1 == len(characters) {
				return nil, false
			}
			index++
			argument.WriteRune(characters[index])
			started = true
		case character == '\'' || character == '"':
			quote = character
			started = true
		case character == ' ' || character == '\t' || character == '\n':
			if started {
				arguments = append(arguments, argument.String())
				argument.Reset()
				started = false
			}
		default:
			argument.WriteRune(character)
			started = true
		}
	}
	if quote != 0 {
		return nil, false
	}
	if started {
		arguments = append(arguments, argument.String())
	}
	return arguments, true
}

// splitGitGlobalOptions returns the global prefix, the subcommand and its
// remaining arguments. Unknown or malformed options fail closed so they never
// receive the private index. Git accepts both --namespace=name and
// --namespace name; both forms must be consumed before looking for -C.
func splitGitGlobalOptions(args []string) (prefix []string, subcommand string, remainder []string, ok bool) {
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch argument {
		case "-C", "--git-dir", "--work-tree", "-c", "--config", "--config-env", "--namespace", "--super-prefix", "--attr-source":
			if index+1 == len(args) {
				return nil, "", nil, false
			}
			prefix = append(prefix, argument, args[index+1])
			index++
		case "-p", "-P", "--no-pager", "--paginate", "--bare", "--no-replace-objects", "--no-lazy-fetch", "--no-optional-locks", "--no-advice", "--literal-pathspecs", "--glob-pathspecs", "--noglob-pathspecs", "--icase-pathspecs", "--exec-path", "-v", "--version", "-h", "--help", "--html-path", "--man-path", "--info-path":
			prefix = append(prefix, argument)
		case "--":
			return nil, "", nil, false
		default:
			if strings.HasPrefix(argument, "-C") && len(argument) > 2 || strings.HasPrefix(argument, "-c") && len(argument) > 2 || hasGitOptionValue(argument) {
				prefix = append(prefix, argument)
				continue
			}
			if strings.HasPrefix(argument, "-") {
				return nil, "", nil, false
			}
			return prefix, argument, args[index+1:], true
		}
	}
	return prefix, "", nil, true
}

func hasGitOptionValue(argument string) bool {
	for _, option := range []string{"--git-dir=", "--work-tree=", "--config=", "--config-env=", "--namespace=", "--super-prefix=", "--attr-source=", "--list-cmds=", "--exec-path="} {
		if strings.HasPrefix(argument, option) {
			return true
		}
	}
	return false
}

func (s *ReviewScope) git(ctx context.Context, env []string, args ...string) (string, error) {
	result, err := s.runGit(ctx, env, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (s *ReviewScope) runGit(ctx context.Context, env []string, args ...string) (runner.Result, error) {
	if reviewIndexEnvironment(env) {
		args = append([]string{"-c", disableSplitIndexConfig}, args...)
	}
	result, err := s.Runner.Run(ctx, runner.Invocation{
		Executable: "git",
		Args:       args,
		Env:        env,
		UnsetEnv:   gitTransportEnvironmentExcept(environmentNames(env)...),
	})
	if err != nil {
		return result, err
	}
	return result, nil
}

func environmentNames(environment []string) []string {
	names := make([]string, 0, len(environment))
	for _, value := range environment {
		if name, _, ok := strings.Cut(value, "="); ok {
			names = append(names, name)
		}
	}
	return names
}

func gitTransportEnvironmentExcept(exceptions ...string) []string {
	allowed := make(map[string]bool, len(exceptions))
	for _, name := range exceptions {
		allowed[name] = true
	}
	unset := make([]string, 0, len(gitTransportEnvironment))
	for _, name := range gitTransportEnvironment {
		if !allowed[name] {
			unset = append(unset, name)
		}
	}
	return unset
}

func sanitizedGitProcessEnvironment() []string {
	remove := make(map[string]bool, len(gitTransportEnvironment))
	for _, name := range gitTransportEnvironment {
		remove[name] = true
	}
	var environment []string
	for _, value := range os.Environ() {
		name, _, ok := strings.Cut(value, "=")
		if ok && remove[name] {
			continue
		}
		environment = append(environment, value)
	}
	return environment
}

func reviewIndexEnvironment(environment []string) bool {
	for _, value := range environment {
		if strings.HasPrefix(value, "GIT_INDEX_FILE=") {
			return true
		}
	}
	return false
}

// privateIndexGitArgs places the disabling configuration after caller-provided
// global options, so an explicit -c core.splitIndex=true cannot re-enable
// split-index maintenance for a disposable index.
func privateIndexGitArgs(args []string) []string {
	prefix, subcommand, remainder, ok := splitGitGlobalOptions(args)
	if !ok {
		return append([]string{"-c", disableSplitIndexConfig}, args...)
	}
	result := append([]string{}, prefix...)
	result = append(result, "-c", disableSplitIndexConfig)
	if subcommand != "" {
		result = append(result, subcommand)
	}
	return append(result, remainder...)
}
