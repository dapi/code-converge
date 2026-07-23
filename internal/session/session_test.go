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
	fake := &fakeRunner{result: runner.Result{Stdout: `Bearer ghp_secret {"token":"json_secret"}`, Stderr: "token=other", ExitCode: 7}, err: errors.New("codex exited: github_pat_secret")}
	_, err = Wrap(fake, writer).Run(runner.WithStageContext(context.Background(), runner.StageContext{Stage: "review", ReviewPhase: 1, Cycle: 2, Model: "model", ReasoningEffort: "high"}), runner.Invocation{Args: []string{"--api-key=inline_secret", "--client-secret", "following_secret", "exec"}, Stdin: "Authorization: Bearer header_secret", Env: []string{"TOKEN=not-logged"}})
	if err == nil || fake.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, fake.calls)
	}
	data, err := os.ReadFile(filepath.Join(writer.Path(), "0001-invocation.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, secret := range []string{"ghp_secret", "json_secret", "other", "github_pat_secret", "inline_secret", "following_secret", "header_secret", "TOKEN=not-logged"} {
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

func TestRedactRemovesCompleteAuthorizationAndCookieHeaders(t *testing.T) {
	value := "Authorization: Basic dXNlcjpwYXNz\nCookie: SID=secret; CSRF=other"
	redacted := redact(value)
	for _, secret := range []string{"dXNlcjpwYXNz", "SID=secret", "CSRF=other"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("redaction leaked %q: %q", secret, redacted)
		}
	}
	if redacted != "Authorization: [REDACTED]\nCookie: [REDACTED]" {
		t.Fatalf("redacted=%q", redacted)
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
	oldRecord, err := json.Marshal(sessionRecord{StartedAt: oldTime.Add(-time.Hour).Format(time.RFC3339Nano), CompletedAt: oldTime.Format(time.RFC3339Nano)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "session.json"), oldRecord, 0o600); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(root, "session-active")
	if err := os.Mkdir(active, 0o700); err != nil {
		t.Fatal(err)
	}
	activeRecord, err := json.Marshal(sessionRecord{StartedAt: oldTime.Format(time.RFC3339Nano)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(active, "session.json"), activeRecord, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(active, oldTime, oldTime); err != nil {
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
	if _, err := os.Stat(active); err != nil {
		t.Fatalf("cleanup removed active session: %v", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("cleanup touched symlink target: %v", err)
	}
}

func TestCleanupRetainsSessionWithSymlinkedMetadata(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	sessionDir := filepath.Join(root, "session-symlinked-metadata")
	if err := os.Mkdir(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	outsideRecord := filepath.Join(t.TempDir(), "session.json")
	old := now.Add(-25 * time.Hour)
	data, err := json.Marshal(sessionRecord{StartedAt: old.Add(-time.Hour).Format(time.RFC3339Nano), CompletedAt: old.Format(time.RFC3339Nano)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsideRecord, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideRecord, filepath.Join(sessionDir, "session.json")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := cleanup(root, 24*time.Hour, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(sessionDir); err != nil {
		t.Fatalf("cleanup removed session with symlinked metadata: %v", err)
	}
}

func TestStartReportsRootPermissionFailureWithoutDiscardingWriter(t *testing.T) {
	originalChmod := chmod
	chmod = func(string, os.FileMode) error { return errors.New("chmod denied") }
	t.Cleanup(func() { chmod = originalChmod })
	writer, err := Start(Config{Dir: t.TempDir(), Retention: time.Hour})
	if writer == nil || err == nil || !strings.Contains(err.Error(), "set session log directory permissions: chmod denied") {
		t.Fatalf("writer=%v err=%v", writer, err)
	}
}

func TestStartRemovesSessionDirectoryWhenInitialRecordWriteFails(t *testing.T) {
	originalCreateTemp := createTemp
	createTemp = func(string, string) (*os.File, error) { return nil, errors.New("create denied") }
	t.Cleanup(func() { createTemp = originalCreateTemp })
	root := t.TempDir()
	writer, err := Start(Config{Dir: root, Retention: time.Hour})
	if writer != nil || err == nil || !strings.Contains(err.Error(), "create session record: create denied") {
		t.Fatalf("writer=%v err=%v", writer, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "session-") {
			t.Fatalf("initialization failure left session directory %q", entry.Name())
		}
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
