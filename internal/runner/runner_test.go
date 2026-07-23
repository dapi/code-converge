package runner

import (
	"context"
	"errors"
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

func TestExecPassesInvocationEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	result, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{Args: []string{"-c", "printf %s \"$CODE_CONVERGE_TEST_ENV\""}, Env: []string{"CODE_CONVERGE_TEST_ENV=present"}})
	if err != nil || result.Stdout != "present" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestExecOverridesInheritedEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	t.Setenv("CODE_CONVERGE_TEST_ENV", "inherited")
	result, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{
		Args: []string{"-c", "printf %s \"$CODE_CONVERGE_TEST_ENV\""},
		Env:  []string{"CODE_CONVERGE_TEST_ENV=overridden"},
	})
	if err != nil || result.Stdout != "overridden" {
		t.Fatalf("result=%#v err=%v", result, err)
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

func TestExecContextCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (Exec{Executable: "sh"}).Run(ctx, Invocation{Args: []string{"-c", "echo ok"}})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestExecLongStderrTruncation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	_, err := (Exec{Executable: "sh"}).Run(context.Background(), Invocation{Args: []string{"-c", "printf '%*s' 10000 '' | tr ' ' 'x' >&2; exit 1"}})
	if err == nil || !strings.Contains(err.Error(), "…") || len(err.Error()) < 8200 {
		t.Fatalf("error not truncated: len=%d", len(err.Error()))
	}
}

func TestExecDefaultExecutableName(t *testing.T) {
	_, err := (Exec{}).Run(context.Background(), Invocation{Args: []string{"__nonexistent_subcommand__"}})
	if err == nil || !strings.Contains(err.Error(), "codex") {
		t.Fatalf("error = %v", err)
	}
}
