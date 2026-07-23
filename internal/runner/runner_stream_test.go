package runner

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOutputQueueBoundsPendingData(t *testing.T) {
	q := newOutputQueue()
	q.add(Output{Source: "stdout", Data: []byte(strings.Repeat("a", maxOutputQueueBytes*3/4))})
	q.add(Output{Source: "stdout", Data: []byte(strings.Repeat("b", maxOutputQueueBytes*3/4))})

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.pendingBytes > maxOutputQueueBytes {
		t.Fatalf("pending bytes = %d, want at most %d", q.pendingBytes, maxOutputQueueBytes)
	}
	if len(q.items) != 1 || string(q.items[0].Data) != strings.Repeat("b", maxOutputQueueBytes*3/4) {
		t.Fatalf("queued items = %d, data length = %d; want only newest chunk", len(q.items), len(q.items[0].Data))
	}
}

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

func TestExecCapturesOutputFromDescendantAfterDirectExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	result, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{
		Args: []string{"-c", "(sleep 0.1; printf late) &"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "late" {
		t.Fatalf("stdout = %q, want late", result.Stdout)
	}
}

func TestExecStreamsLabelBothIndependentSources(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("anonymous-pipe multiplexer is POSIX-only")
	}
	var mu sync.Mutex
	var chunks []Output
	_, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{
		Args: []string{"-c", "printf err >&2; printf out"},
		Output: func(chunk Output) {
			mu.Lock()
			defer mu.Unlock()
			chunks = append(chunks, chunk)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	var seenStdout, seenStderr bool
	for _, chunk := range chunks {
		if chunk.Source == "stdout" && string(chunk.Data) == "out" {
			seenStdout = true
		}
		if chunk.Source == "stderr" && string(chunk.Data) == "err" {
			seenStderr = true
		}
	}
	if !seenStdout || !seenStderr {
		t.Fatalf("chunks = %#v, want labeled stdout and stderr chunks", chunks)
	}
}

func TestExecCaptureIsNotBackpressuredBySlowObserver(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	const size = 256 * 1024
	result, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{
		Args:   []string{"-c", "printf '%*s' 262144 '' | tr ' ' x"},
		Output: func(Output) { time.Sleep(250 * time.Millisecond) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stdout) != size || strings.Trim(result.Stdout, "x") != "" {
		t.Fatalf("captured stdout length=%d, want %d", len(result.Stdout), size)
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
