package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Invocation struct {
	Executable string
	Args       []string
	Stdin      string
	Env        []string
	Output     func(Output)
}

// Output is one observed process chunk. It is delivered while a process is
// running and remains separate from the final captured Result.
type Output struct {
	Source string // stdout or stderr
	Data   []byte
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
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("capture stdout: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, fmt.Errorf("capture stderr: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	err = cmd.Start()
	if err != nil {
		result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
		return result, formatRunError(name, err, result.Stderr)
	}
	var copies sync.WaitGroup
	var outputMu sync.Mutex
	copyStream := func(source string, input io.Reader, capture *bytes.Buffer) {
		defer copies.Done()
		buffer := make([]byte, 32*1024)
		for {
			n, readErr := input.Read(buffer)
			if n > 0 {
				chunk := append([]byte(nil), buffer[:n]...)
				_, _ = capture.Write(chunk)
				if invocation.Output != nil {
					outputMu.Lock()
					invocation.Output(Output{Source: source, Data: chunk})
					outputMu.Unlock()
				}
			}
			if readErr != nil {
				return
			}
		}
	}
	copies.Add(2)
	go copyStream("stdout", stdoutPipe, &stdout)
	go copyStream("stderr", stderrPipe, &stderr)
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
	copies.Wait()
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
