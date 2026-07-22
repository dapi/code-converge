---
title: "FT-009: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Grounded execution plan for human/kv rendering, liveness coordination, public contract documentation and high-risk validation."
derived_from:
  - brief.md
  - design.md
status: archived
audience: humans_and_agents
must_not_define:
  - ft_009_scope
  - ft_009_selected_design
  - ft_009_acceptance_criteria
  - ft_009_validation_profile
---

# FT-009: Implementation Plan

## Цель текущего плана

Implement the accepted additive human progress format and bounded liveness behavior while preserving the existing `kv` default, workflow semantics, diagnostics boundary and raw Codex-output isolation.

## Grounding / Support References

| Document | Role | Facts reused |
| --- | --- | --- |
| `brief.md` | Canonical problem/verify owner | `REQ-*`, `SC-*`, `NEG-*`, `CHK-*`, `EVID-*`, high-risk validation decision |
| `design.md` | Canonical solution owner | `SOL-*`, `C4-03`, `SD-*`, `CTR-*`, `INV-*`, `FM-*`, `RB-*` |
| `decision-log.md` | Provenance | `DL-01`–`DL-08`, confidence and review cycle history |
| `../../../README.md` | Public contract owner | Current CLI/config precedence, event catalog, stdout/stderr and exit-code semantics |
| `../../engineering/testing-policy.md` | Verification policy | Fake/deterministic execution, stdout goldens and required repository checks |

## Discovery Context

### Relevant paths and local patterns

| Path | Current role / pattern | Planned realization target |
| --- | --- | --- |
| `internal/event/event.go`, `event_test.go` | One synchronous `Logger.Emit` encodes stable `key=value` lines and propagates writer errors | Retain kv encoder; add typed presentation/rendering, duration/count helpers, terminal/liveness coordinator and deterministic tests |
| `internal/workflow/workflow.go`, `workflow_test.go` | Sequential state machine emits event facts around four Codex-backed stages; owns durations and terminal exit codes | Emit through the selected presenter and open/close stage liveness scopes without changing transitions |
| `internal/app/app.go`, `app_test.go` | Parses flags, wires stdout/stderr/config/clock/runner and emits startup/config failures | Bind new overrides, supply terminal capability and presentation dependencies, cover pre-workflow failures in both formats |
| `internal/config/config.go`, `config_test.go` | Single resolver and `code-converge config` source metadata | Add/validate `log-format`, `heartbeat`, `color` with existing precedence/display pattern |
| `internal/codex/*`, `internal/runner/*` | Captures raw process streams and propagates context cancellation | Preserve boundary; extend only integration fixtures needed for `REQ-07` |
| `README.md` and dependent Memory Bank docs | Public CLI/stdout owner plus derived architecture/product/ops claims | Publish exact selected contract and reconcile derived references |

### Test surfaces

- Configuration source/precedence/display and invalid-value tables.
- Human and unchanged kv renderer golden catalog, singular/plural counts and duration boundaries.
- Workflow positive and every terminal/error path in both formats.
- Injectable TTY/color capability, monotonic clock/ticker, heartbeat interval, completion/tick races, cancellation and failing writers.
- `go test -race` for the changed event/workflow packages.
- Existing Codex capture/isolation, full Go suite, vet, docs lint, diff check and required CI.

### Execution environment

- Go 1.21.13+ and existing Make targets; no real Codex session, remote mutation or hosted-CI wait in local tests.
- Tests use fake agents/runners/writers/clocks/terminals. The implementation must not depend on wall-clock sleeps.
- `NO_COLOR`, `TERM` and `COLORTERM` are controlled fixture inputs, not assumptions about the developer terminal.

### Open Questions / Ambiguities

None. `AG-01` was approved by the user's 2026-07-22 instruction to implement issue #9 end-to-end and publish the PR.

## Test Strategy

| Surface | Required evidence | Commands / procedure |
| --- | --- | --- |
| Config/app contract | Source precedence, defaults, validation, display, early failures | `go test ./internal/config ./internal/app` |
| Renderer/workflow contract | Exhaustive human goldens, unchanged kv goldens, every terminal path | `go test ./internal/event ./internal/workflow` |
| Concurrency/output safety | Deterministic stage/tick/cancel/write tests and race detector | `go test -race ./internal/event ./internal/workflow` |
| Raw-output isolation | Existing and extended fake-runner/adapter tests | `go test ./internal/codex ./internal/runner ./internal/workflow ./internal/app` |
| Documentation | Public/derived contract convergence and link/frontmatter checks | `make docs-lint`; semantic read-through of README versus design/brief |
| Full regression | Complete local repository floor and required CI | `go test ./...`; `go vet ./...`; `git diff --check`; required CI URL |

## Preconditions

| ID | Required state | Blocks |
| --- | --- | --- |
| `PRE-01` | `brief.md` and `design.md` are active, and issue #9 links this package | all implementation steps |
| `PRE-02` | `AG-01` records human approval to begin the high-risk concurrency/contract change | satisfied before `STEP-01` |
| `PRE-03` | Working tree changes are inventoried so unrelated user edits are preserved | first source edit |

## Design Realization Mapping

| Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- |
| `SOL-01`, `SOL-02`, `SD-01`, `SD-03`, `SD-04`, `SD-09`, `CTR-01`, `INV-02`, `FM-01`, `FM-05`, `RB-02` | `STEP-01` | `CHK-01`, `CHK-05` | `EVID-01`, `EVID-05` |
| `SOL-03`, `SD-02`, `SD-06`, `SD-07`, `SD-08`, `CTR-02`, `CTR-03`, `CTR-05`, `INV-01`, `FM-07` | `STEP-02` | `CHK-02` | `EVID-02` |
| `C4-03`, `SOL-04`, `SOL-05`, `SOL-06`, `SD-05`, `CTR-04`, `INV-03`, `INV-04`, `INV-05`, `INV-07`, `FM-02`, `FM-03`, `FM-04`, `FM-06`, `FM-08` | `STEP-03`, `STEP-04` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` |
| `INV-06` | `STEP-04` | `CHK-04` | `EVID-04` |
| `RB-01` | `STEP-05`, `STEP-06` | `CHK-05`, `CHK-06` | `EVID-05`, `EVID-06` |

## Порядок работ

| Step | Implements | Touchpoints | Verifies |
| --- | --- | --- | --- |
| `STEP-01` | `REQ-01`, `REQ-05`, `REQ-08`; `SOL-01`, `SOL-02`, `SD-09`, `CTR-01` | `internal/config`, `internal/app`, config and startup-failure fixtures | `CHK-01` |
| `STEP-02` | `REQ-02`, `REQ-03`; `SOL-03`, `CTR-02`, `CTR-03`, `CTR-05` | `internal/event`, workflow event adapter/goldens | `CHK-02` |
| `STEP-03` | `REQ-04`, `REQ-05`; `SOL-04`, `SD-03`–`SD-05`, `CTR-04` | terminal capability, shimmer, heartbeat and injected clock/ticker inside presentation boundary | `CHK-03` |
| `STEP-04` | `REQ-06`, `REQ-07`; `SOL-05`, `SOL-06`, `INV-03`–`INV-07` | workflow stage scopes, coordinator, cancellation/failing writers, adapter isolation fixtures | `CHK-03`, `CHK-04` |
| `STEP-05` | `REQ-08`; resolve the current machine-readable-only upstream wording when the new contract is delivered | root README first, then architecture/product/ops/PRD/feature references that depend on it | `CHK-05` |
| `STEP-06` | All requirements; high-risk convergence and evidence | full suite, race run, vet, docs lint, diff check, required CI and evidence records | `CHK-06` plus closure of `CHK-01`–`CHK-05` |

## Approval Gates

- `AG-01` Approved by the user's 2026-07-22 instruction to implement issue #9 end-to-end, validate it, publish a PR and drive it to readiness.
- `AG-02` Any manual-only critical-path gap discovered during execution requires a named human approver and procedure before acceptance. Current gaps: none.

## Checkpoints

| Checkpoint | Condition | Evidence |
| --- | --- | --- |
| `CP-01` | New settings resolve and invalid combinations fail before Codex | `EVID-01` |
| `CP-02` | Human catalog and unchanged kv baseline are exhaustive | `EVID-02` |
| `CP-03` | TTY/no-color/non-TTY/heartbeat behavior is deterministic and race-clean | `EVID-03` |
| `CP-04` | Cancellation, all writer failures and raw-output isolation pass | `EVID-04` |
| `CP-05` | Public and derived docs converge | `EVID-05` |
| `CP-06` | Full local/CI validation and independent final convergence pass are green | `EVID-06` |

## Stop Conditions / Fallback

- Stop and update `design.md` before code if implementation needs a different format default, public setting, human wording, stage ordering or writer-failure policy.
- Stop and request human decision if terminal capability APIs cannot support the selected behavior without a new dependency or platform contract.
- Stop on any changed `kv` golden unrelated to deliberately selecting the format; preserve/revert to the existing encoder contract.
- Stop if race tests expose a design gap in stop/join or diagnostic ordering; do not mask it with sleeps.
- Fall back operationally to explicit `kv` selection and revert `RB-01` if human presentation causes release regressions.

## Готово для приемки

All `CHK-*` have concrete passing `EVID-*`, `go test -race` and the full required CI set are green, the independent high-risk convergence review has no open critical/important findings, public/derived docs agree, and rollback remains explicit `kv` selection or release revert.
