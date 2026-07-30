---
title: "FT-036: Discoverable CLI Help Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Execution sequencing and validation mapping for FT-036."
derived_from:
  - brief.md
  - design.md
  - ../../engineering/testing-policy.md
status: active
audience: humans_and_agents
---

# FT-036: Discoverable CLI Help Implementation Plan

## Discovery Context

| Area | Result |
| --- | --- |
| Relevant paths | `internal/app/app.go` dispatches root help, update, config, flag parsing, config load, sessions and workflow; `internal/app/app_test.go` already tests root-help side effects; `README.md` owns public CLI contract. |
| Local patterns | `App` accepts fake `Runner` and `Updater`, so deterministic tests can prove help bypasses both. Global flags are bound in one contiguous `flag.FlagSet` block. |
| Unresolved questions | none — issue acceptance and current implementation establish all required help forms and no conflicting command pattern was found. |
| Test surfaces | Focused `internal/app` output/side-effect tests, full Go suite/vet, documentation lint, diff check, and hosted `Verify` CI. |
| Environment | Go 1.21 project; commands are `go test ./internal/app`, `go test ./...`, `go vet ./...`, `make docs-lint`, and `git diff --check`. |

## Test Strategy

Validation profile is `standard` in [`brief.md`](brief.md#validation-profile-decision).

| Test surface | Canonical refs | Planned automated coverage | Required local suites / commands | Required CI suites / jobs | Manual-only gap |
| --- | --- | --- | --- | --- | --- |
| Help dispatch/rendering | `REQ-01`–`REQ-03`, `SOL-01`–`SOL-03`, `CTR-01`, `CTR-02`, `INV-01`, `INV-02` | Table-driven root/config/update help tests with exact required fragments, exit codes, and fake collaborator assertions | `go test ./internal/app` | `Verify` / `make verify` | none |
| Existing behavior | `REQ-04`, `SD-03`, `NEG-01` | Existing invalid/update and workflow tests remain green | `go test ./...`; `go vet ./...` | `Verify` | none |
| Public documentation | `REQ-04`, `CHK-04` | Root help reference updated in README | `make docs-lint` | `Verify` | none |

## Preconditions

| Precondition ID | Canonical ref | Required state | Used by steps | Blocks start |
| --- | --- | --- | --- | --- |
| `PRE-01` | `SD-01`–`SD-03` | Solution Ready design is active. | `STEP-01`–`STEP-03` | yes |

## Design Realization Mapping

| Canonical solution refs | Owner | Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- | --- | --- |
| `SOL-01`, `SD-01`, `CTR-01`, `CTR-02`, `INV-01`, `INV-02` | `design.md` | `internal/app/app.go` early dispatch and help renderers | `STEP-01` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `SOL-02`, `SD-02`, `FM-01` | `design.md` | shared global option declaration and root renderer | `STEP-01`, `STEP-02` | `CHK-01` | `EVID-01` |
| `SOL-03`, `SD-03`, `RB-01` | `design.md` | app regression tests and README | `STEP-02`, `STEP-03` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` |
| `C4-00` | `design.md` | no runtime/topology realization required | `STEP-01` | `CHK-03` | `EVID-03` |

## Workstreams

| Workstream | Implements | Result | Owner | Dependencies |
| --- | --- | --- | --- | --- |
| `WS-1` | `REQ-01`–`REQ-03` | Side-effect-free discoverable help implementation and focused tests | agent | `PRE-01` |
| `WS-2` | `REQ-04` | Public contract documentation and complete validation | agent | `WS-1` |

## Approval Gates

No approval gate is required: no production, live-data, security, or irreversible operation is in scope.

## Порядок работ

| Step ID | Actor | Implements | Goal | Touchpoints | Verifies | Evidence IDs | Check command / procedure | Blocked by | Needs approval | Escalate if |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `STEP-01` | agent | `REQ-01`–`REQ-03`, `SOL-01`, `SOL-02` | Add early help dispatch and deterministic renderers sharing global-option metadata with flags. | `internal/app/app.go` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` | `go test ./internal/app` | `PRE-01` | none | A discovered existing pattern conflicts materially with `SD-02`. |
| `STEP-02` | agent | `REQ-01`–`REQ-04`, `SOL-03` | Add focused contract/side-effect/regression coverage. | `internal/app/app_test.go` | `CHK-01`–`CHK-03` | `EVID-01`–`EVID-03` | `go test ./internal/app`; `go test ./...`; `go vet ./...` | `STEP-01` | none | Behavior outside help needs a contract decision. |
| `STEP-03` | agent | `REQ-04` | Update the README root-help contract, then validate docs and diff. | `README.md` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` | `make docs-lint`; `git diff --check` | `STEP-02` | none | Documentation conflicts with accepted CLI contract. |

## Checkpoints

| Checkpoint ID | Refs | Condition | Evidence IDs |
| --- | --- | --- | --- |
| `CP-01` | `STEP-01`, `CHK-01`, `CHK-02` | Focused tests prove all help forms and no side effects. | `EVID-01`, `EVID-02` |
| `CP-02` | `STEP-03`, `CHK-03`, `CHK-04` | Full local validation and docs lint pass before final review. | `EVID-03`, `EVID-04` |

## Execution Risks

| Risk ID | Risk | Impact | Mitigation | Trigger |
| --- | --- | --- | --- | --- |
| `ER-01` | Help text drifts from global flags. | Public contract is misleading. | Use one declaration and focused assertions. | A flag changes without help-test coverage. |

## Stop Conditions / Fallback

| Stop ID | Related refs | Trigger | Immediate action | Safe fallback state |
| --- | --- | --- | --- | --- |
| `STOP-01` | `SD-02`, `ER-01` | Shared declaration would change flag parsing semantics. | Return to Solution Ready and choose a compatible presentation strategy. | No implementation change beyond current clean checkpoint. |
