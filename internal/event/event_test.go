package event

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEmit(t *testing.T) {
	var out bytes.Buffer
	logger := Logger{Out: &out, Now: func() time.Time { return time.Date(2026, 7, 21, 10, 4, 5, 0, time.FixedZone("x", 3600)) }}
	if err := logger.Emit("stage_started", F("stage", "review"), F("model", "gpt-test"), F("reasoning_effort", "medium"), F("cycle", "1")); err != nil {
		t.Fatal(err)
	}
	want := "ts=2026-07-21T09:04:05Z event=stage_started stage=review model=gpt-test reasoning_effort=medium cycle=1\n"
	if out.String() != want {
		t.Fatalf("record = %q, want %q", out.String(), want)
	}
	for _, field := range []Field{F("Bad", "x"), F("ok", "has space"), F("ok", "a=b"), F("ok", "line\nbreak")} {
		if err := logger.Emit("test", field); err == nil {
			t.Errorf("invalid field accepted: %#v", field)
		}
	}
	if strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("invalid records were written: %q", out.String())
	}
}

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		value time.Duration
		want  string
	}{
		{-time.Second, "0s"}, {700 * time.Millisecond, "0.7s"}, {12550 * time.Millisecond, "12.6s"},
		{42 * time.Second, "42s"}, {87 * time.Second, "1m 27s"}, {time.Hour + 2*time.Minute + 5*time.Second, "1h 2m 5s"},
	}
	for _, test := range tests {
		if got := HumanDuration(test.value); got != test.want {
			t.Errorf("HumanDuration(%s) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestHumanEventCatalog(t *testing.T) {
	tests := []struct {
		name   string
		event  string
		fields []Field
		want   string
	}{
		{"run started omitted", "run_started", nil, ""},
		{"review start", "stage_started", []Field{F("stage", "review"), F("review_phase", "1"), F("cycle", "1")}, "Review attempt 1 started\n"},
		{"review after ci", "stage_started", []Field{F("stage", "review"), F("review_phase", "2"), F("cycle", "1")}, "Review attempt 1 started after CI fix 1\n"},
		{"review findings", "review_completed", []Field{F("stage", "review"), F("review_phase", "1"), F("cycle", "2"), F("status", "findings"), F("findings_total", "3"), F("findings_critical", "0"), F("findings_high", "1"), F("findings_medium", "2"), F("findings_low", "0"), F("findings_unknown", "0"), F("duration_ms", "133000")}, "Review attempt 2: 3 findings — 1 high, 2 medium (2m 13s)\n"},
		{"review clean", "review_completed", []Field{F("stage", "review"), F("cycle", "3"), F("status", "clean"), F("duration_ms", "87000")}, "Review attempt 3: clean (1m 27s)\n"},
		{"review failed", "review_completed", []Field{F("stage", "review"), F("cycle", "1"), F("status", "failed"), F("duration_ms", "12500")}, "Review attempt 1 failed (12.5s)\n"},
		{"fix start", "stage_started", []Field{F("stage", "fix-findings"), F("cycle", "2")}, "Fixing findings from review attempt 2...\n"},
		{"fix done", "stage_completed", []Field{F("stage", "fix-findings"), F("status", "success"), F("duration_ms", "263000")}, "Findings fixed (4m 23s)\n"},
		{"fix failed", "stage_completed", []Field{F("stage", "fix-findings"), F("status", "failed"), F("duration_ms", "1000")}, "Fixing findings failed (1s)\n"},
		{"finalize start", "stage_started", []Field{F("stage", "finalize")}, "Finalizing...\n"},
		{"step", "step_completed", []Field{F("stage", "finalize"), F("step", "change_request"), F("status", "skipped")}, "  Change request: not needed\n"},
		{"finalize success", "stage_completed", []Field{F("stage", "finalize"), F("status", "success"), F("verdict", "SUCCESS"), F("duration_ms", "42000")}, "Finalized successfully (42s)\n"},
		{"finalize ci", "stage_completed", []Field{F("stage", "finalize"), F("status", "success"), F("verdict", "CI_FAILED"), F("duration_ms", "42000")}, "Finalized, but CI is failing (42s)\n"},
		{"finalize failed", "stage_completed", []Field{F("stage", "finalize"), F("status", "failed"), F("duration_ms", "42000")}, "Finalization failed (42s)\n"},
		{"ci start", "stage_started", []Field{F("stage", "fix-ci")}, "Fixing CI...\n"},
		{"ci done", "stage_completed", []Field{F("stage", "fix-ci"), F("status", "success"), F("duration_ms", "68000")}, "CI fixed (1m 8s)\n"},
		{"done", "run_completed", []Field{F("status", "success"), F("exit_code", "0"), F("total_duration_ms", "525000")}, "Done (8m 45s)\n"},
		{"findings remain", "run_completed", []Field{F("status", "findings_remaining"), F("exit_code", "1"), F("total_duration_ms", "525000")}, "Stopped: review findings remain (8m 45s, exit 1)\n"},
		{"operational", "run_completed", []Field{F("status", "operational_failure"), F("exit_code", "2"), F("total_duration_ms", "525000")}, "Failed due to an operational error (8m 45s, exit 2)\n"},
		{"ci failure", "run_completed", []Field{F("status", "ci_failure"), F("exit_code", "3"), F("total_duration_ms", "525000")}, "Stopped: CI is still failing (8m 45s, exit 3)\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			logger := Logger{Out: &out, Format: "human"}
			if err := logger.Emit(test.event, test.fields...); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
	}
}

func TestHumanFindingsRequiresNonZeroSeverity(t *testing.T) {
	logger := Logger{Out: ioDiscard{}, Format: "human"}
	err := logger.Emit("review_completed", F("stage", "review"), F("cycle", "1"), F("status", "findings"), F("findings_total", "0"), F("findings_high", "0"), F("duration_ms", "1"))
	if err == nil || !strings.Contains(err.Error(), "no non-zero") {
		t.Fatalf("error = %v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

type signalWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	writes chan string
	fail   bool
}

func (w *signalWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case w.writes <- string(p):
	default:
	}
	if w.fail {
		return 0, errors.New("writer failed")
	}
	_, _ = w.buf.Write(p)
	return len(p), nil
}

func (w *signalWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func TestHeartbeatIsNewlineSafeAndStops(t *testing.T) {
	ticks := make(chan time.Time, 2)
	writer := &signalWriter{writes: make(chan string, 4)}
	start := time.Unix(0, 0)
	logger := Logger{Out: writer, Format: "human", Heartbeat: 30 * time.Second, Tick: func(time.Duration) (<-chan time.Time, func()) { return ticks, func() {} }}
	ctx, cancel := context.WithCancel(context.Background())
	live := logger.StartLiveness(ctx, "review", start, cancel)
	ticks <- start.Add(30 * time.Second)
	got := <-writer.writes
	if got != "Review still running (30s)\n" || strings.Contains(got, "\x1b") {
		t.Fatalf("heartbeat = %q", got)
	}
	if err := live.Stop(); err != nil {
		t.Fatal(err)
	}
	ticks <- start.Add(time.Minute)
	select {
	case late := <-writer.writes:
		t.Fatalf("late write after stop: %q", late)
	default:
	}
}

func TestTransientClearedBeforePermanentAndDiagnostic(t *testing.T) {
	ticks := make(chan time.Time)
	writer := &signalWriter{writes: make(chan string, 8)}
	var stderr bytes.Buffer
	start := time.Unix(0, 0)
	logger := Logger{Out: writer, Err: &stderr, Format: "human", Interactive: true, ColorDepth: 0, Now: func() time.Time { return start }, Tick: func(time.Duration) (<-chan time.Time, func()) { return ticks, func() {} }}
	ctx, cancel := context.WithCancel(context.Background())
	live := logger.StartLiveness(ctx, "review", start, cancel)
	initial := <-writer.writes
	if initial != "\r\x1b[2KReviewing... 0s" {
		t.Fatalf("transient = %q", initial)
	}
	if err := live.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := logger.Emit("review_completed", F("stage", "review"), F("cycle", "1"), F("status", "clean"), F("duration_ms", "1000")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(writer.String(), "\r\x1b[2KReview attempt 1: clean (1s)\n") {
		t.Fatalf("output was not cleared: %q", writer.String())
	}
	logger.Diagnostic("review failed", errors.New("boom"))
	if !strings.Contains(stderr.String(), "code-converge: review failed: boom") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLivenessWriterFailureCancelsStage(t *testing.T) {
	writer := &signalWriter{writes: make(chan string, 1), fail: true}
	start := time.Unix(0, 0)
	logger := Logger{Out: writer, Format: "human", Interactive: true, Now: func() time.Time { return start }}
	ctx, cancelContext := context.WithCancel(context.Background())
	cancelled := make(chan struct{})
	var once sync.Once
	live := logger.StartLiveness(ctx, "review", start, func() { once.Do(func() { close(cancelled) }); cancelContext() })
	<-writer.writes
	if err := live.Stop(); err == nil {
		t.Fatal("expected liveness write failure")
	}
	select {
	case <-cancelled:
	default:
		t.Fatal("stage was not cancelled")
	}
}

func TestCancelledStageDoesNotStartLiveness(t *testing.T) {
	writer := &signalWriter{writes: make(chan string, 1)}
	start := time.Unix(0, 0)
	logger := Logger{Out: writer, Format: "human", Interactive: true, Now: func() time.Time { return start }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	live := logger.StartLiveness(ctx, "review", start, func() {})
	if err := live.Stop(); err != nil {
		t.Fatal(err)
	}
	if got := writer.String(); got != "" {
		t.Fatalf("cancelled stage wrote %q", got)
	}
}

func TestHumanRendererRejectsUnknownStatus(t *testing.T) {
	logger := Logger{Out: ioDiscard{}, Format: "human"}
	for _, stage := range []string{"fix-findings", "fix-ci", "finalize"} {
		err := logger.Emit("stage_completed", F("stage", stage), F("status", "unexpected"), F("duration_ms", "1"))
		if err == nil || !strings.Contains(err.Error(), "unsupported human event") {
			t.Errorf("stage %s error = %v", stage, err)
		}
	}
}

func TestShimmerCoversWholeLineAtSupportedDepths(t *testing.T) {
	for _, depth := range []int{1, 2, 3} {
		got := shimmer("Ab", 1, depth)
		if !strings.Contains(got, "A") || !strings.Contains(got, "b") || !strings.HasSuffix(got, "\x1b[0m") {
			t.Fatalf("depth %d shimmer = %q", depth, got)
		}
		if count := strings.Count(got, "\x1b["); count < 3 {
			t.Fatalf("depth %d did not color every rune: %q", depth, got)
		}
	}
}

func TestShimmerHighlightTravelsAndReturnsWithoutWrapping(t *testing.T) {
	const length = 10
	tests := []struct {
		frame int
		want  int
	}{
		{0, -4},
		{8, 0},
		{34, 13},
		{36, 14},
		{38, 13},
	}
	for _, test := range tests {
		if got := shimmerHighlightCenter(length, test.frame); got != test.want {
			t.Errorf("frame %d: center = %d, want %d", test.frame, got, test.want)
		}
	}
}

func TestLivenessStageLabels(t *testing.T) {
	for _, stage := range []string{"review", "fix-findings", "finalize", "fix-ci"} {
		t.Run(stage, func(t *testing.T) {
			var out bytes.Buffer
			logger := Logger{Out: &out, Format: "human"}
			if err := logger.writeHeartbeat(stage, 2*time.Second); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "still running (2s)\n") {
				t.Fatalf("output = %q", out.String())
			}
		})
	}
	if got := HumanDuration(0); got != "0s" {
		t.Fatal(got)
	}
}
