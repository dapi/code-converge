package workflow

import (
	"context"
	"fmt"
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
	ExitInterrupted       = 130
)

type Agent interface {
	Review(context.Context) (codex.ReviewResult, error)
	FixFindings(context.Context, string) error
	FixCI(context.Context) error
}

type Repository interface {
	HasChanges(context.Context) (bool, error)
	IsClean(context.Context) (bool, error)
	Head(context.Context) (string, error)
	Checkpoint(context.Context, string, bool) (repository.Checkpoint, error)
	Publish(context.Context, bool) (repository.Publication, error)
	WaitCI(context.Context, repository.Publication) (repository.CIResult, error)
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
	initialWorktreeClean := true
	if w.Repository != nil {
		var err error
		initialWorktreeClean, err = w.Repository.IsClean(ctx)
		if err != nil {
			w.diagnostic("initial repository status failed", err)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
	}

	phase, cycle := 1, 1
	fixes, recoveries := 0, 0
	checkpointed := false
	lastCheckpoint := repository.Checkpoint{}
	checkpointSkipReason := ""
	for {
		if ctx.Err() != nil {
			return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
		}
		stageStarted := now()
		if !w.emit("stage_started", event.F("stage", "review"), event.F("model", w.stageModel("review")), event.F("reasoning_effort", w.stageReasoningEffort("review")), intField("review_phase", phase), intField("cycle", cycle)) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		stageCtx, cancelStage := context.WithCancel(ctx)
		liveness := w.Log.StartLiveness(stageCtx, event.StageContext{Stage: "review", Model: w.stageModel("review"), ReasoningEffort: w.stageReasoningEffort("review"), ReviewPhase: phase, Cycle: cycle}, stageStarted, cancelStage)
		if err := w.Log.StartAgent("review " + strconv.Itoa(phase) + "." + strconv.Itoa(cycle)); err != nil {
			_ = liveness.Stop()
			cancelStage()
			w.diagnostic("render interactive view", err)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		review, err := w.Agent.Review(runner.WithStageContext(stageCtx, runner.StageContext{Stage: "review", ReviewPhase: phase, Cycle: cycle, Model: w.stageModel("review"), ReasoningEffort: w.stageReasoningEffort("review")}))
		var presentationErr error
		if err != nil && ctx.Err() != nil {
			presentationErr = w.Log.CompleteAgent("review cancelled")
		} else if err != nil {
			presentationErr = w.Log.CompleteAgent("review failed")
		} else {
			presentationErr = w.Log.CompleteAgent("review completed")
		}
		livenessErr := liveness.Stop()
		cancelStage()
		duration := durationField(now().Sub(stageStarted))
		if livenessErr != nil {
			w.diagnostic("write liveness", livenessErr)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		if presentationErr != nil {
			w.diagnostic("render interactive view", presentationErr)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		if err != nil {
			if ctx.Err() != nil {
				return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
			}
			if !w.emit("review_completed", event.F("stage", "review"), event.F("model", w.stageModel("review")), event.F("reasoning_effort", w.stageReasoningEffort("review")), intField("review_phase", phase), intField("cycle", cycle), event.F("status", "failed"), duration) {
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			w.diagnostic("review failed", err)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		if ctx.Err() != nil {
			return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
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
					if ctx.Err() != nil {
						return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
					}
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
					if ctx.Err() != nil {
						return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
					}
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
			if err := w.Log.StartAgent("fix-findings " + strconv.Itoa(phase) + "." + strconv.Itoa(cycle)); err != nil {
				_ = liveness.Stop()
				cancelStage()
				w.diagnostic("render interactive view", err)
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			err = w.Agent.FixFindings(runner.WithStageContext(stageCtx, runner.StageContext{Stage: "fix-findings", ReviewPhase: phase, Cycle: cycle, Model: w.stageModel("fix-findings"), ReasoningEffort: w.stageReasoningEffort("fix-findings")}), review.Report)
			presentationErr = nil
			if err != nil && ctx.Err() != nil {
				presentationErr = w.Log.CompleteAgent("fix-findings cancelled")
			} else if err != nil {
				presentationErr = w.Log.CompleteAgent("fix-findings failed")
			} else {
				presentationErr = w.Log.CompleteAgent("fix-findings completed")
			}
			livenessErr = liveness.Stop()
			cancelStage()
			if livenessErr != nil {
				w.diagnostic("write liveness", livenessErr)
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			if presentationErr != nil {
				w.diagnostic("render interactive view", presentationErr)
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			if err != nil && ctx.Err() != nil {
				return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
			}
			if ctx.Err() != nil {
				return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
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
					if ctx.Err() != nil {
						return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
					}
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
				if ctx.Err() != nil {
					return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
				}
				w.diagnostic("repository status failed", err)
				return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
			}
			if !hasChanges && !checkpointed {
				return w.complete("success", ExitSuccess, now().Sub(runStarted))
			}
		}

		if w.Repository == nil {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		stageStarted = now()
		if !w.emit("stage_started", event.F("stage", "publish")) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		publication, err := w.Repository.Publish(ctx, initialWorktreeClean)
		if err != nil {
			if ctx.Err() != nil {
				return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
			}
			_ = w.emitPublicationSteps(publication, "failed")
			_ = w.emit("stage_completed", event.F("stage", "publish"), event.F("status", "failed"), durationField(now().Sub(stageStarted)))
			w.diagnostic("publication failed", err)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		if !w.emitPublicationSteps(publication, "") || !w.emit("stage_completed", event.F("stage", "publish"), event.F("status", "success"), durationField(now().Sub(stageStarted))) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		stageStarted = now()
		if !w.emit("stage_started", event.F("stage", "ci"), event.F("head", publication.Head)) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		ciCtx, cancelCI := context.WithTimeout(ctx, w.Config.CITimeout)
		ci, err := w.Repository.WaitCI(ciCtx, publication)
		cancelCI()
		if err != nil {
			if ctx.Err() != nil {
				return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
			}
			w.diagnostic("CI polling failed", err)
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		if !w.emit("step_completed", event.F("stage", "ci"), event.F("step", "ci"), event.F("status", string(ci))) || !w.emit("stage_completed", event.F("stage", "ci"), event.F("status", string(ci)), durationField(now().Sub(stageStarted))) {
			return w.complete("operational_failure", ExitOperational, now().Sub(runStarted))
		}
		switch ci {
		case repository.CISuccess, repository.CISkipped:
			return w.complete("success", ExitSuccess, now().Sub(runStarted))
		case repository.CITimeout:
			return w.complete("ci_timeout", ExitOperational, now().Sub(runStarted))
		case repository.CIFailed:
			checkpointed, lastCheckpoint, checkpointSkipReason = false, repository.Checkpoint{}, ""
			if recoveries >= w.Config.MaxCIRecoveries {
				return w.complete("ci_failure", ExitCI, now().Sub(runStarted))
			}
			if w.runFixCI(ctx, phase, cycle, now) != nil {
				if ctx.Err() != nil {
					return w.complete("cancelled", ExitInterrupted, now().Sub(runStarted))
				}
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

func (w *Workflow) emitPublicationSteps(result repository.Publication, fallback string) bool {
	for _, step := range []struct{ name, status string }{{"commit", result.Commit}, {"push", result.Push}, {"change_request", result.ChangeRequest}} {
		if step.status == "" {
			step.status = fallback
		}
		if step.status == "" {
			step.status = "unknown"
		}
		if !w.emit("step_completed", event.F("stage", "publish"), event.F("step", step.name), event.F("status", step.status)) {
			return false
		}
	}
	return true
}

func (w *Workflow) runFixCI(ctx context.Context, phase, cycle int, now func() time.Time) error {
	started := now()
	if !w.emit("stage_started", event.F("stage", "fix-ci"), event.F("model", w.stageModel("fix-ci")), event.F("reasoning_effort", w.stageReasoningEffort("fix-ci")), intField("review_phase", phase)) {
		return fmt.Errorf("emit CI-fix start")
	}
	err := w.Agent.FixCI(runner.WithStageContext(ctx, runner.StageContext{Stage: "fix-ci", ReviewPhase: phase, Cycle: cycle, Model: w.stageModel("fix-ci"), ReasoningEffort: w.stageReasoningEffort("fix-ci")}))
	status := "success"
	if err != nil {
		status = "failed"
	}
	if !w.emit("stage_completed", event.F("stage", "fix-ci"), event.F("model", w.stageModel("fix-ci")), event.F("reasoning_effort", w.stageReasoningEffort("fix-ci")), intField("review_phase", phase), event.F("status", status), durationField(now().Sub(started))) {
		return fmt.Errorf("emit CI-fix completion")
	}
	if err != nil {
		w.diagnostic("CI fix failed", err)
	}
	return err
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
	case "fix-ci":
		if w.Config.CIFixEffort != "" {
			return w.Config.CIFixEffort
		}
		return "agent-default"
	default:
		return "unknown"
	}
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
