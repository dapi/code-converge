package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dapi/code-converge/internal/runner"
)

// Status reports whether Git sees staged, unstaged, or untracked changes.
type Status struct {
	Runner runner.Runner
}

// Checkpoint is the local commit created for a successful findings-fix stage.
// It is deliberately not pushed; publication remains finalization's job.
type Checkpoint struct {
	Created bool
	Branch  string
	Commit  string
}

// Publication is the deterministic result of making the reviewed revision
// available to GitHub.  The SHA is deliberately retained for CI pinning.
type Publication struct {
	Commit        string
	Push          string
	ChangeRequest string
	URL           string
	Head          string
}

// CIResult is intentionally separate from publication: a deadline is an
// operational outcome, not a failed test run.
type CIResult string

const (
	CISuccess CIResult = "success"
	CIFailed  CIResult = "failed"
	CISkipped CIResult = "skipped"
	CITimeout CIResult = "timeout"
)

func (s Status) HasChanges(ctx context.Context) (bool, error) {
	result, err := s.status(ctx)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(result.Stdout) != "", nil
}

// IsClean reports whether an automatic checkpoint can safely attribute all
// resulting worktree changes to one findings-fix stage.
func (s Status) IsClean(ctx context.Context) (bool, error) {
	result, err := s.status(ctx)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(result.Stdout) == "", nil
}

// Head returns the current commit identity for a fix-stage boundary.
func (s Status) Head(ctx context.Context) (string, error) {
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"rev-parse", "HEAD"}})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// Checkpoint records a commit made during a fix stage and, when allowed,
// creates one for remaining worktree changes. initialHead must be captured
// immediately before the agent starts fixing findings.
func (s Status) Checkpoint(ctx context.Context, initialHead string, canCommit bool) (Checkpoint, error) {
	hasChanges, err := s.HasChanges(ctx)
	if err != nil {
		return Checkpoint{}, err
	}
	if hasChanges && canCommit {
		if _, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"add", "-A"}}); err != nil {
			return Checkpoint{}, fmt.Errorf("stage findings checkpoint: %w", err)
		}
		if _, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"commit", "-m", "chore: checkpoint review fixes"}}); err != nil {
			return Checkpoint{}, fmt.Errorf("commit findings checkpoint: %w", err)
		}
	}
	head, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"rev-parse", "HEAD"}})
	if err != nil {
		return Checkpoint{}, fmt.Errorf("resolve findings-fix head: %w", err)
	}
	if strings.TrimSpace(head.Stdout) == strings.TrimSpace(initialHead) {
		return Checkpoint{}, nil
	}
	branch, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"branch", "--show-current"}})
	if err != nil {
		return Checkpoint{}, fmt.Errorf("resolve checkpoint branch: %w", err)
	}
	commit, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"rev-parse", "--short", "HEAD"}})
	if err != nil {
		return Checkpoint{}, fmt.Errorf("resolve checkpoint commit: %w", err)
	}
	branchName, commitID := strings.TrimSpace(branch.Stdout), strings.TrimSpace(commit.Stdout)
	if branchName == "" || commitID == "" {
		return Checkpoint{}, fmt.Errorf("checkpoint branch and commit must be non-empty")
	}
	return Checkpoint{Created: true, Branch: branchName, Commit: commitID}, nil
}

// Publish commits only changes that appeared in a clean run, pushes through a
// direct refspec (so a local tracking-ref update cannot make publication look
// unsuccessful), then reuses or creates exactly one open pull request.
func (s Status) Publish(ctx context.Context, allowCommit bool) (Publication, error) {
	result := Publication{Commit: "skipped", Push: "skipped", ChangeRequest: "skipped"}
	dirty, err := s.HasChanges(ctx)
	if err != nil {
		return result, fmt.Errorf("inspect publication status: %w", err)
	}
	if dirty {
		if !allowCommit {
			return result, fmt.Errorf("refuse to commit pre-existing worktree changes")
		}
		if _, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"add", "-A"}}); err != nil {
			return result, fmt.Errorf("stage publication commit: %w", err)
		}
		if _, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"commit", "-m", "chore: finalize reviewed changes"}}); err != nil {
			return result, fmt.Errorf("commit reviewed changes: %w", err)
		}
		result.Commit = "success"
	}
	branch, err := s.gitValue(ctx, "branch", "--show-current")
	if err != nil {
		return result, fmt.Errorf("resolve publication branch: %w", err)
	}
	if branch == "" {
		return result, fmt.Errorf("resolve publication branch: detached HEAD")
	}
	remote, err := s.pushRemote(ctx, branch)
	if err != nil {
		return result, err
	}
	if _, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"push", remote, "HEAD:refs/heads/" + branch}}); err != nil {
		return result, fmt.Errorf("push %s/%s: %w", remote, branch, err)
	}
	result.Push = "success"
	result.Head, err = s.gitValue(ctx, "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("resolve published head: %w", err)
	}
	if result.Head == "" {
		return result, fmt.Errorf("resolve published head: empty SHA")
	}
	url, err := s.openPR(ctx, branch)
	if err != nil {
		return result, err
	}
	result.URL, result.ChangeRequest = url, "success"
	return result, nil
}

func (s Status) pushRemote(ctx context.Context, branch string) (string, error) {
	for _, args := range [][]string{{"config", "--get", "branch." + branch + ".pushRemote"}, {"config", "--get", "remote.pushDefault"}} {
		value, err := s.gitValue(ctx, args...)
		if err == nil && value != "" {
			return value, nil
		}
	}
	remotes, err := s.gitValue(ctx, "remote")
	if err != nil {
		return "", fmt.Errorf("resolve push remote: %w", err)
	}
	items := strings.Fields(remotes)
	for _, remote := range items {
		if remote == "origin" {
			return remote, nil
		}
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return "", fmt.Errorf("resolve push remote: ambiguous remotes")
}

func (s Status) gitValue(ctx context.Context, args ...string) (string, error) {
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: args})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

type pullRequest struct {
	URL string `json:"url"`
}

func (s Status) openPR(ctx context.Context, branch string) (string, error) {
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "gh", Args: []string{"pr", "list", "--head", branch, "--state", "open", "--json", "url", "--limit", "2"}})
	if err != nil {
		return "", fmt.Errorf("discover pull request: %w", err)
	}
	var prs []pullRequest
	if err := json.Unmarshal([]byte(result.Stdout), &prs); err != nil {
		return "", fmt.Errorf("parse pull request discovery: %w", err)
	}
	if len(prs) > 1 {
		return "", fmt.Errorf("discover pull request: ambiguous open pull requests")
	}
	if len(prs) == 1 && strings.TrimSpace(prs[0].URL) != "" {
		return prs[0].URL, nil
	}
	// gh pr create writes the created PR URL to stdout; unlike gh pr list it
	// does not provide a JSON output mode. Keep parsing local and reject any
	// unexpected response instead of guessing which PR was created.
	result, err = s.Runner.Run(ctx, runner.Invocation{Executable: "gh", Args: []string{"pr", "create", "--head", branch, "--fill"}})
	if err != nil {
		return "", fmt.Errorf("create pull request: %w", err)
	}
	url := strings.TrimSpace(result.Stdout)
	if url == "" || strings.ContainsAny(url, " \t\r\n") {
		return "", fmt.Errorf("parse created pull request: expected one URL")
	}
	return url, nil
}

type checkRuns struct {
	CheckRuns []checkRun `json:"check_runs"`
}
type checkRun struct {
	Status     string  `json:"status"`
	Conclusion *string `json:"conclusion"`
}

// WaitCI selects GitHub check-runs returned by the exact published SHA. No
// check-runs is a documented skipped outcome. Transient command failures are
// retried inside ctx's deadline; authentication-like failures fail immediately.
func (s Status) WaitCI(ctx context.Context, publication Publication) (CIResult, error) {
	interval := 5 * time.Second
	for {
		if err := ctx.Err(); err != nil {
			if err == context.DeadlineExceeded {
				return CITimeout, nil
			}
			return "", err
		}
		result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "gh", Args: []string{"api", "repos/{owner}/{repo}/commits/" + publication.Head + "/check-runs"}})
		if err != nil {
			if permanentProviderError(err.Error()) {
				return "", fmt.Errorf("query CI checks: %w", err)
			}
			if !wait(ctx, interval) {
				if ctx.Err() == context.DeadlineExceeded {
					return CITimeout, nil
				}
				return "", ctx.Err()
			}
			continue
		}
		var checks checkRuns
		if err := json.Unmarshal([]byte(result.Stdout), &checks); err != nil {
			return "", fmt.Errorf("parse CI checks: %w", err)
		}
		if len(checks.CheckRuns) == 0 {
			return CISkipped, nil
		}
		pending := false
		for _, check := range checks.CheckRuns {
			if check.Status != "completed" {
				pending = true
				continue
			}
			conclusion := ""
			if check.Conclusion != nil {
				conclusion = *check.Conclusion
			}
			switch conclusion {
			case "success", "skipped", "neutral":
			default:
				return CIFailed, nil
			}
		}
		if !pending {
			return CISuccess, nil
		}
		if !wait(ctx, interval) {
			if ctx.Err() == context.DeadlineExceeded {
				return CITimeout, nil
			}
			return "", ctx.Err()
		}
	}
}

func permanentProviderError(message string) bool {
	message = strings.ToLower(message)
	// gh's diagnostic text varies by version and transport. These classes cannot
	// recover through polling, so surface them immediately instead of spending
	// the operator's CI deadline on an impossible retry.
	for _, marker := range []string{
		"authentication", "authorization", "not logged in", "auth login",
		"http 400", "http 401", "http 403", "http 404", "http 422",
		"unsupported protocol", "protocol error",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s Status) status(ctx context.Context) (runner.Result, error) {
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"status", "--porcelain", "--untracked-files=all"}})
	if err != nil {
		return runner.Result{}, err
	}
	return result, nil
}
