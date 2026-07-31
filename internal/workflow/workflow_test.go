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

type workflowAgent struct {
	reviews  []codex.ReviewResult
	fixes    int
	fixErr   error
	ciFixes  int
	ciFixErr error
}

func (a *workflowAgent) Review(context.Context) (codex.ReviewResult, error) {
	if len(a.reviews) == 0 {
		return codex.ReviewResult{}, errors.New("missing review result")
	}
	result := a.reviews[0]
	a.reviews = a.reviews[1:]
	return result, nil
}
func (a *workflowAgent) FixFindings(context.Context, string) error {
	a.fixes++
	return a.fixErr
}
func (a *workflowAgent) FixCI(context.Context) error {
	a.ciFixes++
	return a.ciFixErr
}

type workflowRepository struct {
	changes     []bool
	clean       []bool
	publication repository.Publication
	publishErr  error
	ci          []repository.CIResult
	ciErr       error
	publishes   int
	ciWaits     int
}

func (r *workflowRepository) next(values []bool) bool {
	if len(values) == 0 {
		return false
	}
	value := values[0]
	if len(values) > 1 {
		r.changes = values[1:]
	}
	return value
}
func (r *workflowRepository) HasChanges(context.Context) (bool, error) {
	if len(r.changes) == 0 {
		return false, nil
	}
	value := r.changes[0]
	r.changes = r.changes[1:]
	return value, nil
}
func (r *workflowRepository) IsClean(context.Context) (bool, error) {
	if len(r.clean) == 0 {
		return true, nil
	}
	value := r.clean[0]
	r.clean = r.clean[1:]
	return value, nil
}
func (*workflowRepository) Head(context.Context) (string, error) { return "head", nil }
func (*workflowRepository) Checkpoint(context.Context, string, bool) (repository.Checkpoint, error) {
	return repository.Checkpoint{}, nil
}
func (r *workflowRepository) Publish(context.Context, bool) (repository.Publication, error) {
	r.publishes++
	if r.publishErr != nil {
		return repository.Publication{}, r.publishErr
	}
	if r.publication.Head == "" {
		r.publication = repository.Publication{Commit: "skipped", Push: "success", ChangeRequest: "success", Head: "published"}
	}
	return r.publication, nil
}
func (r *workflowRepository) WaitCI(context.Context, repository.Publication) (repository.CIResult, error) {
	r.ciWaits++
	if r.ciErr != nil {
		return "", r.ciErr
	}
	if len(r.ci) == 0 {
		return repository.CISuccess, nil
	}
	value := r.ci[0]
	r.ci = r.ci[1:]
	return value, nil
}

func cleanReview() codex.ReviewResult { return codex.ReviewResult{Clean: true} }

func runWorkflow(t *testing.T, cfg config.Config, agent *workflowAgent, repo *workflowRepository) (int, string) {
	t.Helper()
	var output, stderr bytes.Buffer
	now := func() time.Time { return time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC) }
	w := Workflow{Config: cfg, Agent: agent, Repository: repo, Log: &event.Logger{Out: &output, Format: "kv", Now: now}, Err: &stderr, Now: now}
	code := w.Run(context.Background())
	return code, output.String()
}

func TestCleanReviewPublishesAndWaitsForCI(t *testing.T) {
	repo := &workflowRepository{changes: []bool{true}, ci: []repository.CIResult{repository.CISuccess}}
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute}, &workflowAgent{reviews: []codex.ReviewResult{cleanReview()}}, repo)
	if code != ExitSuccess || repo.publishes != 1 || repo.ciWaits != 1 {
		t.Fatalf("code=%d publishes=%d waits=%d", code, repo.publishes, repo.ciWaits)
	}
	for _, want := range []string{"stage=publish", "step=push status=success", "stage=ci", "status=success"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in:\n%s", want, output)
		}
	}
}

func TestNoApplicableCISucceeds(t *testing.T) {
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute}, &workflowAgent{reviews: []codex.ReviewResult{cleanReview()}}, &workflowRepository{changes: []bool{true}, ci: []repository.CIResult{repository.CISkipped}})
	if code != ExitSuccess || !strings.Contains(output, "stage=ci step=ci status=skipped") {
		t.Fatalf("code=%d output=%s", code, output)
	}
}

func TestCIFailureRunsFixThenReviewsAndPublishesAgain(t *testing.T) {
	repo := &workflowRepository{changes: []bool{true, true}, ci: []repository.CIResult{repository.CIFailed, repository.CISuccess}}
	agent := &workflowAgent{reviews: []codex.ReviewResult{cleanReview(), cleanReview()}}
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute, MaxCIRecoveries: 1}, agent, repo)
	if code != ExitSuccess || agent.ciFixes != 1 || repo.publishes != 2 {
		t.Fatalf("code=%d fixes=%d publishes=%d", code, agent.ciFixes, repo.publishes)
	}
	if !strings.Contains(output, "stage=fix-ci") {
		t.Fatalf("missing fix-ci: %s", output)
	}
}

func TestFindingsFixThenCleanPublishes(t *testing.T) {
	findings := codex.ReviewResult{Clean: false, Report: "fix this"}
	agent := &workflowAgent{reviews: []codex.ReviewResult{findings, cleanReview()}}
	repo := &workflowRepository{changes: []bool{true}, ci: []repository.CIResult{repository.CISuccess}}
	code, _ := runWorkflow(t, config.Config{CITimeout: time.Minute, MaxCycles: 1}, agent, repo)
	if code != ExitSuccess || agent.fixes != 1 || repo.publishes != 1 {
		t.Fatalf("code=%d fixes=%d publishes=%d", code, agent.fixes, repo.publishes)
	}
}

func TestCIRecoveryLimitRemainsEffective(t *testing.T) {
	agent := &workflowAgent{reviews: []codex.ReviewResult{cleanReview()}}
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute, MaxCIRecoveries: 0}, agent, &workflowRepository{changes: []bool{true}, ci: []repository.CIResult{repository.CIFailed}})
	if code != ExitCI || agent.ciFixes != 0 || !strings.Contains(output, "status=ci_failure exit_code=3") {
		t.Fatalf("code=%d fixes=%d output=%s", code, agent.ciFixes, output)
	}
}

func TestCIFixFailureStopsRecovery(t *testing.T) {
	agent := &workflowAgent{reviews: []codex.ReviewResult{cleanReview()}, ciFixErr: errors.New("repair failed")}
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute, MaxCIRecoveries: 1}, agent, &workflowRepository{changes: []bool{true}, ci: []repository.CIResult{repository.CIFailed}})
	if code != ExitCI || agent.ciFixes != 1 || !strings.Contains(output, "stage=fix-ci") || !strings.Contains(output, "status=ci_failure exit_code=3") {
		t.Fatalf("code=%d fixes=%d output=%s", code, agent.ciFixes, output)
	}
}

func TestCITimeoutIsOperationalAndDoesNotFix(t *testing.T) {
	agent := &workflowAgent{reviews: []codex.ReviewResult{cleanReview()}}
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute}, agent, &workflowRepository{changes: []bool{true}, ci: []repository.CIResult{repository.CITimeout}})
	if code != ExitOperational || agent.ciFixes != 0 || !strings.Contains(output, "stage_completed stage=ci status=timeout") || !strings.Contains(output, "timeout_ms=60000") || !strings.Contains(output, "status=ci_timeout exit_code=2") {
		t.Fatalf("code=%d fixes=%d output=%s", code, agent.ciFixes, output)
	}
}

func TestUnknownCIResultIsOperational(t *testing.T) {
	agent := &workflowAgent{reviews: []codex.ReviewResult{cleanReview()}}
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute}, agent, &workflowRepository{changes: []bool{true}, ci: []repository.CIResult{"unknown"}})
	if code != ExitOperational || agent.ciFixes != 0 || !strings.Contains(output, "status=operational_failure") {
		t.Fatalf("code=%d fixes=%d output=%s", code, agent.ciFixes, output)
	}
}

func TestPreexistingDirtyWorktreeIsNotCommitted(t *testing.T) {
	repo := &workflowRepository{clean: []bool{false}, changes: []bool{true}, publishErr: errors.New("refuse dirty worktree")}
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute}, &workflowAgent{reviews: []codex.ReviewResult{cleanReview()}}, repo)
	if code != ExitOperational || repo.publishes != 1 || !strings.Contains(output, "status=operational_failure") {
		t.Fatalf("code=%d publishes=%d output=%s", code, repo.publishes, output)
	}
}

func TestCIProviderErrorIsOperationalAndDoesNotFix(t *testing.T) {
	agent := &workflowAgent{reviews: []codex.ReviewResult{cleanReview()}}
	code, output := runWorkflow(t, config.Config{CITimeout: time.Minute}, agent, &workflowRepository{changes: []bool{true}, ciErr: errors.New("authentication failed")})
	if code != ExitOperational || agent.ciFixes != 0 || !strings.Contains(output, "status=operational_failure") {
		t.Fatalf("code=%d fixes=%d output=%s", code, agent.ciFixes, output)
	}
}
