package workflow

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/dapi/code-converge/internal/codex"
	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/event"
)

const (
	ExitSuccess           = 0
	ExitFindingsRemaining = 1
	ExitOperational       = 2
	ExitCI                = 3
)

type Agent interface {
	Review(context.Context) (codex.ReviewResult, error)
	FixFindings(context.Context, string) error
	Finalize(context.Context) (codex.Finalization, error)
	FixCI(context.Context) error
}

type Workflow struct {
	Config config.Config
	Agent  Agent
	Log    event.Logger
	Err    io.Writer
	Now    func() time.Time
}

func (w Workflow) Run(ctx context.Context) int {
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	runStarted := now()
	if !w.emit("run_started") {
		return ExitOperational
	}

	phase, cycle := 1, 1
	fixes, recoveries := 0, 0
	for {
		stageStarted := now()
		if !w.emit("stage_started", event.F("stage", "review"), event.F("model", w.stageModel("review")), event.F("reasoning_effort", w.stageReasoningEffort("review")), intField("review_phase", phase), intField("cycle", cycle)) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		review, err := w.Agent.Review(ctx)
		duration := durationField(now().Sub(stageStarted))
		if err != nil {
			if !w.emit("review_completed", event.F("stage", "review"), event.F("model", w.stageModel("review")), event.F("reasoning_effort", w.stageReasoningEffort("review")), intField("review_phase", phase), intField("cycle", cycle), event.F("status", "failed"), duration) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			w.diagnostic("review failed", err)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		status := "findings"
		if review.Clean {
			status = "clean"
		}
		fields := []event.Field{event.F("stage", "review"), event.F("model", w.stageModel("review")), event.F("reasoning_effort", w.stageReasoningEffort("review")), intField("review_phase", phase), intField("cycle", cycle), event.F("status", status)}
		fields = append(fields, countFields(review.Counts)...)
		fields = append(fields, duration)
		if !w.emit("review_completed", fields...) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}

		if !review.Clean {
			if fixes >= w.Config.MaxCycles {
				return w.complete("findings_remaining", ExitFindingsRemaining, now().Sub(runStarted))
			}
			stageStarted = now()
			if !w.emit("stage_started", event.F("stage", "fix-findings"), event.F("model", w.stageModel("fix-findings")), event.F("reasoning_effort", w.stageReasoningEffort("fix-findings")), intField("review_phase", phase), intField("cycle", cycle)) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			err = w.Agent.FixFindings(ctx, review.Report)
			stageStatus := "success"
			if err != nil {
				stageStatus = "failed"
			}
			if !w.emit("stage_completed", event.F("stage", "fix-findings"), event.F("model", w.stageModel("fix-findings")), event.F("reasoning_effort", w.stageReasoningEffort("fix-findings")), intField("review_phase", phase), intField("cycle", cycle), event.F("status", stageStatus), durationField(now().Sub(stageStarted))) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			if err != nil {
				w.diagnostic("fix-findings failed", err)
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			fixes++
			cycle++
			continue
		}

		stageStarted = now()
		if !w.emit("stage_started", event.F("stage", "finalize"), event.F("model", w.stageModel("finalize")), event.F("reasoning_effort", w.stageReasoningEffort("finalize"))) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		finalization, err := w.Agent.Finalize(ctx)
		if err != nil {
			if !w.emitUnknownSteps() || !w.emit("stage_completed", event.F("stage", "finalize"), event.F("model", w.stageModel("finalize")), event.F("reasoning_effort", w.stageReasoningEffort("finalize")), event.F("status", "failed"), durationField(now().Sub(stageStarted))) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			w.diagnostic("finalization failed", err)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		if !w.emitSteps(finalization) || !w.emit("stage_completed", event.F("stage", "finalize"), event.F("model", w.stageModel("finalize")), event.F("reasoning_effort", w.stageReasoningEffort("finalize")), event.F("status", "success"), event.F("verdict", finalization.Verdict), durationField(now().Sub(stageStarted))) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}

		switch finalization.Verdict {
		case "SUCCESS":
			return w.complete("success", ExitSuccess, now().Sub(runStarted))
		case "FAILED":
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		case "CI_FAILED":
			if recoveries >= w.Config.MaxCIRecoveries {
				return w.complete("ci_failure", ExitCI, now().Sub(runStarted))
			}
			stageStarted = now()
			if !w.emit("stage_started", event.F("stage", "fix-ci"), event.F("model", w.stageModel("fix-ci")), event.F("reasoning_effort", w.stageReasoningEffort("fix-ci"))) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			err = w.Agent.FixCI(ctx)
			stageStatus := "success"
			if err != nil {
				stageStatus = "failed"
			}
			if !w.emit("stage_completed", event.F("stage", "fix-ci"), event.F("model", w.stageModel("fix-ci")), event.F("reasoning_effort", w.stageReasoningEffort("fix-ci")), event.F("status", stageStatus), durationField(now().Sub(stageStarted))) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			if err != nil {
				w.diagnostic("CI fix failed", err)
				return w.complete("ci_failure", ExitCI, now().Sub(runStarted))
			}
			recoveries++
			phase++
			cycle, fixes = 1, 0
		}
	}
}

func (w Workflow) emitSteps(result codex.Finalization) bool {
	for _, step := range []struct{ name, status string }{
		{"commit", result.Commit}, {"push", result.Push}, {"change_request", result.ChangeRequest}, {"ci", result.CI},
	} {
		if !w.emit("step_completed", event.F("stage", "finalize"), event.F("model", w.stageModel("finalize")), event.F("reasoning_effort", w.stageReasoningEffort("finalize")), event.F("step", step.name), event.F("status", step.status)) {
			return false
		}
	}
	return true
}

func (w Workflow) stageModel(stage string) string {
	switch stage {
	case "review":
		if w.Config.ReviewModel == "" {
			return "gpt-5.6-sol"
		}
		return w.Config.ReviewModel
	case "fix-findings":
		if w.Config.FixModel == "" {
			return "gpt-5.6-luna"
		}
		return w.Config.FixModel
	case "finalize":
		if w.Config.FinalizeModel == "" {
			return "gpt-5.3-codex-spark"
		}
		return w.Config.FinalizeModel
	case "fix-ci":
		if w.Config.CIFixModel != "" {
			return w.Config.CIFixModel
		}
		return "agent-default"
	default:
		return "unknown"
	}
}

func (w Workflow) stageReasoningEffort(stage string) string {
	switch stage {
	case "review":
		if w.Config.ReviewEffort != "" {
			return w.Config.ReviewEffort
		}
		return "medium"
	case "fix-findings":
		if w.Config.FixEffort != "" {
			return w.Config.FixEffort
		}
		return "medium"
	case "finalize", "fix-ci":
		return "agent-default"
	default:
		return "unknown"
	}
}

func (w Workflow) emitUnknownSteps() bool {
	return w.emitSteps(codex.Finalization{Commit: "unknown", Push: "unknown", ChangeRequest: "unknown", CI: "unknown"})
}

func (w Workflow) emit(name string, fields ...event.Field) bool {
	if err := w.Log.Emit(name, fields...); err != nil {
		w.diagnostic("write event stream", err)
		return false
	}
	return true
}

func (w Workflow) diagnostic(message string, err error) {
	if w.Err != nil {
		fmt.Fprintf(w.Err, "code-converge: %s: %v\n", message, err)
	}
}

func (w Workflow) complete(status string, exitCode int, elapsed time.Duration) int {
	if !w.emit("run_completed", event.F("status", status), intField("exit_code", exitCode), event.F("total_duration_ms", milliseconds(elapsed))) {
		return ExitOperational
	}
	return exitCode
}

func intField(key string, value int) event.Field { return event.F(key, strconv.Itoa(value)) }

func durationField(value time.Duration) event.Field {
	return event.F("duration_ms", milliseconds(value))
}

func milliseconds(value time.Duration) string {
	ms := value.Milliseconds()
	if ms < 0 {
		ms = 0
	}
	return strconv.FormatInt(ms, 10)
}

func countFields(counts codex.Counts) []event.Field {
	return []event.Field{
		intField("findings_total", counts.Total()),
		intField("findings_critical", counts.Critical),
		intField("findings_high", counts.High),
		intField("findings_medium", counts.Medium),
		intField("findings_low", counts.Low),
		intField("findings_unknown", counts.Unknown),
	}
}
