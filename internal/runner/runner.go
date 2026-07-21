package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Invocation struct {
	Args  []string
	Stdin string
}

type Result struct {
	Stdout string
	Stderr string
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
	if name == "" {
		name = "codex"
	}
	cmd := exec.Command(name, invocation.Args...)
	configureProcessGroup(cmd)
	cmd.Dir = r.Dir
	cmd.Stdin = bytes.NewBufferString(invocation.Stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	err := cmd.Start()
	if err != nil {
		result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
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
