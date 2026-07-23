package repository

import (
	"context"
	"fmt"
	"strings"

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

func (s Status) status(ctx context.Context) (runner.Result, error) {
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"status", "--porcelain", "--untracked-files=all"}})
	if err != nil {
		return runner.Result{}, err
	}
	return result, nil
}
