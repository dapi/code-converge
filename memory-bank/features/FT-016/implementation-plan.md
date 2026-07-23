---
title: "FT-016: Archived Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Archived execution plan for FT-016 base discovery and one complete branch-and-worktree review snapshot."
derived_from:
  - brief.md
  - design.md
  - ../../engineering/testing-policy.md
status: archived
audience: humans_and_agents
must_not_define:
  - ft_016_scope
  - ft_016_selected_design
  - ft_016_acceptance_criteria
  - ft_016_blocker_state
  - ft_016_validation_profile
---

# FT-016: Implementation Plan

> Archived after the implementation was merged in PR [#19](https://github.com/dapi/code-converge/pull/19) and the required `Verify` run passed. Canonical outcome and evidence are in [`brief.md`](brief.md).

## Discovery Context

| Context | Findings |
| --- | --- |
| Relevant paths | `internal/config` owns settings resolution; `internal/app` wires config, Codex and repository collaborators; `internal/codex` owns the review command; `internal/repository` currently owns Git status only; `internal/runner` captures process invocations; `internal/workflow` owns transitions. |
| Local patterns | Config uses a single `spec` resolver and source metadata; adapters use injected runners; tests use deterministic fake runners/executables; current review is `codex review --uncommitted`. |
| Open questions | none after `DL-03`; implementation must stop if installed Codex cannot honor a temporary-index environment for `--base`. |
| Environment | Tests must fake Git/`gh`/Codex and must not invoke a real Codex session, mutate a real remote or wait for hosted CI. |

## Workstreams

| Workstream | Implements | Result |
| --- | --- | --- |
| `WS-01` | `SOL-01`, `CTR-01`, `SD-01`–`SD-02` | Config/flag/source reporting for review base. |
| `WS-02` | `SOL-02`, `CTR-02`, `FM-01`–`FM-04` | Fakeable local Git and optional provider discovery. |
| `WS-03` | `SOL-03`–`SOL-05`, `CTR-03`–`CTR-05`, `INV-01`–`INV-04` | Temporary-index snapshot, Codex environment propagation and metadata events. |
| `WS-04` | `REQ-08`, `RB-01` | Public/domain/architecture documentation and complete evidence. |

## Steps

1. `STEP-01` Add `review-base` through config/app flags and config-format tests (`REQ-02`, `CTR-01`, `CHK-03`).
2. `STEP-02` Extend runner invocations with a scoped environment and introduce a fakeable review-discovery/snapshot collaborator under `internal/repository`; write the candidate/ambiguity/error matrix before production wiring (`SOL-02`, `CTR-02`, `FM-01`–`FM-04`, `CHK-01`).
3. `STEP-03` Wire discovery once per workflow run and snapshot refresh before every Codex review; update the Codex adapter to invoke `review --base` with a wrapper-prefixed `PATH` forced through its per-invocation Codex shell-policy override and disable login-shell startup for that review. Keep `GIT_INDEX_FILE` and helper configuration local to snapshot/helper execution via a private wrapper sidecar, and set `GIT_EXEC_PATH` only within the wrapper's Git child process (`SOL-03`–`SOL-04`, `INV-01`–`INV-04`, `CHK-01`, `CHK-02`).
4. `STEP-04` Add structured-safe review metadata fields and app/workflow golden tests for all source values and operational failures (`SOL-05`, `CTR-05`, `CHK-03`).
5. `STEP-05` Update root README first, then domain rules/states and architecture to describe only delivered behavior; update FT-016 evidence state (`REQ-08`, `RB-01`, `CHK-04`).
6. `STEP-06` Run focused tests, full Go checks, documentation lint and diff hygiene; record evidence and required CI in the feature package (`CHK-01`–`CHK-05`).

## Checkpoints and Stop Conditions

- `CP-01` After `STEP-02`, each selection source has positive, unavailable, ambiguity and stale/missing cases; no test needs live GitHub or remote access.
- `CP-02` After `STEP-03`, assert only snapshot/helper Git mutations carry the temporary `GIT_INDEX_FILE`; every Codex review forces a wrapper-prefixed `PATH` through `shell_environment_policy.set` and disables login-shell startup, so profile initialization cannot bypass the helper; the wrapper sidecar keeps helper configuration out of restrictive `include_only` environments; every reviewed-root wrapper command uses a disposable index copy so absolute or nested descendant Git cannot corrupt the stable snapshot; and repository creation uses only the target repository index.
- `CP-03` Before publication, map every `REQ-*` to a passing `CHK-*` and an `EVID-*` carrier.
- `STOP-01` If `codex review --base` does not inherit the runner's environment or does not honor the temporary index, stop before implementation alters the public contract; return to `design.md` with observed evidence.
- `STOP-02` If a provider query cannot distinguish unavailable service/authentication from a discovered ambiguous PR set without parsing raw output safely, stop and refine the provider adapter boundary.

## Test Strategy

`high-risk` validation uses table-driven config/repository/workflow tests, deterministic shell-policy and provider-identity failure coverage, fake-executable Codex/Git integration and event golden tests, then `go test ./...`, `go vet ./...`, `make docs-lint` and `git diff --check`. The user's 2026-07-22 remediation directive is the profile approval; no manual-only gap is planned. Required CI and independent review remain delivery gates.

## Evidence Realization

| Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- |
| `CTR-01`, `CTR-02`, `FM-01`–`FM-04` | `STEP-01`, `STEP-02` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `CTR-03`–`CTR-04`, `INV-01`–`INV-04` | `STEP-03` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `CTR-05`, `SOL-05` | `STEP-04` | `CHK-03` | `EVID-03` |
| `REQ-08`, `RB-01` | `STEP-05`, `STEP-06` | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` |
