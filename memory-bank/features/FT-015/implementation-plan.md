---
title: "FT-015: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Derived execution plan for FT-015; realizes the accepted strict structured-review and no-change solution without redefining requirements or design."
derived_from:
  - brief.md
  - design.md
  - ../../engineering/testing-policy.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_015_scope
  - ft_015_solution_space
  - ft_015_acceptance_criteria
  - ft_015_evidence_contract
---

# FT-015: Implementation Plan

## Discovery Context

| Context | Findings |
| --- | --- |
| Relevant paths | `internal/codex/adapter.go` owns `ParseReview`; `internal/workflow/workflow.go` owns the review-to-finalize transition; `internal/runner` owns local process execution; `README.md`, `memory-bank/domain/rules.md` and `memory-bank/domain/states.md` own delivered public/domain wording. |
| Local reference patterns | `ParseFinalization` already rejects duplicate JSON keys, unknown fields and trailing JSON; adapter/workflow tests use deterministic fake runners and agents. |
| Unresolved questions | `OQ-01: none after discovery.` The issue sample, installed CLI 0.145.0 and recorded local structured finding establish `CTR-01`/`CTR-02`; versions emitting another shape remain fail-closed by `SD-01`. |
| Test surfaces | `internal/codex`, `internal/workflow`, `internal/app`, fake executable/runner integration, docs lint and full Go checks. |
| Execution environment | Local Go repository; tests must not start a real Codex session, mutate a remote or create a change request. |

## Preconditions

- `PRE-01` `brief.md` and `design.md` are active, all `SD-*`/`CTR-*` facts are accepted and validation profile `standard` remains applicable.
- `PRE-02` A fakeable repository-status seam can be introduced without adding an unrelated configuration or public CLI contract.
- `PRE-03` Implementer has a fixture for the issue's structured clean response and for the observed complete prioritized finding response.

## Workstreams

| Workstream | Scope | Canonical refs |
| --- | --- | --- |
| `WS-01` | Strict review parser and fixtures | `SOL-01`–`SOL-03`, `CTR-01`–`CTR-03`, `INV-01`–`INV-02`, `FM-01`–`FM-02` |
| `WS-02` | Clean-review repository-status transition | `SOL-04`–`SOL-05`, `CTR-04`, `INV-03`, `FM-03`–`FM-04` |
| `WS-03` | Public/domain documentation and end-to-end evidence | `REQ-05`, `RB-01`, `CHK-03`–`CHK-05` |

## Steps

1. `STEP-01` Add table-driven parser fixtures for valid structured clean and prioritized findings, then the complete rejected matrix from `NEG-01`; reuse the strict finalization JSON scanning pattern only where it realizes `CTR-01`–`CTR-03`.
2. `STEP-02` Preserve legacy plain-text fixtures and add regression assertions for counter mapping and normalized structured report hand-off to fix-findings (`REQ-02`, `CHK-01`).
3. `STEP-03` Add a fakeable repository-status collaborator and workflow branch that realizes `CTR-04`; cover changes, no changes and query failure without making a real Git or Codex call (`REQ-04`, `CHK-02`).
4. `STEP-04` Add fake-executable/app integration coverage proving the review command remains unchanged, structured clean emits existing counters, no-change skips finalize and changed worktrees retain finalization (`CHK-03`).
5. `STEP-05` Update `README.md` first, then `memory-bank/domain/rules.md`, `memory-bank/domain/states.md` and any directly dependent current-state docs so the public contract reflects `SD-01` and `SD-03`; do not add to-be claims before implementation is atomic (`REQ-05`, `CHK-04`).
6. `STEP-06` Run the selected local suites, docs lint, full Go verification and diff hygiene; record concrete evidence carriers and required CI after publication (`CHK-01`–`CHK-05`).

## Checkpoints and Stop Conditions

- `CP-01` After `STEP-02`, review every accepted/rejected fixture against `CTR-01`–`CTR-03`; stop if a supported observed field cannot be represented without relaxing validation.
- `CP-02` After `STEP-04`, confirm no-change produces neither finalize-stage records nor a finalization invocation; stop if a test relies on a real repository or external process side effect.
- `CP-03` Before delivery closure, confirm each `REQ-*` maps to a passing `CHK-*` and concrete `EVID-*` carrier.
- `STOP-01` If another Codex version must be supported but emits a schema other than `CTR-01`/`CTR-02`, stop and return to `design.md`/decision log for evidence-backed contract expansion.
- `STOP-02` If no safe status seam fits the existing runner/workflow boundaries, stop before implementation and obtain a design decision; do not hide Git semantics in an unrelated component.

## Test Strategy

`standard` validation is realized by focused parser, workflow/app and fake-executable integration coverage, followed by all affected/full local suites and documentation convergence. No manual-only gap is planned. Functional validation, simplify review and acceptance evidence are separate passes: after tests, simplify touched code; then verify event sequences and document contract against `SC-01`–`SC-05`.

## Evidence Realization

| Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- |
| `CTR-01`–`CTR-03`, `INV-01`–`INV-02`, `FM-01`–`FM-02` | `STEP-01`, `STEP-02` | `CHK-01` | `EVID-01` |
| `CTR-04`, `INV-03`, `FM-03`–`FM-04` | `STEP-03` | `CHK-02` | `EVID-02` |
| `SOL-01`–`SOL-05`, `SD-01`–`SD-03` | `STEP-01`–`STEP-04` | `CHK-01`–`CHK-03` | `EVID-01`–`EVID-03` |
| `REQ-05`, `RB-01` | `STEP-05`, `STEP-06` | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` |
