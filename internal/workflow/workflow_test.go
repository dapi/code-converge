package workflow

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dapi/code-converge/internal/codex"
	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/event"
	"github.com/dapi/code-converge/internal/repository"
)

type fakeAgent struct {
	reviews        []codex.ReviewResult
	reviewFailures map[int]error
	finalizations  []codex.Finalization
	finalizeErr    error
	fixErr         error
	ciFixErr       error
	fixReports     []string
	reviewCalls    int
	reviewWait     bool
	reviewStarted  chan struct{}
	fixCalls       int
	finalizeCalls  int
	ciFixCalls     int
	ciFixWait      bool
	ciFixStarted   chan struct{}
}

type fakeRepository struct {
	hasChanges bool
	err        error
	calls      int
}

func (f *fakeRepository) HasChanges(context.Context) (bool, error) {
	f.calls++
	return f.hasChanges, f.err
}

func (f *fakeAgent) Review(ctx context.Context) (codex.ReviewResult, error) {
	index := f.reviewCalls
	f.reviewCalls++
	if f.reviewStarted != nil {
		close(f.reviewStarted)
	}
	if f.reviewWait {
		<-ctx.Done()
		return codex.ReviewResult{}, ctx.Err()
	}
	if err := f.reviewFailures[index]; err != nil {
		return codex.ReviewResult{}, err
	}
	if index >= len(f.reviews) {
		return codex.ReviewResult{}, errors.New("missing review fixture")
	}
	return f.reviews[index], nil
}

func (f *fakeAgent) FixFindings(_ context.Context, report string) error {
	f.fixCalls++
	f.fixReports = append(f.fixReports, report)
	return f.fixErr
}

func (f *fakeAgent) Finalize(context.Context) (codex.Finalization, error) {
	index := f.finalizeCalls
	f.finalizeCalls++
	if f.finalizeErr != nil {
		return codex.Finalization{}, f.finalizeErr
	}
	if index >= len(f.finalizations) {
		return codex.Finalization{}, errors.New("missing finalization fixture")
	}
	return f.finalizations[index], nil
}

func (f *fakeAgent) FixCI(ctx context.Context) error {
	f.ciFixCalls++
	if f.ciFixStarted != nil {
		close(f.ciFixStarted)
	}
	if f.ciFixWait {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.ciFixErr
}

func success() codex.Finalization {
	return codex.Finalization{Verdict: "SUCCESS", Commit: "success", Push: "success", ChangeRequest: "skipped", CI: "success"}
}

func ciFailed() codex.Finalization {
	return codex.Finalization{Verdict: "CI_FAILED", Commit: "success", Push: "success", ChangeRequest: "success", CI: "failed"}
}

func findings() codex.ReviewResult {
	return codex.ReviewResult{Counts: codex.Counts{High: 1}, Report: "## Findings\n- [P1] a finding"}
}

func clean() codex.ReviewResult { return codex.ReviewResult{Clean: true} }

func run(t *testing.T, cfg config.Config, agent *fakeAgent) (int, string, string) {
	return runWithRepository(t, cfg, agent, &fakeRepository{hasChanges: true})
}

func runWithRepository(t *testing.T, cfg config.Config, agent *fakeAgent, repository Repository) (int, string, string) {
	t.Helper()
	var out, stderr bytes.Buffer
	tick := 0
	now := func() time.Time {
		tick++
		return time.Date(2026, 7, 21, 10, 0, 0, tick*int(time.Millisecond), time.UTC)
	}
	w := Workflow{Config: cfg, Agent: agent, Repository: repository, Log: &event.Logger{Out: &out, Now: now, Format: cfg.LogFormat, Heartbeat: cfg.Heartbeat}, Err: &stderr, Now: now}
	return w.Run(context.Background()), out.String(), stderr.String()
}

func TestHumanHappyPath(t *testing.T) {
	agent := &fakeAgent{
		reviews:       []codex.ReviewResult{{Counts: codex.Counts{High: 1, Medium: 2}, Report: "findings"}, clean()},
		finalizations: []codex.Finalization{success()},
	}
	code, output, stderr := run(t, config.Config{LogFormat: "human", MaxCycles: 1}, agent)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{
		"10:00:00 [1/1] [gpt-5.6-sol/medium] Review started\n", "10:00:00 [1/1] [gpt-5.6-sol/medium] Review: 3 findings — 1 high, 2 medium (0s)\n",
		"10:00:00 [1/1] [gpt-5.6-luna/medium] Fixing findings\n", "10:00:00 [1/1] [gpt-5.6-luna/medium] Findings fixed (0s)\n", "10:00:00 [2/1] [gpt-5.6-sol/medium] Review: clean (0s)\n",
		"10:00:00 [gpt-5.3-codex-spark/agent-default] Finalizing\n", "10:00:00 [gpt-5.3-codex-spark/agent-default]   Commit: done\n", "10:00:00 [gpt-5.3-codex-spark/agent-default]   Change request: not needed\n", "10:00:00 [gpt-5.3-codex-spark/agent-default] Finalized successfully (0s)\n", "10:00:00 Done (0s)\n",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("missing %q in:\n%s", want, output)
		}
	}
	if strings.Contains(output, "event=") || strings.Contains(output, "findings_critical") || strings.Contains(output, "duration_ms") {
		t.Fatalf("human output leaked kv fields:\n%s", output)
	}
}

func TestHumanTerminalPaths(t *testing.T) {
	tests := []struct {
		name  string
		cfg   config.Config
		agent *fakeAgent
		code  int
		want  string
	}{
		{"findings", config.Config{LogFormat: "human", MaxCycles: 0}, &fakeAgent{reviews: []codex.ReviewResult{findings()}}, ExitFindingsRemaining, "Stopped: review findings remain"},
		{"operational", config.Config{LogFormat: "human"}, &fakeAgent{reviewFailures: map[int]error{0: errors.New("bad")}}, ExitOperational, "Failed due to an operational error"},
		{"ci", config.Config{LogFormat: "human", MaxCIRecoveries: 0}, &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{ciFailed()}}, ExitCI, "Stopped: CI is still failing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, output, _ := run(t, test.cfg, test.agent)
			if code != test.code || !strings.Contains(output, test.want) {
				t.Fatalf("code=%d output=\n%s", code, output)
			}
		})
	}
}

func TestHappyPath(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{success()}}
	code, output, stderr := run(t, config.Config{MaxCycles: 10, MaxCIRecoveries: 3, ReviewModel: "review-model", ReviewEffort: "high", FixModel: "fix-model", FixEffort: "low", FinalizeModel: "finalize-model"}, agent)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	assertRecord(t, output, "event=review_completed", "status=clean", "findings_total=0")
	assertRecord(t, output, "event=stage_started", "stage=review", "model=review-model")
	assertRecord(t, output, "event=stage_started", "stage=review", "reasoning_effort=high")
	assertRecord(t, output, "event=stage_started", "stage=finalize", "model=finalize-model")
	assertRecord(t, output, "event=stage_started", "stage=finalize", "reasoning_effort=agent-default")
	assertRecord(t, output, "event=step_completed", "stage=finalize", "model=finalize-model")
	assertRecord(t, output, "event=run_completed", "status=success", "exit_code=0")
	for _, step := range []string{"commit", "push", "change_request", "ci"} {
		assertRecord(t, output, "event=step_completed", "step="+step)
	}
}

func TestCleanNoChangeCompletesWithoutFinalization(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{clean()}}
	repository := &fakeRepository{}
	code, output, stderr := runWithRepository(t, config.Config{}, agent, repository)
	if code != ExitSuccess || stderr != "" || repository.calls != 1 || agent.finalizeCalls != 0 {
		t.Fatalf("code=%d stderr=%q status calls=%d finalize calls=%d", code, stderr, repository.calls, agent.finalizeCalls)
	}
	assertRecord(t, output, "event=review_completed", "status=clean", "findings_total=0")
	assertRecord(t, output, "event=run_completed", "status=success", "exit_code=0")
	if strings.Contains(output, "stage=finalize") {
		t.Fatalf("no-change run started finalization:\n%s", output)
	}
}

func TestReviewMetadataUsesResolvedCommitForEventSafety(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{{Clean: true, Scope: repository.ReviewTarget{Base: "release=1", BaseCommit: "0123456789abcdef", MergeBase: "abcdef0123456789", Source: "explicit"}}}, finalizations: []codex.Finalization{success()}}
	code, output, stderr := run(t, config.Config{}, agent)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	assertRecord(t, output, "event=review_completed", "review_base=0123456789abcdef", "review_merge_base=abcdef0123456789", "review_base_source=explicit")
	if strings.Contains(output, "release=1") {
		t.Fatalf("raw ref leaked into event stream:\n%s", output)
	}
}

func TestRepositoryStatusFailureIsOperational(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{clean()}}
	repository := &fakeRepository{err: errors.New("git unavailable")}
	code, output, stderr := runWithRepository(t, config.Config{}, agent, repository)
	if code != ExitOperational || agent.finalizeCalls != 0 {
		t.Fatalf("code=%d finalize calls=%d", code, agent.finalizeCalls)
	}
	assertRecord(t, output, "event=review_completed", "status=clean")
	assertRecord(t, output, "event=run_completed", "status=operational_failure", "exit_code=2")
	if !strings.Contains(stderr, "repository status failed") {
		t.Fatalf("stderr=%q", stderr)
	}
}

func TestMandatoryVerificationAndFindingsLimit(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{findings(), findings(), findings()}}
	code, output, _ := run(t, config.Config{MaxCycles: 2}, agent)
	if code != ExitFindingsRemaining || agent.fixCalls != 2 || agent.reviewCalls != 3 {
		t.Fatalf("code=%d fixes=%d reviews=%d", code, agent.fixCalls, agent.reviewCalls)
	}
	assertRecord(t, output, "event=stage_started", "stage=review", "cycle=3")
	assertRecord(t, output, "event=run_completed", "status=findings_remaining", "exit_code=1")
}

func TestZeroFixBudget(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{findings()}}
	code, _, _ := run(t, config.Config{MaxCycles: 0}, agent)
	if code != ExitFindingsRemaining || agent.fixCalls != 0 || agent.reviewCalls != 1 {
		t.Fatalf("code=%d fixes=%d reviews=%d", code, agent.fixCalls, agent.reviewCalls)
	}
}

func TestFixReceivesReviewReport(t *testing.T) {
	result := findings()
	agent := &fakeAgent{reviews: []codex.ReviewResult{result, clean()}, finalizations: []codex.Finalization{success()}}
	code, _, _ := run(t, config.Config{MaxCycles: 1}, agent)
	if code != ExitSuccess || len(agent.fixReports) != 1 || agent.fixReports[0] != result.Report {
		t.Fatalf("code=%d reports=%q", code, agent.fixReports)
	}
}

func TestCIRecoveryRestartsReviewPhase(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{clean(), clean()}, finalizations: []codex.Finalization{ciFailed(), success()}}
	code, output, _ := run(t, config.Config{MaxCycles: 1, MaxCIRecoveries: 1}, agent)
	if code != ExitSuccess || agent.ciFixCalls != 1 {
		t.Fatalf("code=%d ci fixes=%d", code, agent.ciFixCalls)
	}
	assertRecord(t, output, "event=stage_started", "stage=review", "review_phase=2", "cycle=1")
}

func TestStageModelsAreLogged(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{findings(), clean(), clean()}, finalizations: []codex.Finalization{ciFailed(), success()}}
	_, output, _ := run(t, config.Config{MaxCycles: 1, MaxCIRecoveries: 1, ReviewModel: "review", ReviewEffort: "high", FixModel: "fix", FixEffort: "low", FinalizeModel: "final", FinalizeEffort: "medium", CIFixModel: "ci", CIFixEffort: "high"}, agent)
	for _, stage := range []struct{ name, model, effort string }{{"review", "review", "high"}, {"fix-findings", "fix", "low"}, {"finalize", "final", "medium"}, {"fix-ci", "ci", "high"}} {
		assertRecord(t, output, "event=stage_started", "stage="+stage.name, "model="+stage.model, "reasoning_effort="+stage.effort)
	}
}

func TestCIFailurePaths(t *testing.T) {
	tests := []struct {
		name  string
		cfg   config.Config
		agent *fakeAgent
	}{
		{"exhausted", config.Config{MaxCIRecoveries: 0}, &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{ciFailed()}}},
		{"fix failed", config.Config{MaxCIRecoveries: 1}, &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{ciFailed()}, ciFixErr: errors.New("red")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, output, _ := run(t, test.cfg, test.agent)
			if code != ExitCI {
				t.Fatalf("code=%d", code)
			}
			assertRecord(t, output, "event=run_completed", "status=ci_failure", "exit_code=3")
		})
	}
}

func TestOperationalFailures(t *testing.T) {
	tests := []struct {
		name      string
		agent     *fakeAgent
		wantEvent []string
	}{
		{"review", &fakeAgent{reviewFailures: map[int]error{0: errors.New("bad report")}}, []string{"event=review_completed", "status=failed"}},
		{"fix", &fakeAgent{reviews: []codex.ReviewResult{findings()}, fixErr: errors.New("fix failed")}, []string{"event=stage_completed", "stage=fix-findings", "status=failed"}},
		{"finalize", &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizeErr: errors.New("bad json")}, []string{"event=stage_completed", "stage=finalize", "status=failed"}},
		{"failed verdict", &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{{Verdict: "FAILED", Commit: "failed", Push: "skipped", ChangeRequest: "skipped", CI: "skipped"}}}, []string{"event=stage_completed", "verdict=FAILED"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, output, _ := run(t, config.Config{MaxCycles: 1}, test.agent)
			if code != ExitOperational {
				t.Fatalf("code=%d", code)
			}
			assertRecord(t, output, test.wantEvent...)
			assertRecord(t, output, "event=run_completed", "status=operational_failure", "exit_code=2")
			if test.name == "finalize" {
				if countRecords(output, "event=step_completed") != 4 || countRecords(output, "status=unknown") != 4 {
					t.Fatalf("unknown steps missing:\n%s", output)
				}
			}
		})
	}
}

func TestEveryRecordIsMachineSafe(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{success()}}
	_, output, _ := run(t, config.Config{}, agent)
	for number, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "ts=") || !strings.HasPrefix(fields[1], "event=") {
			t.Fatalf("line %d has invalid prefix: %q", number+1, line)
		}
		for _, field := range fields {
			if strings.Count(field, "=") != 1 {
				t.Fatalf("line %d invalid field %q", number+1, field)
			}
		}
	}
}

type failingWriter struct{ writes int }

func (w *failingWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.writes >= 2 {
		return 0, errors.New("closed stdout")
	}
	return len(data), nil
}

func TestEventFailureStopsBeforeAgentSideEffects(t *testing.T) {
	agent := &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{success()}}
	writer := &failingWriter{}
	var stderr bytes.Buffer
	w := Workflow{Config: config.Config{}, Agent: agent, Log: &event.Logger{Out: writer}, Err: &stderr}
	if code := w.Run(context.Background()); code != ExitOperational {
		t.Fatalf("code=%d", code)
	}
	if agent.reviewCalls != 0 || agent.fixCalls != 0 || agent.finalizeCalls != 0 || agent.ciFixCalls != 0 {
		t.Fatalf("agent invoked after event failure: %#v", agent)
	}
	if !strings.Contains(stderr.String(), "write event stream") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

type configurableFailingWriter struct {
	writes    int
	failAfter int
}

func (w *configurableFailingWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.writes >= w.failAfter {
		return 0, errors.New("closed stdout")
	}
	return len(data), nil
}

func TestEmitFailureMidWorkflow(t *testing.T) {
	tests := []struct {
		name      string
		failAfter int
		agent     *fakeAgent
	}{
		{"review_completed", 3, &fakeAgent{reviews: []codex.ReviewResult{clean()}}},
		{"fix-findings stage_started", 3, &fakeAgent{reviews: []codex.ReviewResult{findings()}}},
		{"finalize stage_started", 4, &fakeAgent{reviews: []codex.ReviewResult{clean()}}},
		{"run_completed", 10, &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{success()}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writer := &configurableFailingWriter{failAfter: test.failAfter}
			var stderr bytes.Buffer
			w := Workflow{Config: config.Config{MaxCycles: 1}, Agent: test.agent, Log: &event.Logger{Out: writer}, Err: &stderr}
			if code := w.Run(context.Background()); code != ExitOperational {
				t.Fatalf("code=%d", code)
			}
			if !strings.Contains(stderr.String(), "write event stream") {
				t.Fatalf("stderr=%q", stderr.String())
			}
		})
	}
}

func TestMillisecondsNegative(t *testing.T) {
	if got := milliseconds(-time.Second); got != "0" {
		t.Fatalf("milliseconds(-1s) = %q, want 0", got)
	}
}

func TestEmitStepsFailure(t *testing.T) {
	writer := &configurableFailingWriter{failAfter: 5}
	var stderr bytes.Buffer
	agent := &fakeAgent{reviews: []codex.ReviewResult{clean()}, finalizations: []codex.Finalization{success()}}
	w := Workflow{Config: config.Config{}, Agent: agent, Log: &event.Logger{Out: writer}, Err: &stderr}
	if code := w.Run(context.Background()); code != ExitOperational {
		t.Fatalf("code=%d", code)
	}
}

type failOnLivenessWriter struct{ bytes.Buffer }

func (w *failOnLivenessWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "still running") {
		return 0, errors.New("closed stdout")
	}
	return w.Buffer.Write(p)
}

func TestCIFixLivenessWriteFailureIsOperational(t *testing.T) {
	ticks := make(chan time.Time, 1)
	started := make(chan struct{})
	agent := &fakeAgent{
		reviews:       []codex.ReviewResult{clean()},
		finalizations: []codex.Finalization{ciFailed()},
		ciFixWait:     true,
		ciFixStarted:  started,
	}
	writer := &failOnLivenessWriter{}
	var stderr bytes.Buffer
	logger := &event.Logger{
		Out: writer, Format: "human", Heartbeat: time.Second,
		Tick: func(time.Duration) (<-chan time.Time, func()) { return ticks, func() {} },
	}
	w := Workflow{Config: config.Config{LogFormat: "human", Heartbeat: time.Second, MaxCIRecoveries: 1}, Agent: agent, Log: logger, Err: &stderr}
	result := make(chan int, 1)
	go func() { result <- w.Run(context.Background()) }()
	<-started
	ticks <- time.Now().Add(time.Second)
	if code := <-result; code != ExitOperational {
		t.Fatalf("code=%d output=%q stderr=%q", code, writer.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "write liveness") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestCancellationStopsActiveLivenessWithoutLateWrites(t *testing.T) {
	ticks := make(chan time.Time, 2)
	started := make(chan struct{})
	agent := &fakeAgent{reviewWait: true, reviewStarted: started}
	var output, stderr bytes.Buffer
	logger := &event.Logger{
		Out: &output, Format: "human", Heartbeat: time.Second,
		Tick: func(time.Duration) (<-chan time.Time, func()) { return ticks, func() {} },
	}
	w := Workflow{Config: config.Config{LogFormat: "human", Heartbeat: time.Second}, Agent: agent, Log: logger, Err: &stderr}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan int, 1)
	go func() { result <- w.Run(ctx) }()
	<-started
	cancel()
	if code := <-result; code != ExitOperational {
		t.Fatalf("code=%d output=%q stderr=%q", code, output.String(), stderr.String())
	}
	before := output.String()
	ticks <- time.Now().Add(time.Minute)
	if after := output.String(); after != before {
		t.Fatalf("late output after cancellation: before=%q after=%q", before, after)
	}
	if !strings.Contains(before, "[1/0] [gpt-5.6-sol/medium] Review failed") || !strings.Contains(before, "Failed due to an operational error") {
		t.Fatalf("missing cancellation terminal output: %q", before)
	}
}

func TestCIFixResetsPhaseAndFixes(t *testing.T) {
	agent := &fakeAgent{
		reviews:       []codex.ReviewResult{clean(), findings(), clean()},
		finalizations: []codex.Finalization{ciFailed(), success()},
	}
	code, output, _ := run(t, config.Config{MaxCycles: 1, MaxCIRecoveries: 1}, agent)
	if code != ExitSuccess {
		t.Fatalf("code=%d", code)
	}
	assertRecord(t, output, "event=stage_started", "stage=review", "review_phase=2", "cycle=1")
	if agent.fixCalls != 1 {
		t.Fatalf("expected one fix attempt in second phase, got %d", agent.fixCalls)
	}
}

func assertRecord(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		matched := true
		for _, fragment := range fragments {
			if !strings.Contains(line, fragment) {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("no record contains %v:\n%s", fragments, output)
}

func countRecords(output, fragment string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, fragment) {
			count++
		}
	}
	return count
}
