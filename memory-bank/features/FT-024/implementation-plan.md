---
title: "FT-024: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Grounded execution plan for FT-024."
derived_from:
  - brief.md
  - design.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_024_scope
  - ft_024_selected_design
  - ft_024_acceptance_criteria
  - ft_024_blocker_state
  - ft_024_validation_profile
---

# FT-024: Implementation Plan

## Grounding

| Path | Role |
| --- | --- |
| `internal/workflow/workflow.go` | Review/fix/finalize state machine and terminal events. |
| `internal/repository/status.go` | Existing fakeable porcelain status boundary. |
| `internal/codex/adapter.go` | Finalization prompt and strict result carrier. |
| `internal/event/event.go` | Human rendering for public `kv` events. |

## Test Strategy

| Surface | Refs | Automated coverage | Required commands |
| --- | --- | --- | --- |
| Repository Git sequence | `REQ-01`–`REQ-03`, `SC-01`, `SC-02`, `SC-05` | Scripted runner proves clean check, no empty commit, local add/commit, branch/SHA, and no push. | `go test ./internal/repository` |
| Workflow transition | `REQ-04`–`REQ-06`, `SC-03`–`SC-05` | Fake agent/repository tests for hand-off, failures, budget exhaustion, and output fields. | `go test ./internal/workflow ./internal/event ./internal/codex` |
| Documentation | `REQ-08`, `CHK-02` | Lint and semantic owner read-through. | `make docs-lint` |

## Preconditions

| ID | Required state |
| --- | --- |
| `PRE-01` | `brief.md` and `design.md` are active; `standard` profile governs validation. |

## Design Realization Mapping

| Design refs | Target | Steps | Checks |
| --- | --- | --- | --- |
| `SOL-01`, `SD-01`–`SD-03`, `CTR-01`, `FM-01` | repository status boundary | `STEP-01` | `CHK-01` |
| `SOL-02`, `SOL-03`, `INV-01`–`INV-03`, `FM-02` | workflow and Codex adapter | `STEP-02` | `CHK-01` |
| `SOL-04` | event renderer and README | `STEP-03` | `CHK-02` |

## Steps

| Step | Implements | Action | Verifies | Evidence |
| --- | --- | --- | --- | --- |
| `STEP-01` | `REQ-01`–`REQ-03` | Add local-only repository checkpoint primitives and deterministic runner tests. | `CHK-01` | `EVID-01` |
| `STEP-02` | `REQ-04`, `REQ-05` | Add workflow state, finalizer checkpoint context, and failure paths. | `CHK-01` | `EVID-01` |
| `STEP-03` | `REQ-06`, `REQ-08` | Add terminal rendering and converge public/current-state docs. | `CHK-02` | `EVID-02` |
| `STEP-04` | all | Run functional validation, simplify review, then acceptance audit. | `CHK-03` | `EVID-03` |

## Checkpoints

| Checkpoint | Condition | Evidence |
| --- | --- | --- |
| `CP-01` | A checkpoint is local-only and all post-fix failure paths stop before review. | `EVID-01` |
| `CP-02` | Clean checkpointed review finalizes once; exhausted budget is explicit. | `EVID-01`, `EVID-02` |
| `CP-03` | Full required local validation is green. | `EVID-03` |
