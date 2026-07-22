package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dapi/code-converge/internal/runner"
)

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
	tempDir                             string
	copyIndex                           func(context.Context, string) error
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
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(s.tempDir, "index")}
	if err := s.copyRealIndex(ctx, filepath.Join(s.tempDir, "index")); err != nil {
		return ReviewTarget{}, fmt.Errorf("prepare review index: %w", err)
	}
	if _, err := s.git(ctx, env, "add", "-A"); err != nil {
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
		base, err := s.resolveOpenPRBase(ctx, candidate)
		if err != nil {
			return "", "", fmt.Errorf("resolve open PR base %q: %w", candidate, err)
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

func (s *ReviewScope) openPRBase(ctx context.Context, branch string) (string, bool, error) {
	output, err := s.Runner.Run(ctx, runner.Invocation{Executable: "gh", Args: []string{"pr", "list", "--head", branch, "--state", "open", "--json", "baseRefName"}})
	if err != nil {
		return "", false, nil // Provider discovery is optional.
	}
	var prs []struct{ BaseRefName string }
	if err := json.Unmarshal([]byte(output.Stdout), &prs); err != nil {
		return "", false, fmt.Errorf("parse open PR candidates: %w", err)
	}
	switch len(prs) {
	case 0:
		return "", false, nil
	case 1:
		if strings.TrimSpace(prs[0].BaseRefName) == "" {
			return "", false, fmt.Errorf("open PR has an empty base")
		}
		return prs[0].BaseRefName, true, nil
	default:
		return "", false, fmt.Errorf("discover review base: found %d open pull requests; set --review-base", len(prs))
	}
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
func (s *ReviewScope) resolveOpenPRBase(ctx context.Context, candidate string) (string, error) {
	if !strings.Contains(candidate, "/") {
		matches, err := s.remoteTrackingRefs(ctx, candidate)
		if err != nil {
			return "", err
		}
		switch len(matches) {
		case 1:
			return matches[0], nil
		case 0:
			// A local-only target remains usable when no tracking ref exists.
		default:
			return "", fmt.Errorf("reference is ambiguous across %d remote refs", len(matches))
		}
	}
	return s.resolve(ctx, candidate)
}

func (s *ReviewScope) remoteTrackingRefs(ctx context.Context, candidate string) ([]string, error) {
	remotes, err := s.git(ctx, nil, "remote")
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, remote := range strings.Fields(remotes) {
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

func (s *ReviewScope) git(ctx context.Context, env []string, args ...string) (string, error) {
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: args, Env: env})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}
