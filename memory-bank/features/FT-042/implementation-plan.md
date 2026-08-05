---
title: "FT-042: Implementation plan"
doc_kind: feature
doc_function: derived
purpose: "Execution and verification mapping for configurable document-review prompts."
derived_from:
  - brief.md
  - design.md
  - ../../engineering/testing-policy.md
status: active
audience: humans_and_agents
---

# FT-042: Implementation plan

## Grounding

| Path | Role | Reuse |
| --- | --- | --- |
| `internal/app/app.go` | root flags, commands and error/event dispatch | command dispatch and `OptionalString` binding pattern |
| `internal/config/config.go` | prompt-file loading and source diagnostics | explicit file validation/read error pattern only |
| `internal/repository/review.go` | private snapshot lifecycle | derive changed paths from the pinned snapshot |
| `internal/codex/adapter.go` | schema-constrained review and fix stdin | preserve schema/scope safety prefix and model invocation |
| `internal/*/*_test.go` | deterministic fakes | table-driven runner/command tests |

Open questions: none; issue-comment decisions and `design.md` settle CLI semantics.

## Test strategy

| Surface | Refs | Automated coverage | Commands |
| --- | --- | --- | --- |
| selector/resolver/export | `REQ-01`, `REQ-02`, `REQ-05`, `SOL-01`–`SOL-03` | conflicts, relative and absolute Markdown paths (including outside repo), names, fallback, create/force/failure | `go test ./internal/app ./internal/config` |
| snapshot and adapter | `REQ-03`, `REQ-04`, `SOL-04`, `SOL-05` | Markdown filtering, excluded paths, empty scope, prompt/schema composition | `go test ./internal/repository ./internal/codex` |
| workflow fix routing | `REQ-06`, `SOL-06` | built-in/custom document fix and unchanged ordinary fix | `go test ./internal/workflow ./internal/codex` |
| public docs | `REQ-07` | help/config examples and links | `make docs-lint` |
| full contract | all | regression and hygiene | `go test ./...`; `go vet ./...`; `git diff --check` |

Manual-only gaps: none. Required CI: repository Verify workflow for published head.

## Preconditions

| ID | Ref | State |
| --- | --- | --- |
| `PRE-01` | `SD-01`, `INV-01`–`INV-04` | Feature package is Plan Ready and no external approval is required for local code/doc changes. |

## Design realization mapping

| Refs | Target | Steps | Checks | Evidence |
| --- | --- | --- | --- | --- |
| `SOL-01`–`SOL-03`, `SD-01`, `FM-01`–`FM-02` | app/config command and resolver | `STEP-01` | `CHK-01` | `EVID-01` |
| `SOL-04`, `SOL-05`, `CTR-01`, `INV-02`–`INV-04`, `FM-03` | repository and Codex adapter | `STEP-02` | `CHK-01` | `EVID-01` |
| `SOL-06`, `INV-01`, `FM-02` | workflow/adapter fix wiring | `STEP-03` | `CHK-01` | `EVID-01` |
| `TRD-01`, `RB-01` | README and package evidence | `STEP-04` | `CHK-03` | `EVID-03` |

## Steps

| ID | Implements | Work | Verifies | Evidence |
| --- | --- | --- | --- | --- |
| `STEP-01` | `REQ-01`, `REQ-02`, `REQ-05` | Add selectors, safe resolution and export command with deterministic errors. | `CHK-01` | `EVID-01` |
| `STEP-02` | `REQ-03`, `REQ-04` | Add snapshot file filtering, built-in/default prompt composition and empty-scope completion. | `CHK-01` | `EVID-01` |
| `STEP-03` | `REQ-06` | Route document findings to the selected/built-in fix prompt without changing ordinary fix. | `CHK-01` | `EVID-01` |
| `STEP-04` | `REQ-07` | Update README/help contract and feature artifacts. | `CHK-03` | `EVID-03` |
| `STEP-05` | all | Run focused and full validation, simplify review, independent implementation review, commit/push and CI. | `CHK-02`–`CHK-04` | `EVID-02`–`EVID-04` |

## Checkpoints and risks

- `CP-01`: `STEP-01` and `STEP-02` pass focused tests before workflow wiring.
- `CP-02`: all changed behavior passes full local validation before independent review.
- `ER-01`: private snapshot filtering may not see untracked Markdown files; stop and return to `design.md` if it cannot preserve the current snapshot contract.
- `ER-02`: a public event-schema change is discovered; stop and update `brief.md`/`design.md` before implementation.
- `STOP-01`: any new security, persistent-data, cross-system, or release trigger raises the validation profile before further code changes.
