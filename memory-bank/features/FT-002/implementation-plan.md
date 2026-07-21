---
title: "FT-002: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Execution plan for implementing, validating and publishing the complete Reviewer CLI against the active brief and design."
derived_from:
  - brief.md
  - design.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_002_scope
  - ft_002_selected_design
  - ft_002_acceptance_criteria
  - ft_002_blocker_state
  - ft_002_validation_profile
---

# FT-002: Implementation Plan

## Goal And Grounding

Deliver the single Go CLI and deterministic distribution evidence. Current repository has only specifications, a documentation Makefile, and Memory Bank CI; there is no Go module or runtime code to preserve. Canonical owners are `brief.md` and `design.md`; no support artifact overrides them.

| Path | Current role | Reuse / action |
| --- | --- | --- |
| `README.md` | Public CLI contract | Treat as acceptance SSoT; add only installation/build facts selected in `SD-04`. |
| `memory-bank/engineering/architecture.md` | Conceptual responsibility boundaries | Realize its five responsibilities as small Go packages. |
| `memory-bank/engineering/testing-policy.md` | Required test surfaces | Implement fakes, fixtures, table/golden tests; no live external side effects. |
| `.github/workflows/memory-bank-lint.yml` | Existing required CI | Extend CI to Go test/vet, docs lint and distribution smoke. |

## Open Questions / Ambiguities

`none`; former blockers are resolved by `decision-log.md` `DL-03` and owned as `SD-01`–`SD-05`.

## Environment Contract

Go `1.21.13+`, Git, and authenticated GitHub are needed for delivery. Runtime needs `codex` and a Git repository. Tests replace Codex/remotes/CI with fakes. Commands: `go test ./...`, `go vet ./...`, `make docs-lint`, `make dist`, `git diff --check`.

## Preconditions

| ID | Canonical ref | Required state | Used by | Blocks start |
| --- | --- | --- | --- | --- |
| `PRE-01` | `brief.md`, `design.md`, `DL-03` | active owners and resolved gates | all steps | yes |

## Test Strategy

| Surface | Canonical refs | Planned automated coverage | Commands / CI | Manual gap |
| --- | --- | --- | --- | --- |
| Config/CLI | `REQ-01`, `FM-02` | full precedence, invalid values/paths, config output | `go test ./...` | none |
| Codex boundary | `REQ-02`, `REQ-04`, `CTR-01`–`CTR-04` | fake runner, parser corpus, schema and consistency | `go test ./...` | real Codex compatibility is bounded by fixtures and final independent review |
| Workflow/events | `REQ-03`–`REQ-06` | all transitions/exits, counters, golden record validation | `go test ./...` | none |
| Distribution | `REQ-08`, `RB-01` | two-build checksum comparison, archive/install smoke | `make dist`, CI smoke | none |

## Design Realization Mapping

| Solution refs | Target | Steps | Checks | Evidence |
| --- | --- | --- | --- | --- |
| `SOL-01`, `C4-01`, `SD-03` | `cmd/`, `internal/config`, `internal/runner` | `STEP-01`, `STEP-02` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `SOL-02`, `SD-01`, `SD-02`, `CTR-01`–`CTR-04`, `INV-01`, `INV-03` | `internal/codex`, `internal/event` | `STEP-02`, `STEP-03` | `CHK-02`, `CHK-03` | `EVID-02`, `EVID-03` |
| `SOL-03`, `INV-02`, `INV-04`, `FM-01`–`FM-04` | `internal/workflow` | `STEP-04` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `SOL-04`, `SD-04`, `SD-05`, `FM-05`, `RB-01`, `RB-02` | Makefile, build helper, CI, README | `STEP-05`, `STEP-06` | `CHK-04`–`CHK-06` | `EVID-04`–`EVID-06` |

## Workstreams And Steps

| Step | Implements | Goal | Touchpoints | Verifies | Evidence | Blocked by |
| --- | --- | --- | --- | --- | --- | --- |
| `STEP-01` | `REQ-01` | Bootstrap module, CLI and config resolver | `go.mod`, `cmd/reviewer`, `internal/config` | `CHK-01` | `EVID-01` | `PRE-01` |
| `STEP-02` | `REQ-02`, `REQ-04` | Process runner and fail-closed Codex adapter | `internal/runner`, `internal/codex`, fixtures | `CHK-02` | `EVID-02` | `STEP-01` |
| `STEP-03` | `REQ-06` | Stable event renderer | `internal/event` | `CHK-03` | `EVID-03` | `STEP-01` |
| `STEP-04` | `REQ-03`–`REQ-05` | Complete orchestration and exits | `internal/workflow`, CLI wiring | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` | `STEP-02`, `STEP-03` |
| `STEP-05` | `REQ-07`, `REQ-08` | Tests, deterministic artifacts, docs and CI | tests, scripts, Makefile, README, workflow | `CHK-04`–`CHK-06` | `EVID-04`–`EVID-06` | `STEP-04` |
| `STEP-06` | all | Independent review, fixes, commit, push and PR | complete diff | `CHK-04`–`CHK-06` | PR/CI/review links | `STEP-05` |

No code workstream writes the same package in parallel. Approval is required only for an actual external release or production/live-data action; this plan creates a draft/ready PR and local/CI artifacts only.

## Checkpoints / Stop Conditions

- `CP-01` parsers and config pass deterministic negative tests before orchestration is trusted.
- `CP-02` every terminal state has event/exit evidence before distribution work.
- `CP-03` two independent distribution builds produce identical checksums and each target archive contains one expected binary.
- `STOP-01` unknown agent output, missing required CI, merge conflict, or critical/high review finding stops publication readiness and returns to the owning step.

## Execution Risks

- `ER-01` Codex prose drift: mitigated by fixture-gated fail-closed compatibility.
- `ER-02` finalization schema support differs by Codex version: invocation tests pin the required flags and runtime errors remain operational failures.
- `ER-03` archive normalization differs by host: implemented in Go rather than host `tar`.

## Ready For Acceptance

All steps/checkpoints complete; required local/CI suites green; no manual-only critical gap; independent review has no critical/high finding; final acceptance uses `brief.md#verify`.
