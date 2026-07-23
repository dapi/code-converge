---
title: "FT-024: Local checkpoints for findings fixes"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem and verification owner for retaining successful fix-findings work without publishing intermediate commits."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/24
status: active
delivery_status: in_progress
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-024: Local checkpoints for findings fixes

## What

### Problem

When the review-fix budget is exhausted, successful prior repairs can remain only as uncommitted worktree changes. The terminal result does not clearly distinguish that state from clean finalization. Publishing each repair would create unnecessary remote churn; only the clean-review finalization should push, create a change request, and wait for CI.

### Outcome

- `MET-01`: Every successful `fix-findings` stage that changes an initially clean worktree creates one local checkpoint commit before verification review.
- `MET-02`: A checkpointed clean review enters finalization even when the worktree is now clean, so the current branch is pushed exactly at finalization.
- `MET-03`: Exit `1` names exhausted budget, skipped finalization, and the latest local checkpoint state.

### Scope

- `REQ-01` Require a clean worktree before an automated findings-fix stage; fail closed before the agent runs when pre-existing work could be captured.
- `REQ-02` After a successful fix, skip empty commits; otherwise stage and create the stable local commit `chore: checkpoint review fixes`.
- `REQ-03` Do not push a checkpoint. Push, change-request creation, and CI remain finalization responsibilities after a clean review.
- `REQ-04` Run finalization after a clean review when this run created a checkpoint, even if Git status has no remaining worktree changes; tell the finalizer not to create an empty commit.
- `REQ-05` Fail operationally and do not start another review if checkpoint preparation or commit fails.
- `REQ-06` On `findings_remaining`, report the exhausted budget, skipped finalization, and local checkpoint status, branch, and short commit when available in both human and `kv` output.
- `REQ-07` Cover changed, no-change, dirty-worktree, checkpoint-failure, exhausted-budget, finalization hand-off and output paths with deterministic fakes.
- `REQ-08` Update the root public contract and directly dependent current-state Memory Bank documents; `make docs-lint` passes.

### Non-Scope

- `NS-01` Pushing after each fix, creating or updating a change request before clean review, or waiting for CI before finalization.
- `NS-02` Publishing pre-existing staged, unstaged, or untracked worktree changes.
- `NS-03` New CLI configuration for the checkpoint message; the stable built-in message is sufficient.
- `NS-04` Changing review/fix budgets, finalization verdicts, exit-code meanings, or CI recovery semantics.

### Constraints / Assumptions

- `CON-01` The root [README](../../../README.md) owns the public CLI, stdout and exit contracts.
- `CON-02` A clean pre-fix worktree is the safety boundary that makes `git add -A` attributable to the agent's fix stage.
- `CON-03` A local checkpoint is recoverable progress, not proof of a clean review or successful delivery.

### Design Requirement Decision

| Decision | Rationale | Owner |
| --- | --- | --- |
| Design required: yes | The change adds Git mutation timing, a workflow transition, a worktree safety boundary, and public terminal fields. | [design.md](design.md) |

### Validation Profile Decision

Validation profile: `standard`

Triggers / rationale: executable workflow, Git mutation timing and public stdout contract change; no security, persistent-data, concurrency, deployment, or cross-system protocol trigger applies.

Downgrade approval: none.

## Verify

### Acceptance Scenarios

| Scenario | Requirements | Observable result |
| --- | --- | --- |
| `SC-01` Changed fix | `REQ-01`–`REQ-03` | A clean worktree changes during a successful fix; one local commit is made and no `git push` is invoked before the next review. |
| `SC-02` No-change fix | `REQ-02` | A successful fix with no status change makes no commit. |
| `SC-03` Clean after checkpoint | `REQ-04` | A verification review is clean; finalization runs despite an otherwise clean worktree and receives checkpoint context. |
| `SC-04` Exhausted budget | `REQ-06` | Exit `1` states budget exhaustion and skipped finalization; `kv` includes local checkpoint branch/commit when one exists. |
| `SC-05` Safety and failures | `REQ-01`, `REQ-05` | Dirty initial status prevents fixing; status, staging, commit or identifier failure exits `2` without another review. |

### Checks and Evidence

| Check | Covers | Evidence |
| --- | --- | --- |
| `CHK-01` | `SC-01`–`SC-05` deterministic workflow and repository regression tests | `EVID-01` |
| `CHK-02` | Public README and Memory Bank contract consistency | `EVID-02` |
| `CHK-03` | Full Go, vet, documentation and diff validation | `EVID-03` |

| Evidence | Carrier |
| --- | --- |
| `EVID-01` | `go test ./internal/repository ./internal/workflow ./internal/codex ./internal/event` output |
| `EVID-02` | `make docs-lint` output and semantic owner read-through |
| `EVID-03` | `go test ./...`, `go vet ./...`, `git diff --check` output |
