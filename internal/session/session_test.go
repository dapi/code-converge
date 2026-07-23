package session

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dapi/code-converge/internal/runner"
)

type fakeRunner struct {
	result runner.Result
	err    error
	calls  int
}

func (f *fakeRunner) Run(context.Context, runner.Invocation) (runner.Result, error) {
	f.calls++
	return f.result, f.err
}

func TestRecordsRedactedCodexInvocation(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	writer, err := Start(Config{Dir: t.TempDir(), Retention: time.Hour, Now: func() time.Time { return now }})
	if err != nil || writer == nil {
		t.Fatalf("writer=%v err=%v", writer, err)
	}
	fake := &fakeRunner{result: runner.Result{Stdout: "Bearer ghp_secret", Stderr: "token=other", ExitCode: 7}, err: errors.New("codex exited: github_pat_secret")}
	_, err = Wrap(fake, writer).Run(runner.WithStageContext(context.Background(), runner.StageContext{Stage: "review", ReviewPhase: 1, Cycle: 2, Model: "model", ReasoningEffort: "high"}), runner.Invocation{Args: []string{"--api-key=secret", "exec"}, Stdin: "Authorization: Bearer secret", Env: []string{"TOKEN=not-logged"}})
	if err == nil || fake.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, fake.calls)
	}
	data, err := os.ReadFile(filepath.Join(writer.Path(), "0001-invocation.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, secret := range []string{"ghp_secret", "other", "github_pat_secret", "TOKEN=not-logged"} {
		if strings.Contains(text, secret) {
			t.Fatalf("session record leaked %q: %s", secret, text)
		}
	}
	var record invocationRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	if record.Executable != "codex" || record.Stage != "review" || record.ReviewPhase != 1 || record.Cycle != 2 || record.ExitCode != 7 || !strings.Contains(record.Stdin, "[REDACTED]") {
		t.Fatalf("record=%#v", record)
	}
	info, err := os.Stat(filepath.Join(writer.Path(), "0001-invocation.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v err=%v", info.Mode(), err)
	}
}

func TestCleanupIsBoundedAndOptInRecordDirectoriesAreConcurrent(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	old := filepath.Join(root, "session-old")
	if err := os.Mkdir(old, 0o700); err != nil {
		t.Fatal(err)
	}
	oldTime := now.Add(-25 * time.Hour)
	if err := os.Chtimes(old, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "session-link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	one, err := Start(Config{Dir: root, Retention: 24 * time.Hour, Now: func() time.Time { return now }})
	if err != nil || one == nil {
		t.Fatalf("first=%v err=%v", one, err)
	}
	two, err := Start(Config{Dir: root, Retention: 24 * time.Hour, Now: func() time.Time { return now }})
	if err != nil || two == nil || one.Path() == two.Path() {
		t.Fatalf("second=%v err=%v paths=%q/%q", two, err, one.Path(), two.Path())
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("expired log remains: %v", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("cleanup touched symlink target: %v", err)
	}
}

func TestWriteFailureIsDiagnosticOnly(t *testing.T) {
	var diagnostics []string
	writer, err := Start(Config{Dir: t.TempDir(), Retention: time.Hour, Diagnostic: func(message string, err error) { diagnostics = append(diagnostics, message+": "+err.Error()) }})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(writer.Path()); err != nil {
		t.Fatal(err)
	}
	fake := &fakeRunner{result: runner.Result{ExitCode: 0}}
	result, runErr := Wrap(fake, writer).Run(runner.WithStageContext(context.Background(), runner.StageContext{Stage: "review"}), runner.Invocation{})
	if runErr != nil || result.ExitCode != 0 || fake.calls != 1 || len(diagnostics) != 1 || !strings.Contains(diagnostics[0], "write session log") {
		t.Fatalf("result=%#v err=%v calls=%d diagnostics=%q", result, runErr, fake.calls, diagnostics)
	}
}
