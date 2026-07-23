package runner

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestExecStreamsAndCapturesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	var mu sync.Mutex
	var chunks []Output
	result, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{
		Args: []string{"-c", "printf out; printf err >&2"},
		Output: func(chunk Output) {
			mu.Lock()
			defer mu.Unlock()
			chunks = append(chunks, chunk)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "out" || result.Stderr != "err" {
		t.Fatalf("result = %#v", result)
	}
	mu.Lock()
	defer mu.Unlock()
	var stdout, stderr string
	for _, chunk := range chunks {
		switch chunk.Source {
		case "stdout":
			stdout += string(chunk.Data)
		case "stderr":
			stderr += string(chunk.Data)
		default:
			t.Fatalf("unexpected source %q", chunk.Source)
		}
	}
	if stdout != "out" || stderr != "err" {
		t.Fatalf("chunks = %#v", chunks)
	}
}

func TestExecStreamingFailureStillIncludesDiagnostic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	_, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{Args: []string{"-c", "printf diagnostic >&2; exit 1"}, Output: func(Output) {}})
	if err == nil || !strings.Contains(err.Error(), "diagnostic") {
		t.Fatalf("error = %v", err)
	}
	if _, ok := err.(*exec.ExitError); ok {
		t.Fatalf("error should be wrapped with executable context: %v", err)
	}
}
