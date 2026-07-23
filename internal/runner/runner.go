package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Invocation struct {
	Executable string
	Args       []string
	Stdin      string
	Env        []string
}

// StageContext identifies a Codex-backed workflow stage for private
// diagnostics. It is deliberately separate from Invocation so the process
// boundary stays usable for non-workflow commands such as git and gh.
type StageContext struct {
	Stage           string
	ReviewPhase     int
	Cycle           int
	Model           string
	ReasoningEffort string
}

type stageContextKey struct{}

func WithStageContext(ctx context.Context, stage StageContext) context.Context {
	return context.WithValue(ctx, stageContextKey{}, stage)
}

func StageContextFrom(ctx context.Context) (StageContext, bool) {
	stage, ok := ctx.Value(stageContextKey{}).(StageContext)
	return stage, ok && stage.Stage != ""
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Runner interface {
	Run(context.Context, Invocation) (Result, error)
}

type Exec struct {
	Executable string
	Dir        string
}

func (r Exec) Run(ctx context.Context, invocation Invocation) (Result, error) {
	name := r.Executable
	if invocation.Executable != "" {
		name = invocation.Executable
	}
	if name == "" {
		name = "codex"
	}
	cmd := exec.Command(name, invocation.Args...)
	configureProcessGroup(cmd)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), invocation.Env...)
	cmd.Stdin = bytes.NewBufferString(invocation.Stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := ctx.Err(); err != nil {
		return Result{ExitCode: -1}, err
	}
	err := cmd.Start()
	if err != nil {
		result := Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: -1}
		return result, formatRunError(name, err, result.Stderr)
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			terminateProcessGroup(cmd)
		case <-done:
		}
	}()
	err = cmd.Wait()
	close(done)
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result, formatRunError(name, err, result.Stderr)
	}
	return result, nil
}

func formatRunError(name string, err error, stderr string) error {
	detail := strings.TrimSpace(stderr)
	if len(detail) > 8192 {
		detail = detail[:8192] + "…"
	}
	if detail != "" {
		return fmt.Errorf("%s exited unsuccessfully: %w: %s", name, err, detail)
	}
	return fmt.Errorf("%s exited unsuccessfully: %w", name, err)
}
