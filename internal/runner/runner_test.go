package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecCapturesStreamsAndCwd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake")
	content := "#!/bin/sh\npwd\nread value\nprintf '%s' \"$value\"\nprintf 'diagnostic' >&2\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := (Exec{Executable: script, Dir: dir}).Run(context.Background(), Invocation{Stdin: "prompt\n"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, dir) || !strings.HasSuffix(result.Stdout, "prompt") || result.Stderr != "diagnostic" {
		t.Fatalf("result = %#v", result)
	}
}

func TestExecCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := (Exec{Executable: "sh"}).Run(ctx, Invocation{Args: []string{"-c", "sleep 5"}})
	if err == nil {
		t.Fatal("cancelled process succeeded")
	}
}

func TestExecCancellationTerminatesDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process groups are POSIX-only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := (Exec{Executable: "sh"}).Run(ctx, Invocation{Args: []string{"-c", "(sleep 5) & wait"}})
	if err == nil {
		t.Fatal("cancelled process succeeded")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("cancellation took %s; descendant likely survived", elapsed)
	}
}

func TestExecIncludesCapturedDiagnosticOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	_, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{Args: []string{"-c", "printf 'model unavailable' >&2; exit 7"}})
	if err == nil || !strings.Contains(err.Error(), "model unavailable") {
		t.Fatalf("error = %v", err)
	}
}
