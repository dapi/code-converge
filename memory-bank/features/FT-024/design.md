---
title: "FT-024: Local checkpoints for findings fixes — Design"
doc_kind: feature
doc_function: canonical
purpose: "Selected workflow and Git safety design for FT-024."
derived_from:
  - brief.md
  - ../../engineering/architecture.md
  - ../../../README.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_024_scope
  - ft_024_acceptance_criteria
  - ft_024_validation_profile
---

# FT-024: Design

## Selected Design

- `SOL-01`: `internal/repository.Status` owns the Git boundary: porcelain status, clean-precondition, local `git add -A` / `git commit`, and branch/short-SHA lookup. It never pushes.
- `SOL-02`: `internal/workflow.Workflow` checks checkpoint eligibility before `fix-findings`, continues remediation on a dirty baseline without checkpointing, checkpoints only a successful clean-baseline fix, and fails closed on repository-operation errors.
- `SOL-03`: The workflow remembers a checkpoint only until finalization. A clean review then enters finalization even with no remaining worktree changes; after a `CI_FAILED` publication result, checkpoint metadata is cleared before the next review phase so terminal output cannot call an already-pushed commit local.
- `SOL-04`: `run_completed findings_remaining` carries machine-safe `checkpoint_status`, plus `checkpoint_branch` and `checkpoint_commit` for a local commit; human rendering expands that state into an unambiguous terminal sentence.

## C4 Applicability Decision

- `C4-00`: not required. The change stays inside existing workflow, repository, Codex adapter, and event-rendering components; it creates no new runtime or external-system boundary.

## Architecture Coverage Decision

| Aspect | Result |
| --- | --- |
| Components | covered: workflow orchestrates; repository executes local Git; Codex adapter finalizes; event renders. |
| Connectors | covered: synchronous local `git` commands and Codex invocation; no checkpoint push connector exists. |
| Configuration | N/A: the fixed checkpoint message introduces no setting. |
| Behavioral semantics | covered by `CTR-01` and `INV-01`–`INV-03`. |
| Quality/evolution | covered by fail-closed errors and deterministic fakes. |

## Decisions, Contracts, and Failure Modes

- `SD-01`: The safety boundary is a clean porcelain status immediately before each automatic findings-fix invocation. A dirty worktree does not prevent remediation; it disables the checkpoint so pre-existing user work cannot be committed automatically.
- `SD-02`: The stable checkpoint message is `chore: checkpoint review fixes`; an empty post-fix status does not commit.
- `SD-03`: Checkpoints remain local. Only finalization pushes or coordinates external systems.
- `CTR-01`: A checkpoint result is either no commit or `{branch, commit}` after a successful local commit. It is never a finalization verdict.
- `INV-01`: No verification review follows a failed checkpoint operation.
- `INV-02`: No automatic checkpoint contains changes present before the corresponding fix stage.
- `INV-03`: A run can succeed after checkpointing only through a clean review and normal finalization success.
- `FM-01`: Status, staging, commit, branch, or SHA lookup failure → diagnostic plus `operational_failure` / exit `2`.
- `FM-02`: Findings after the budget → exit `1`; finalization is explicitly not reached and the latest checkpoint state is rendered.

## Risk-Based Design Verification

| Risk | Required | Method and result |
| --- | --- | --- |
| Contract compatibility | yes | Golden event and workflow tests cover new terminal fields and human text. |
| State transitions | yes | Table-like fake workflow paths cover changed/no-change/failure/exhaustion/finalization. |
| Failure propagation | yes | Repository runner errors stop before another review. |
| Concurrency, security, capacity, migration | no | No new concurrent execution, trust boundary, load path, data format, or migration. |
