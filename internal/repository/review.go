package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dapi/code-converge/internal/runner"
)

var errNoProviderRemote = errors.New("no provider remote is configured")

// ReviewTarget is the resolved base and private index used for one review.
type ReviewTarget struct {
	Base       string
	BaseCommit string
	MergeBase  string
	Source     string
	Env        []string
}

// ReviewScope discovers a base once and refreshes a private index before each review.
// It never changes the caller's real Git index or worktree.
type ReviewScope struct {
	Runner runner.Runner
	Base   string
	Root   string

	base, baseCommit, mergeBase, source string
	tempDir, gitWrapperDir              string
	copyIndex                           func(context.Context, string) error
}

type scopedGitConfiguration struct {
	Executable string `json:"executable"`
	Root       string `json:"root"`
	Index      string `json:"index"`
	WrapperDir string `json:"wrapper_dir"`
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
		mergeBase, err := s.git(ctx, nil, "merge-base", "HEAD", base)
		if err != nil {
			return ReviewTarget{}, fmt.Errorf("find merge-base for %q: %w", base, err)
		}
		s.base, s.baseCommit, s.mergeBase, s.source = base, baseCommit, mergeBase, source
	}
	if s.tempDir == "" {
		dir, err := os.MkdirTemp("", "code-converge-review-index-")
		if err != nil {
			return ReviewTarget{}, fmt.Errorf("create review index: %w", err)
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
	if _, err := s.git(ctx, s.snapshotEnvironment(), "add", "-A"); err != nil {
		return ReviewTarget{}, fmt.Errorf("snapshot worktree for review: %w", err)
	}
	return ReviewTarget{Base: s.base, BaseCommit: s.baseCommit, MergeBase: s.mergeBase, Source: s.source, Env: env}, nil
}

func (s *ReviewScope) Close() error {
	if s.tempDir == "" {
		return nil
	}
	err := os.RemoveAll(s.tempDir)
	s.tempDir = ""
	s.gitWrapperDir = ""
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
		output, err := s.Runner.Run(ctx, runner.Invocation{Executable: "gh", Args: []string{"pr", "list", "--repo", target, "--head", branch, "--state", "open", "--json", "baseRefName,baseRefOid,headRefName,headRepository"}})
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
	configuredRemote := false
	for _, key := range []string{"branch." + branch + ".pushRemote", "remote.pushDefault", "branch." + branch + ".remote"} {
		if candidate, err := s.git(ctx, nil, "config", "--get", key); err == nil && candidate != "" {
			remote = candidate
			configuredRemote = true
			break
		}
	}
	if remote == "." {
		return "", errNoProviderRemote
	}
	if remote == "" {
		remote = "origin"
	}
	remoteURLs, err := s.git(ctx, nil, "remote", "get-url", "--push", "--all", remote)
	if err != nil {
		if !configuredRemote {
			return "", errNoProviderRemote
		}
		return "", fmt.Errorf("read push URL for remote %q: %w", remote, err)
	}
	identities := make(map[string]struct{})
	for _, remoteURL := range strings.Fields(remoteURLs) {
		identity, err := repositoryIdentity(remoteURL)
		if err != nil {
			continue // A local or unsupported-provider remote leaves local base discovery available.
		}
		identities[identity] = struct{}{}
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

func repositoryIdentity(remoteURL string) (string, error) {
	value := strings.TrimSuffix(strings.TrimSpace(remoteURL), "/")
	var host string
	var path string
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" {
		host, path = parsed.Hostname(), parsed.Path
	} else if userHost, repositoryPath, ok := strings.Cut(value, ":"); ok {
		host = strings.TrimPrefix(userHost, "git@")
		path = repositoryPath
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
	if strings.Contains(candidate, "/") {
		return "", fmt.Errorf("reference is unavailable locally")
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
	index, err := s.git(ctx, nil, "rev-parse", "--git-path", "index")
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

func (s *ReviewScope) reviewEnvironment() ([]string, error) {
	if s.gitWrapperDir == "" {
		gitExecutable, err := exec.LookPath("git")
		if err != nil {
			return nil, fmt.Errorf("locate git for scoped review: %w", err)
		}
		helper, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("locate scoped git helper: %w", err)
		}
		wrapperDir := filepath.Join(s.tempDir, "bin")
		if err := os.Mkdir(wrapperDir, 0o700); err != nil {
			return nil, fmt.Errorf("create scoped git wrapper: %w", err)
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
		if err := linkGitHelpers(gitExecPath, wrapperDir); err != nil {
			return nil, err
		}
		s.gitWrapperDir = wrapperDir
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
		return s.scopedGitEnvironment(), nil
	}
	return s.scopedGitEnvironment(), nil
}

func gitExecutablePath(gitExecutable string) (string, error) {
	output, err := exec.Command(gitExecutable, "--exec-path").Output()
	if err != nil {
		return "", fmt.Errorf("locate git exec path for scoped review: %w", err)
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", errors.New("locate git exec path for scoped review: git returned an empty exec path")
	}
	return path, nil
}

// linkGitHelpers makes the scoped directory a complete GIT_EXEC_PATH. The
// wrapper sets that value only inside the real Git child process, after it has
// loaded its sidecar configuration; Codex never has to transport it through a
// shell-environment policy.
func linkGitHelpers(gitExecPath, wrapperDir string) error {
	entries, err := os.ReadDir(gitExecPath)
	if err != nil {
		return fmt.Errorf("read git exec path for scoped review: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == "git" {
			continue
		}
		if err := os.Symlink(filepath.Join(gitExecPath, entry.Name()), filepath.Join(wrapperDir, entry.Name())); err != nil {
			return fmt.Errorf("link git helper %q for scoped review: %w", entry.Name(), err)
		}
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
		return runScopedGit(configuration.Executable, args, configuration.Index, configuration.WrapperDir)
	}
	return runScopedGit(configuration.Executable, args, "", configuration.WrapperDir)
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
	if err := json.Unmarshal(data, &configuration); err != nil || configuration.Executable == "" || configuration.Root == "" || configuration.Index == "" || configuration.WrapperDir != filepath.Dir(wrapper) {
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
	command := exec.Command(gitExecutable, args...)
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
	if gitExecutable == "" || visitedAliases[subcommand] {
		return false
	}
	visitedAliases[subcommand] = true
	expansion, found := gitAliasExpansion(gitExecutable, directory, prefix, subcommand)
	if !found {
		return false
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
	expandedArgs = append(append([]string{}, prefix...), expandedArgs...)
	expandedArgs = append(expandedArgs, remainder...)
	expandedPrefix, expandedSubcommand, expandedRemainder, ok := splitGitGlobalOptions(expandedArgs)
	if !ok {
		return true
	}
	return gitCommandCreatesRepository(expandedPrefix, expandedSubcommand, expandedRemainder, gitExecutable, directory, visitedAliases)
}

func shellAliasCreatesRepository(expansion string) bool {
	fields := strings.Fields(expansion)
	for index, field := range fields {
		if field != "git" || index+1 == len(fields) {
			continue
		}
		if fields[index+1] == "clone" {
			return true
		}
		if fields[index+1] == "worktree" && index+2 < len(fields) && fields[index+2] == "add" {
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
	for _, character := range input {
		switch {
		case quote != 0:
			if character == quote {
				quote = 0
				continue
			}
			argument.WriteRune(character)
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
		case "-C", "--git-dir", "--work-tree", "-c", "--config", "--config-env", "--namespace", "--super-prefix":
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
	for _, option := range []string{"--git-dir=", "--work-tree=", "--config=", "--config-env=", "--exec-path=", "--namespace=", "--super-prefix="} {
		if strings.HasPrefix(argument, option) {
			return true
		}
	}
	return false
}

func (s *ReviewScope) git(ctx context.Context, env []string, args ...string) (string, error) {
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: args, Env: env})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}
