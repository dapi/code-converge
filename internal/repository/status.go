package repository

import (
	"context"
	"strings"

	"github.com/dapi/code-converge/internal/runner"
)

// Status reports whether Git sees staged, unstaged, or untracked changes.
type Status struct {
	Runner runner.Runner
}

func (s Status) HasChanges(ctx context.Context) (bool, error) {
	result, err := s.Runner.Run(ctx, runner.Invocation{Executable: "git", Args: []string{"status", "--porcelain"}})
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(result.Stdout) != "", nil
}
