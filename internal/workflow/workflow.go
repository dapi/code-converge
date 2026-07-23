package workflow

import (
	"context"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/dapi/code-converge/internal/codex"
	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/event"
	"github.com/dapi/code-converge/internal/repository"
	"github.com/dapi/code-converge/internal/runner"
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
	Finalize(context.Context, bool) (codex.Finalization, error)
	FixCI(context.Context) error
}

type Repository interface {
	HasChanges(context.Context) (bool, error)
	IsClean(context.Context) (bool, error)
	Head(context.Context) (string, error)
	Checkpoint(context.Context, string, bool) (repository.Checkpoint, error)
}

type Workflow struct {
	Config     config.Config
	Agent      Agent
	Repository Repository
	Log        *event.Logger
	Err        io.Writer
	Now        func() time.Time
}

func (w *Workflow) Run(ctx context.Context) int {
	w.Log.Err = w.Err
	w.Log.HumanMaxCycles = w.Config.MaxCycles
	w.Log.HumanMaxCIRecoveries = w.Config.MaxCIRecoveries
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
	checkpointed := false
	lastCheckpoint := repository.Checkpoint{}
	checkpointSkipReason := ""
	for {
		stageStarted := now()
		if !w.emit("stage_started", event.F("stage", "review"), event.F("model", w.stageModel("review")), event.F("reasoning_effort", w.stageReasoningEffort("review")), intField("review_phase", phase), intField("cycle", cycle)) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		stageCtx, cancelStage := context.WithCancel(ctx)
		liveness := w.Log.StartLiveness(stageCtx, event.StageContext{Stage: "review", Model: w.stageModel("review"), ReasoningEffort: w.stageReasoningEffort("review"), ReviewPhase: phase, Cycle: cycle}, stageStarted, cancelStage)
		review, err := w.Agent.Review(runner.WithStageContext(stageCtx, runner.StageContext{Stage: "review", ReviewPhase: phase, Cycle: cycle, Model: w.stageModel("review"), ReasoningEffort: w.stageReasoningEffort("review")}))
		livenessErr := liveness.Stop()
		cancelStage()
		duration := durationField(now().Sub(stageStarted))
		if livenessErr != nil {
			w.diagnostic("write liveness", livenessErr)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
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
		if review.Scope.Source != "" {
			fields = append(fields,
				event.F("review_scope", "branch_and_worktree"),
				event.F("review_base", review.Scope.BaseCommit),
				event.F("review_merge_base", review.Scope.MergeBase),
				event.F("review_base_source", review.Scope.Source),
			)
		}
		fields = append(fields, countFields(review.Counts)...)
		fields = append(fields, duration)
		if !w.emit("review_completed", fields...) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}

		if !review.Clean {
			if fixes >= w.Config.MaxCycles {
				return w.completeFindingsRemaining(now().Sub(runStarted), lastCheckpoint, fixes > 0, checkpointSkipReason)
			}
			canCheckpoint := w.Repository != nil
			initialHead := ""
			if w.Repository != nil {
				clean, err := w.Repository.IsClean(ctx)
				if err != nil {
					w.diagnostic("checkpoint status failed", err)
					return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
				}
				canCheckpoint = clean
				if !clean {
					checkpointSkipReason = "pre_existing_changes"
				} else {
					checkpointSkipReason = ""
				}
				initialHead, err = w.Repository.Head(ctx)
				if err != nil {
					w.diagnostic("checkpoint head failed", err)
					return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
				}
			}
			stageStarted = now()
			if !w.emit("stage_started", event.F("stage", "fix-findings"), event.F("model", w.stageModel("fix-findings")), event.F("reasoning_effort", w.stageReasoningEffort("fix-findings")), intField("review_phase", phase), intField("cycle", cycle)) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			stageCtx, cancelStage = context.WithCancel(ctx)
			liveness = w.Log.StartLiveness(stageCtx, event.StageContext{Stage: "fix-findings", Model: w.stageModel("fix-findings"), ReasoningEffort: w.stageReasoningEffort("fix-findings"), ReviewPhase: phase, Cycle: cycle}, stageStarted, cancelStage)
			err = w.Agent.FixFindings(runner.WithStageContext(stageCtx, runner.StageContext{Stage: "fix-findings", ReviewPhase: phase, Cycle: cycle, Model: w.stageModel("fix-findings"), ReasoningEffort: w.stageReasoningEffort("fix-findings")}), review.Report)
			livenessErr = liveness.Stop()
			cancelStage()
			if livenessErr != nil {
				w.diagnostic("write liveness", livenessErr)
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
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
			if w.Repository != nil {
				checkpoint, checkpointErr := w.Repository.Checkpoint(ctx, initialHead, canCheckpoint)
				if checkpointErr != nil {
					w.diagnostic("findings checkpoint failed", checkpointErr)
					return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
				}
				if checkpoint.Created {
					checkpointed = true
					lastCheckpoint = checkpoint
				}
			}
			fixes++
			cycle++
			continue
		}

		if w.Repository != nil {
			hasChanges, err := w.Repository.HasChanges(ctx)
			if err != nil {
				w.diagnostic("repository status failed", err)
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			if !hasChanges && !checkpointed {
				return w.complete("success", ExitSuccess, now().Sub(runStarted))
			}
		}

		stageStarted = now()
		if !w.emit("stage_started", event.F("stage", "finalize"), event.F("model", w.stageModel("finalize")), event.F("reasoning_effort", w.stageReasoningEffort("finalize"))) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		stageCtx, cancelStage = context.WithCancel(ctx)
		liveness = w.Log.StartLiveness(stageCtx, event.StageContext{Stage: "finalize", Model: w.stageModel("finalize"), ReasoningEffort: w.stageReasoningEffort("finalize"), ReviewPhase: phase, Cycle: cycle}, stageStarted, cancelStage)
		finalization, err := w.Agent.Finalize(runner.WithStageContext(stageCtx, runner.StageContext{Stage: "finalize", ReviewPhase: phase, Cycle: cycle, Model: w.stageModel("finalize"), ReasoningEffort: w.stageReasoningEffort("finalize")}), checkpointed)
		livenessErr = liveness.Stop()
		cancelStage()
		if livenessErr != nil {
			w.diagnostic("write liveness", livenessErr)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
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
			// CI_FAILED is a published finalization result. A subsequent review
			// phase must not describe this already-pushed checkpoint as local.
			checkpointed = false
			lastCheckpoint = repository.Checkpoint{}
			checkpointSkipReason = ""
			if recoveries >= w.Config.MaxCIRecoveries {
				return w.complete("ci_failure", ExitCI, now().Sub(runStarted))
			}
			stageStarted = now()
			if !w.emit("stage_started", event.F("stage", "fix-ci"), event.F("model", w.stageModel("fix-ci")), event.F("reasoning_effort", w.stageReasoningEffort("fix-ci")), intField("review_phase", phase)) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			stageCtx, cancelStage = context.WithCancel(ctx)
			liveness = w.Log.StartLiveness(stageCtx, event.StageContext{Stage: "fix-ci", Model: w.stageModel("fix-ci"), ReasoningEffort: w.stageReasoningEffort("fix-ci"), ReviewPhase: phase, Cycle: cycle}, stageStarted, cancelStage)
			err = w.Agent.FixCI(runner.WithStageContext(stageCtx, runner.StageContext{Stage: "fix-ci", ReviewPhase: phase, Cycle: cycle, Model: w.stageModel("fix-ci"), ReasoningEffort: w.stageReasoningEffort("fix-ci")}))
			livenessErr = liveness.Stop()
			cancelStage()
			if livenessErr != nil {
				w.diagnostic("write liveness", livenessErr)
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			stageStatus := "success"
			if err != nil {
				stageStatus = "failed"
			}
			if !w.emit("stage_completed", event.F("stage", "fix-ci"), event.F("model", w.stageModel("fix-ci")), event.F("reasoning_effort", w.stageReasoningEffort("fix-ci")), intField("review_phase", phase), event.F("status", stageStatus), durationField(now().Sub(stageStarted))) {
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

func (w *Workflow) completeFindingsRemaining(elapsed time.Duration, checkpoint repository.Checkpoint, hadFixes bool, skippedReason string) int {
	fields := []event.Field{event.F("status", "findings_remaining"), intField("exit_code", ExitFindingsRemaining), event.F("total_duration_ms", milliseconds(elapsed))}
	if checkpoint.Created {
		fields = append(fields, event.F("checkpoint_status", "committed_local"), event.F("checkpoint_branch", url.QueryEscape(checkpoint.Branch)), event.F("checkpoint_commit", checkpoint.Commit))
	} else if hadFixes {
		if skippedReason != "" {
			fields = append(fields, event.F("checkpoint_status", "not_attempted"), event.F("checkpoint_reason", skippedReason))
		} else {
			fields = append(fields, event.F("checkpoint_status", "no_changes"))
		}
	} else {
		fields = append(fields, event.F("checkpoint_status", "not_attempted"), event.F("checkpoint_reason", "fix_budget_exhausted"))
	}
	if !w.emit("run_completed", fields...) {
		return ExitOperational
	}
	return ExitFindingsRemaining
}

func (w *Workflow) emitSteps(result codex.Finalization) bool {
	for _, step := range []struct{ name, status string }{
		{"commit", result.Commit}, {"push", result.Push}, {"change_request", result.ChangeRequest}, {"ci", result.CI},
	} {
		if !w.emit("step_completed", event.F("stage", "finalize"), event.F("model", w.stageModel("finalize")), event.F("reasoning_effort", w.stageReasoningEffort("finalize")), event.F("step", step.name), event.F("status", step.status)) {
			return false
		}
	}
	return true
}

func (w *Workflow) stageModel(stage string) string {
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

func (w *Workflow) stageReasoningEffort(stage string) string {
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
	case "finalize":
		if w.Config.FinalizeEffort != "" {
			return w.Config.FinalizeEffort
		}
		return "agent-default"
	case "fix-ci":
		if w.Config.CIFixEffort != "" {
			return w.Config.CIFixEffort
		}
		return "agent-default"
	default:
		return "unknown"
	}
}

func (w *Workflow) emitUnknownSteps() bool {
	return w.emitSteps(codex.Finalization{Commit: "unknown", Push: "unknown", ChangeRequest: "unknown", CI: "unknown"})
}

func (w *Workflow) emit(name string, fields ...event.Field) bool {
	if err := w.Log.Emit(name, fields...); err != nil {
		w.diagnostic("write event stream", err)
		return false
	}
	return true
}

func (w *Workflow) diagnostic(message string, err error) {
	w.Log.Diagnostic(message, err)
}

func (w *Workflow) complete(status string, exitCode int, elapsed time.Duration) int {
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
