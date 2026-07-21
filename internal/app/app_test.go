package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/dapi/reviewer/internal/runner"
	"github.com/dapi/reviewer/internal/workflow"
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
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"config", "--mode=best", "--max-cycles=4"})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	for _, want := range []string{
		"mode: best (cli; built-in: fast)",
		"max-cycles: 4 (cli; built-in: 10)",
		"review-model: gpt-5.6-sol (best profile; built-in: gpt-5.6-terra)",
		"finalize-reasoning-effort: medium (best profile)",
		"ci-fix-reasoning-effort: high (best profile; built-in: medium)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("missing %q in config output:\n%s", want, stdout.String())
		}
	}
	if t.Failed() {
		t.Fatalf("config output:\n%s", stdout.String())
	}
}

func TestInvalidModeDoesNotInvokeCodex(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{t: t}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), []string{"--mode=unknown"})
	if code != workflow.ExitOperational || len(fake.invocations) != 0 || !strings.Contains(stderr.String(), "mode must be one of") {
		t.Fatalf("code=%d invocations=%d stderr=%q", code, len(fake.invocations), stderr.String())
	}
}

func TestVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr}).Run(context.Background(), []string{"--version"})
	if code != 0 || stdout.String() != "reviewer vdev\n" || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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

func TestInvalidFlagEmitsTerminalRun(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"--unknown"})
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout.String(), "event=run_started") || !strings.Contains(stdout.String(), "event=run_completed status=operational_failure exit_code=2") {
		t.Fatalf("stdout:\n%s", stdout.String())
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

type appFakeRunner struct {
	t           *testing.T
	invocations []runner.Invocation
	review      runner.Result
	finalizeMsg string
	err         error
}

func (f *appFakeRunner) Run(_ context.Context, invocation runner.Invocation) (runner.Result, error) {
	f.invocations = append(f.invocations, invocation)
	for i, arg := range invocation.Args {
		if arg == "--output-last-message" && i+1 < len(invocation.Args) && f.err == nil {
			if err := os.WriteFile(invocation.Args[i+1], []byte(f.finalizeMsg), 0o600); err != nil {
				f.t.Fatalf("write finalize message: %v", err)
			}
		}
	}
	if len(f.invocations) == 1 {
		return f.review, f.err
	}
	return runner.Result{}, f.err
}

func TestNilStreamsAndCwdDoNotPanic(t *testing.T) {
	root, home := testRepo(t)
	fake := &appFakeRunner{t: t, review: runner.Result{Stdout: "No findings.\n"}, finalizeMsg: `{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`}
	code := (App{Cwd: root, Home: home, Runner: fake}).Run(context.Background(), nil)
	if code != workflow.ExitSuccess {
		t.Fatalf("code=%d", code)
	}
}

func TestConfigCommandInvalidFlagDoesNotEmitRunEvents(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"config", "--unknown-flag"})
	if code != workflow.ExitOperational || stdout.Len() != 0 || !strings.Contains(stderr.String(), "flag provided") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunnerPassedFromAppIsUsed(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{t: t, err: errors.New("runner reached")}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), nil)
	if code != workflow.ExitOperational {
		t.Fatalf("code=%d", code)
	}
	if len(fake.invocations) == 0 {
		t.Fatal("app did not use injected runner")
	}
}

func TestAppWorkflowSuccessWithFakeRunner(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{
		t:           t,
		review:      runner.Result{Stdout: "No findings.\n"},
		finalizeMsg: `{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`,
	}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), nil)
	if code != workflow.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "event=run_completed status=success exit_code=0") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if len(fake.invocations) < 2 {
		t.Fatalf("expected review and finalize invocations, got %d", len(fake.invocations))
	}
}
