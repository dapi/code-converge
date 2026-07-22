---
title: "FT-012: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Execution plan for the verified self-update command, grounded in FT-012 canonical problem and solution owners."
derived_from:
  - brief.md
  - design.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_012_scope
  - selected_solution
  - ft_012_acceptance_criteria
---

# FT-012: Implementation Plan

## Canonical Ownership

| Owner | Owns | Change rule |
| --- | --- | --- |
| `brief.md` | `REQ-*`, `SC-*`, `CHK-*`, `EVID-*`, validation profile | Update first if requirements/evidence change. |
| `design.md` | `SOL-*`, `CTR-*`, `INV-*`, `FM-*`, `RB-*` | Update first if solution/failure semantics change. |
| `../../../README.md` | Public CLI and exit/output contract | Update atomically with implementation. |

## Preconditions

| ID | Requirement | Evidence |
| --- | --- | --- |
| `PRE-01` | `D-01` accepted; current/declined exit `0`, operational error `2`, status stdout/diagnostics stderr | decision log and `CTR-03` |
| `PRE-02` | High-risk bounded execution approved by user instruction; no production release mutation | `D-01` execution approval |

## Steps

| Step | Implements | Touchpoints | Verifies |
| --- | --- | --- | --- |
| `STEP-01` | `REQ-01`–`REQ-05`, `SOL-01`–`SOL-04`, `CTR-01`, `CTR-03` | `internal/app`, new `internal/update`, deterministic fakes | `CHK-01`, `CHK-02` |
| `STEP-02` | `REQ-02`, `REQ-03`, `REQ-06`, `REQ-07`, `SOL-05`, `CTR-02`, `INV-01`–`INV-03`, `FM-01` | `internal/update`, archive/checksum/filesystem tests | `CHK-02`, `CHK-03` |
| `STEP-03` | `REQ-08` | root README and applicable Memory Bank status/evidence | `CHK-04` |
| `STEP-04` | All requirements, high-risk convergence | full suites, release smoke, docs lint, diff check, PR CI/review | `CHK-05` |

## Checkpoints and Stop Conditions

- `CP-01`: parser and output/exit contract tests prove no workflow behavior is invoked for `update`.
- `CP-02`: all no-change/error cases prove old executable bytes survive.
- `CP-03`: a verified archive replacement reports target version; smoke coverage agrees with the release matrix.
- Stop and update `design.md` before code if the OS cannot atomically rename a staged same-directory replacement or if a required asset/checksum convention differs from `scripts/install.sh`.
- Stop for a human decision if testing reveals a new public output/exit distinction or a non-atomic replacement behavior not covered by `RB-01`.

## Approval and Closure

- `AG-01`: User authorized the feature-local FPF decision and end-to-end bounded implementation in this conversation; no production/live-release action is authorized.
- Closure requires `CHK-01`–`CHK-05`, independent `code-converge` convergence, no open critical/high review finding, a green required CI set and a mergeable PR.
