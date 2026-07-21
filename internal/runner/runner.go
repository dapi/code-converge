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
	cmd := exec.CommandContext(ctx, name, invocation.Args...)
	cmd.Dir = r.Dir
	cmd.Stdin = bytes.NewBufferString(invocation.Stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		detail := strings.TrimSpace(result.Stderr)
		if len(detail) > 8192 {
			detail = detail[:8192] + "…"
		}
		if detail != "" {
			return result, fmt.Errorf("%s exited unsuccessfully: %w: %s", name, err, detail)
		}
		return result, fmt.Errorf("%s exited unsuccessfully: %w", name, err)
	}
	return result, nil
}
