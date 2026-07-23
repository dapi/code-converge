// Package session persists private, best-effort diagnostic records for Codex
// workflow invocations. It deliberately has no knowledge of review results or
// workflow exit policy.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dapi/code-converge/internal/runner"
)

type Config struct {
	Dir        string
	Retention  time.Duration
	Now        func() time.Time
	Diagnostic func(string, error)
}

type Writer struct {
	dir        string
	sessionDir string
	startedAt  string
	now        func() time.Time
	diagnostic func(string, error)
	mu         sync.Mutex
	sequence   int
}

type sessionRecord struct {
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type invocationRecord struct {
	Sequence        int      `json:"sequence"`
	Executable      string   `json:"executable"`
	Arguments       []string `json:"arguments"`
	Stdin           string   `json:"stdin,omitempty"`
	Stdout          string   `json:"stdout"`
	Stderr          string   `json:"stderr"`
	ExitCode        int      `json:"exit_code"`
	Error           string   `json:"error,omitempty"`
	Stage           string   `json:"stage"`
	ReviewPhase     int      `json:"review_phase,omitempty"`
	Cycle           int      `json:"cycle,omitempty"`
	Model           string   `json:"model"`
	ReasoningEffort string   `json:"reasoning_effort"`
	StartedAt       string   `json:"started_at"`
	CompletedAt     string   `json:"completed_at"`
	DurationMS      int64    `json:"duration_ms"`
}

func Start(cfg Config) (*Writer, error) {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if strings.TrimSpace(cfg.Dir) == "" {
		return nil, fmt.Errorf("session log directory is empty")
	}
	if err := os.MkdirAll(cfg.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("create session log directory: %w", err)
	}
	_ = os.Chmod(cfg.Dir, 0o700)
	cleanupErr := cleanup(cfg.Dir, cfg.Retention, cfg.Now())

	random, err := randomSuffix()
	if err != nil {
		return nil, fmt.Errorf("generate session log suffix: %w", err)
	}
	sessionDir := filepath.Join(cfg.Dir, fmt.Sprintf("session-%d-%d-%s", cfg.Now().UTC().UnixNano(), os.Getpid(), random))
	if err := os.Mkdir(sessionDir, 0o700); err != nil {
		return nil, fmt.Errorf("create session log: %w", err)
	}
	startedAt := cfg.Now().UTC().Format(time.RFC3339Nano)
	writer := &Writer{dir: cfg.Dir, sessionDir: sessionDir, startedAt: startedAt, now: cfg.Now, diagnostic: cfg.Diagnostic}
	if err := writer.writeJSON("session.json", sessionRecord{StartedAt: startedAt}); err != nil {
		return nil, err
	}
	return writer, cleanupErr
}

func (w *Writer) Path() string { return w.sessionDir }

func (w *Writer) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.writeJSON("session.json", sessionRecord{StartedAt: w.startedAt, CompletedAt: w.now().UTC().Format(time.RFC3339Nano)}); err != nil {
		w.warn("write session log", err)
	}
}

func Wrap(next runner.Runner, writer *Writer) runner.Runner {
	return loggingRunner{next: next, writer: writer}
}

type loggingRunner struct {
	next   runner.Runner
	writer *Writer
}

func (r loggingRunner) Run(ctx context.Context, invocation runner.Invocation) (runner.Result, error) {
	stage, ok := runner.StageContextFrom(ctx)
	if !ok || (invocation.Executable != "" && invocation.Executable != "codex") {
		return r.next.Run(ctx, invocation)
	}
	started := r.writer.now()
	result, err := r.next.Run(ctx, invocation)
	completed := r.writer.now()
	if recordErr := r.writer.Record(stage, invocation, result, err, started, completed); recordErr != nil {
		r.writer.warn("write session log", recordErr)
	}
	return result, err
}

func (w *Writer) Record(stage runner.StageContext, invocation runner.Invocation, result runner.Result, runErr error, started, completed time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sequence++
	executable := invocation.Executable
	if executable == "" {
		executable = "codex"
	}
	record := invocationRecord{
		Sequence: w.sequence, Executable: redact(executable), Arguments: redactAll(invocation.Args), Stdin: redact(invocation.Stdin),
		Stdout: redact(result.Stdout), Stderr: redact(result.Stderr), ExitCode: result.ExitCode,
		Stage: stage.Stage, ReviewPhase: stage.ReviewPhase, Cycle: stage.Cycle, Model: redact(stage.Model), ReasoningEffort: redact(stage.ReasoningEffort),
		StartedAt: started.UTC().Format(time.RFC3339Nano), CompletedAt: completed.UTC().Format(time.RFC3339Nano), DurationMS: completed.Sub(started).Milliseconds(),
	}
	if runErr != nil {
		record.Error = redact(runErr.Error())
	}
	return w.writeJSON(fmt.Sprintf("%04d-invocation.json", record.Sequence), record)
}

func (w *Writer) writeJSON(name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session record: %w", err)
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(w.sessionDir, ".session-log-*")
	if err != nil {
		return fmt.Errorf("create session record: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set session record permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write session record: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close session record: %w", err)
	}
	if err := os.Rename(temporaryName, filepath.Join(w.sessionDir, name)); err != nil {
		return fmt.Errorf("publish session record: %w", err)
	}
	return nil
}

func (w *Writer) warn(message string, err error) {
	if w.diagnostic != nil {
		w.diagnostic(message, err)
	}
}

func cleanup(root string, retention time.Duration, now time.Time) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("list session logs: %w", err)
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "session-") || entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect session log: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !info.ModTime().Before(now.Add(-retention)) {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove expired session log: %w", err)
		}
	}
	return nil
}

func randomSuffix() (string, error) {
	data := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

var (
	keySecret   = regexp.MustCompile(`(?i)((?:token|secret|password|api[-_]?key|authorization|cookie)\s*(?:[:=]\s*|\s+))([^\s,;]+)`)
	bearer      = regexp.MustCompile(`(?i)\bbearer\s+[^\s,;]+`)
	githubToken = regexp.MustCompile(`(?i)\b(?:ghp|gho|ghs|ghr)_[a-z0-9_]+\b|\bgithub_pat_[a-z0-9_]+\b`)
)

func redact(value string) string {
	value = keySecret.ReplaceAllString(value, "$1[REDACTED]")
	value = bearer.ReplaceAllString(value, "Bearer [REDACTED]")
	return githubToken.ReplaceAllString(value, "[REDACTED]")
}

func redactAll(values []string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = redact(value)
	}
	return result
}
