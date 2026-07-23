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

// RequireClean prevents an automatic checkpoint from including work that was
// already present before the agent started fixing review findings.
func (s Status) RequireClean(ctx context.Context) error {
	result, err := s.status(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(result.Stdout) != "" {
		return fmt.Errorf("cannot checkpoint findings fixes: worktree has pre-existing changes")
	}
	return nil
}

// Checkpoint commits all changes made by a fix stage. Callers must first use
// RequireClean, which defines the safety boundary for this operation.
func (s Status) Checkpoint(ctx context.Context) (Checkpoint, error) {
	hasChanges, err := s.HasChanges(ctx)
	if err != nil {
		return Checkpoint{}, err
	}
	if !hasChanges {
		return Checkpoint{}, nil
	}
	if _, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"add", "-A"}}); err != nil {
		return Checkpoint{}, fmt.Errorf("stage findings checkpoint: %w", err)
	}
	if _, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"commit", "-m", "chore: checkpoint review fixes"}}); err != nil {
		return Checkpoint{}, fmt.Errorf("commit findings checkpoint: %w", err)
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
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"status", "--porcelain"}})
	if err != nil {
		return runner.Result{}, err
	}
	return result, nil
}
