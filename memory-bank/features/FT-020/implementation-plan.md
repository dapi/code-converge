---
title: "FT-020: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Grounded execution plan for FT-020 without redefining the accepted root-help contract."
derived_from:
  - brief.md
  - design.md
  - ../../engineering/testing-policy.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_020_scope
  - ft_020_solution_space
  - ft_020_acceptance_criteria
  - ft_020_evidence_contract
---

# FT-020: Implementation Plan

## Discovery Context

| Context | Findings |
| --- | --- |
| Relevant paths | `internal/app/app.go` owns root command dispatch; `internal/app/app_test.go` has buffered streams and fake runner/updater patterns; root `README.md` owns public CLI text. |
| Local pattern | `App.Run` returns command-specific terminal results before setup for `--version` and `update`; tests inject dependencies and never call real Codex. |
| Unresolved questions | None: `DL-03` resolves the public payload. |
| Test surfaces | `internal/app`, root README, FT-020 package, full Go/docs/diff checks. |

## Preconditions

- `PRE-01` `brief.md` and `design.md` are active; `DL-03`, `SOL-01`–`SOL-03` and `CTR-01` are accepted.
- `PRE-02` The implementation stays inside the CLI boundary and does not add subcommand help or dynamic flag output.

## Workstreams

| Workstream | Scope | Canonical refs |
| --- | --- | --- |
| `WS-01` | Root early-return implementation and deterministic alias tests | `SOL-01`–`SOL-03`, `CTR-01`, `INV-01`, `FM-01` |
| `WS-02` | Public contract and feature/evidence convergence | `REQ-04`, `TRD-01`, `RB-01` |

## Steps

1. `STEP-01` Add the sole-argument alias branch and fixed usage writer to `internal/app/app.go` before any operational setup (`SOL-01`–`SOL-03`).
2. `STEP-02` Add table-driven app coverage for both aliases using buffered streams plus fake runner/updater; assert exact stdout, empty stderr, exit `0` and zero side effects (`CTR-01`, `INV-01`, `CHK-01`).
3. `STEP-03` Update the root README and FT-020 owners with the identical payload, selected design and evidence status (`REQ-04`, `CHK-02`).
4. `STEP-04` Run focused tests, full Go verification, docs lint and diff hygiene; inspect the final diff before publication (`CHK-01`–`CHK-03`).
5. `STEP-05` Commit, push, create/update a PR against `master`, inspect CI/review signals and repair any critical/high finding before closure (`EVID-03`).

## Checkpoints and Stop Conditions

- `CP-01` After `STEP-02`, both aliases have byte-identical stdout and neither fake dependency is called.
- `CP-02` Before publication, README/brief/design/plan use the same exact newline-terminated payload.
- `STOP-01` If satisfying a test requires expanding into subcommand help, configuration, workflow or a dynamic flag listing, stop and return to `brief.md`/`design.md`; that is outside current scope.
- `STOP-02` If CI exposes a critical/high issue, repair it and repeat affected local checks before closure.

## Test Strategy and Evidence Realization

`standard` validation is realized through deterministic app tests, public-contract review, all Go tests, vet, documentation lint and diff hygiene. No manual-only gap is planned.

| Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- |
| `CTR-01`, `INV-01`, `FM-01` | `STEP-01`, `STEP-02` | `CHK-01` | `EVID-01` |
| `SOL-01`–`SOL-03`, `TRD-01` | `STEP-01`–`STEP-03` | `CHK-02` | `EVID-02` |
| `REQ-01`–`REQ-04`, `RB-01` | `STEP-04`, `STEP-05` | `CHK-03` | `EVID-03` |
