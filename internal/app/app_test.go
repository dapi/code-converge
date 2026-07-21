package app

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
)

func testRepo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if output, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	return root, t.TempDir()
}

func TestConfigCommand(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"config", "--max-cycles=4"})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "max-cycles: 4 (cli; built-in: 10)") || !strings.Contains(stdout.String(), "ci-fix-model: agent-default (built-in default)") {
		t.Fatalf("config output:\n%s", stdout.String())
	}
}

func TestInvalidConfigurationEmitsTerminalRun(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"--max-cycles=-1"})
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout.String(), "event=run_started") || !strings.Contains(stdout.String(), "event=run_completed status=operational_failure exit_code=2") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "non-negative integer") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestUnknownCommandFailsWithoutWorkflow(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"unknown"})
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "usage") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
