---
title: "FT-014: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Execution plan for the accepted private diagnostic session-record design, grounded in FT-014 canonical owners."
derived_from:
  - brief.md
  - design.md
  - decision-log.md
  - ../../../README.md
  - ../../engineering/testing-policy.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_014_scope
  - selected_solution
  - ft_014_acceptance_criteria
---

# FT-014: Implementation Plan

## Canonical Ownership

| Owner | Owns | Change rule |
| --- | --- | --- |
| `brief.md` | `REQ-*`, `SC-*`, `CHK-*`, `EVID-*`, validation profile | Update first if requirements/evidence change. |
| `design.md` | `SOL-*`, `SD-*`, `C4-*`, `CTR-*`, `INV-*`, `FM-*`, `RB-*` | Update first if solution/failure semantics change. |
| `../../../README.md` | Public CLI, configuration, stdout/stderr and exit contracts | Update atomically with implementation. |

## Preconditions

| Precondition ID | Canonical ref | Required state | Used by steps | Blocks start |
| --- | --- | --- | --- | --- |
| `PRE-01` | `decision-log.md#d-01`, `design.md` | FPF decision accepted and canonical solution active | All steps | yes |
| `PRE-02` | `decision-log.md#d-01` execution authorization | User's end-to-end implementation instruction approves `AG-01` for this bounded local/PR delivery; no production or live-data action is authorized. | `STEP-01`–`STEP-05` | no |

## Open Questions / Ambiguities

none. Discovery must stop and update `design.md` if platform filesystem semantics cannot meet `CTR-02`, `INV-03` or `FM-01`.

## Design Realization Mapping

| Canonical solution refs | Owner | Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- | --- |
| `SOL-01`, `SOL-02`, `SD-01`, `SD-02` | `design.md` | `internal/config`, `internal/app`, root README | `STEP-01`, `STEP-04` | `CHK-02`, `CHK-04` | `EVID-02`, `EVID-04` |
| `SOL-03`, `SOL-04`, `SD-04`, `C4-01`, `CTR-01`, `CTR-02`, `INV-01` | `design.md` | New diagnostic/session package and runner/app integration | `STEP-02` | `CHK-01`, `CHK-05` | `EVID-01`, `EVID-05` |
| `SOL-05`–`SOL-08`, `SD-03`, `CTR-03`, `INV-02`–`INV-06`, `FM-01`, `RB-01` | `design.md` | Session writer, redactor, cleanup, diagnostics and human path handoff | `STEP-03`, `STEP-05` | `CHK-03`–`CHK-05` | `EVID-03`–`EVID-05` |

## Workstreams

| Workstream | Implements | Result | Owner | Dependencies |
| --- | --- | --- | --- | --- |
| `WS-01` | `REQ-03`, `REQ-05` | Validated effective settings and opt-out dispatch | agent | `PRE-01`, `PRE-02` |
| `WS-02` | `REQ-01`, `REQ-02`, `REQ-06`, `REQ-08` | Private ordered session/invocation records, human path handoff and no event-stream contamination | agent | `WS-01` |
| `WS-03` | `REQ-04`, `REQ-07` | Redaction, permissions, bounded cleanup and non-fatal diagnostics | agent | `WS-02` |
| `WS-04` | All requirements | Public documentation and high-risk verification evidence | agent/human reviewer | `WS-01`–`WS-03` |

## Approval Gates

| Approval Gate ID | Trigger | Applies to | Why approval is required | Approver / evidence |
| --- | --- | --- | --- | --- |
| `AG-01` | Before implementation starts | `STEP-01`–`STEP-05` | `high-risk` profile requires explicit approval for risk-bearing execution involving durable sensitive diagnostic data. | Human approval recorded in issue, PR or this log. |
| `AG-02` | A platform cannot apply/document the selected privacy or atomicity semantics | Affected step | Continuing would weaken `CTR-02`, `INV-03` or `FM-01`. | Human decision recorded in `design.md` / decision log. |

## Order of Work

| Step ID | Actor | Implements | Goal | Touchpoints | Verifies | Evidence IDs | Blocked by | Needs approval | Escalate if |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `STEP-01` | agent | `REQ-03`, `REQ-05`, `SOL-01`, `SOL-02` | Add validated configuration, config display and opt-out parsing. | `internal/config`, `internal/app`, config/app tests | `CHK-02`, `CHK-04` | `EVID-02`, `EVID-04` | `PRE-01`, `PRE-02` | `AG-01` | Existing resolver cannot preserve precedence/source display. |
| `STEP-02` | agent | `REQ-01`, `REQ-02`, `REQ-06`, `REQ-08`, `SOL-03`, `SOL-04`, `SOL-08`, `CTR-01`, `CTR-02`, `INV-01`, `INV-06` | Build private per-session writer, human path handoff and runner integration. | New session package, runner/app/codex/event integration, deterministic fake runner tests | `CHK-01`, `CHK-05` | `EVID-01`, `EVID-05` | `STEP-01` | `AG-01` | Required metadata cannot be captured without changing result classification or stdout. |
| `STEP-03` | agent | `REQ-04`, `REQ-07`, `SOL-05`–`SOL-07`, `CTR-03`, `INV-03`–`INV-05`, `FM-01` | Add redaction, permissions, atomic records, bounded cleanup and warning-only errors. | Session package, filesystem/time/parallel tests | `CHK-03`, `CHK-05` | `EVID-03`, `EVID-05` | `STEP-02` | `AG-01`, `AG-02` if platform gap | Symlink/permission/atomicity behavior violates a selected invariant. |
| `STEP-04` | agent | `REQ-03`, `REQ-05`, `REQ-06`, `REQ-07` | Update root README and dependent Memory Bank operational/architecture facts. | `README.md`, relevant Memory Bank owners | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` | `STEP-01`–`STEP-03` | `AG-01` | Documentation reveals an unrepresented public contract. |
| `STEP-05` | agent + independent reviewer | All requirements | Run high-risk convergence and attach closure evidence. | Tests, docs lint, vet, diff check, independent code-converge/CI | `CHK-01`–`CHK-05` | `EVID-01`–`EVID-05` | `STEP-01`–`STEP-04` | `AG-01` | A required check fails or a manual-only gap lacks approval. |

## Checkpoints and Stop Conditions

- `CP-01`: `STEP-01` proves source precedence, validation and no-artifact opt-out before any writer is integrated.
- `CP-02`: `STEP-02` proves multiple Codex invocations become ordered private records while fake workflow stdout contains no raw stream.
- `CP-03`: `STEP-03` proves redaction, no-environment capture, private permissions, concurrent isolation and symlink-safe retention cleanup.
- Stop and update `design.md` before code proceeds if a required record cannot be atomically published inside an owned session directory, cleanup cannot avoid symlink traversal, or stream capture lacks required stage context.
- Stop for `AG-02` if a supported platform cannot provide/document owner-only modes where the issue requires them.

## Environment Contract

| Area | Contract | Used by | Failure symptom |
| --- | --- | --- | --- |
| Setup | Go toolchain and deterministic fake Codex/filesystem fixtures; no real Codex, remote or hosted CI mutation. | `STEP-01`–`STEP-05` | Tests call a live external agent or cannot inject filesystem/time. |
| Test | `go test ./...`, `go vet ./...`, `make docs-lint`, `git diff --check`; targeted race coverage when the relevant package is available. | `STEP-05` | An unavailable command is recorded as a gap, never a pass. |
| Access / secrets | No real authentication token or process environment value may be introduced into test fixtures or evidence. | `STEP-02`–`STEP-05` | A fixture/evidence carrier contains a credential or live secret. |

## Closure

- Complete `CHK-01`–`CHK-05`; attach the `EVID-*` carriers defined by `brief.md`.
- Obtain the high-risk approval and independent review/convergence required by the selected validation profile.
- Record any unavailable required check as a blocker or approved manual-only gap; do not mark delivery done otherwise.
