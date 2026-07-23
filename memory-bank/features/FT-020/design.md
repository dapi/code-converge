---
title: "FT-020: Design"
doc_kind: feature
doc_function: canonical
purpose: "Solution-space owner for the stable root-help stdout contract and early return."
derived_from:
  - brief.md
  - decision-log.md
  - ../../../README.md
  - ../../engineering/architecture.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_020_scope
  - ft_020_acceptance_criteria
  - ft_020_evidence_contract
  - implementation_sequence
---

# FT-020: Design

## Design Pack

| Artifact | Role | Owns |
| --- | --- | --- |
| `design.md` | Feature-local solution owner | `SOL-*`, `ALT-*`, `TRD-*`, `C4-*`, `CTR-*`, `INV-*`, `FM-*`, `RB-*` |
| `decision-log.md` | Reasoning provenance | FPF closure and review cycles; no canonical solution facts |
| `../../../README.md` | Public CLI contract | Exact published root-help text |

## Context

`REQ-01`–`REQ-03` add a public root-command branch inside the existing CLI boundary. `DL-03` selects a compact fixed payload so agents can recognize help without parsing a changeable flag list. Workflow, configuration, update and progress components must remain untouched.

## C4 Applicability

| C4 ID | Decision | Trigger / reason | Artifact |
| --- | --- | --- |
| `C4-00` | not required | One existing CLI component gets a synchronous local early return; no component responsibility, connector, container or external boundary changes. | none |

## Architecture Coverage Decision

| Aspect | Status | Canonical owner / refs | Supporting view / artifact | Reason if N/A / coverage note |
| --- | --- | --- | --- | --- |
| Components / responsibilities | covered | `SOL-01`, `SD-01` | none | `internal/app` remains the CLI boundary and owns the alias decision. |
| Connectors / interactions | N/A | none | none | No process, API, event, storage or configuration connector is entered. |
| Configuration / topology | N/A | none | none | Help returns before configuration resolution; no topology changes. |
| Behavioral semantics | covered | `CTR-01`, `INV-01`, `FM-01` | none | Exact stdout, stderr, exit and early-return semantics are specified below. |
| Quality / evolution concerns | covered | `TRD-01`, `RB-01` | none | A fixed minimal line avoids volatile agent-facing output. |

## Selected Solution

- `SOL-01` In `App.Run`, recognize exactly one root argument equal to `-h` or `--help` before version, update, cwd, flag parsing or configuration handling.
- `SOL-02` Write the fixed `usage: code-converge [flags] [config]\n` payload to the configured stdout and return `workflow.ExitSuccess`.
- `SOL-03` Keep every other argument path unchanged, including `update` parsing and invalid positional-argument diagnostics.

## Alternatives and Trade-offs

| Alternative ID | Option | Why not selected |
| --- | --- | --- |
| `ALT-01` | Delegate root help to Go `flag` defaults | Its dynamically rendered flag list would become an undocumented, changeable agent interface. |
| `ALT-02` | Maintain a full manually written flag/command list | It adds content and drift risk outside issue #20. |
| `ALT-03` | Reuse the error-path line with `code-converge:` prefix | The prefix marks a diagnostic, whereas successful help must be stdout-only with no error. |

| Trade-off ID | Decision | Benefit | Cost / Risk |
| --- | --- | --- | --- |
| `TRD-01` | Publish only a fixed usage line | Stable for humans and agents; no drift with flags | It intentionally does not enumerate options. |

## Accepted Local Decisions and Contracts

- `SD-01` Root help is recognized before all operational setup; no new configuration or workflow seam is introduced.
- `CTR-01` For exactly `-h` or `--help` as the sole root argument, stdout is exactly `usage: code-converge [flags] [config]\n`, stderr is empty, and the exit code is `0`.
- `INV-01` A successful root-help invocation cannot invoke update, process runner, configuration loading, review setup or workflow/progress output.

## Failure Modes

| ID | Failure | Handling |
| --- | --- | --- |
| `FM-01` | Alias is processed after operational setup or emits dynamic/error output | Deterministic buffered-output and fake-dependency tests fail; keep recognition at the top of `App.Run`. |

## Rollout / Backout

| Stage ID | Stage | Entry condition | Backout |
| --- | --- | --- |
| `RB-01` | Local binary release | Focused and full checks pass | Revert the isolated CLI branch; no persisted state, migration or external action exists. |

## Design Verification

| Analysis | Required | Reason / risk | Method | Result / evidence |
| --- | --- | --- | --- | --- |
| Contract compatibility | yes | Public stdout/exit contract affects agents | Exact buffered-output tests for both aliases | `CHK-01` |
| State / transition completeness | yes | Help must bypass every operational state | Fake runner/updater assertions | `CHK-01` |
| Failure propagation | no | No new external or fallible connector | N/A | N/A |
| Concurrency / ordering | no | Synchronous single return only | N/A | N/A |
| Security boundaries | no | No auth/trust/data change | N/A | N/A |
| Capacity / latency | no | No process/configuration work occurs | N/A | N/A |
| Migration / evolution safety | yes | Public docs and tests must converge | README + feature contract review | `CHK-02`, `CHK-03` |

## Traceability

| Requirement ID | Solution refs | Contracts / invariants | Failure / rollout refs |
| --- | --- | --- | --- |
| `REQ-01` | `SOL-01`, `SOL-02`, `SD-01` | `CTR-01`, `INV-01` | `FM-01`, `RB-01` |
| `REQ-02` | `SOL-02` | `CTR-01` | `FM-01`, `RB-01` |
| `REQ-03` | `SOL-01`, `SOL-03` | `INV-01` | `FM-01`, `RB-01` |
| `REQ-04` | `SOL-01`–`SOL-03` | `CTR-01`, `INV-01` | `RB-01` |
