---
title: "FT-010: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Derived execution sequence for interactive terminal presentation and its evidence."
derived_from:
  - brief.md
  - design.md
  - ../../adr/ADR-001-interactive-terminal-runtime.md
  - ../../engineering/testing-policy.md
status: archived
audience: humans_and_agents
must_not_define:
  - feature_scope
  - selected_solution
  - canonical_acceptance_criteria
  - canonical_evidence_contract
---

# FT-010: Implementation Plan

## Discovery context

- `internal/runner` currently returns fully captured process stdout/stderr; it needs an observer-capable streaming seam without changing non-interactive capture semantics.
- `internal/codex` owns active Codex process invocation; `internal/workflow` owns stage identity and cancellation; `internal/event` owns stdout/liveness coordination; `internal/app` wires terminal capabilities.
- `process_unix.go` and `process_windows.go` demonstrate existing platform split. ADR-001 selects `golang.org/x/term` instead of duplicating terminal-mode mechanics.
- Existing fakes in app, adapter, runner, workflow, and event tests are the reference pattern for deterministic tests. No real Codex session or terminal is allowed in automation.

## Preconditions

- `PRE-01` `brief.md` and `design.md` are active and `DEC-01` is resolved by `DL-03`.
- `PRE-02` ADR-001 is accepted.
- `PRE-03` Working-tree changes are inventoried before source edits.

## Design realization mapping

| Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- |
| `SOL-01`, `SOL-02`, `SD-01`, `SD-02`, `CTR-01`, `INV-01`, `INV-02` | `STEP-01`, `STEP-04` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `SOL-03`, `SD-03`, `SD-04`, `CTR-03`, `INV-05`, `FM-02` | `STEP-02` | `CHK-01` | `EVID-01` |
| `SOL-04`, `SOL-05`, `SD-05`, `SD-06`, `CTR-02`, `INV-03`, `FM-03`, `FM-04` | `STEP-03` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `SOL-06`, `SD-07`, `INV-04`, `FM-01`, `FM-05`, `ADR-001` | `STEP-01`, `STEP-04` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `RB-01` | `STEP-05`, `STEP-06` | `CHK-03`–`CHK-05` | `EVID-03`–`EVID-05` |

## Work sequence

| Step | Implements | Touchpoints | Verifies |
| --- | --- | --- | --- |
| `STEP-01` | terminal eligibility, raw mode, restore | `go.mod`, app wiring, repository terminal runtime | `CHK-01` |
| `STEP-02` | bounded panes, layout, key/resize event loop | `internal/event` or dedicated presentation package | `CHK-01` |
| `STEP-03` | streamed runner observer, sanitizer, active stream identity | `internal/runner`, `internal/codex`, `internal/workflow`, presentation | `CHK-01`, `CHK-02` |
| `STEP-04` | lifecycle integration and non-interactive invariants | app/workflow/event integration and deterministic fakes | `CHK-01`, `CHK-02` |
| `STEP-05` | public/derived documentation | root README first, then architecture and FT-010 evidence status | `CHK-04` |
| `STEP-06` | convergence and delivery evidence | full suite, race tests, vet, docs lint, diff check, CI, terminal smoke | `CHK-03`–`CHK-05` |

## Checkpoints

| Checkpoint | Condition | Evidence |
| --- | --- | --- |
| `CP-01` | TTY eligibility and every cleanup path are deterministic. | `EVID-01` |
| `CP-02` | Pane/stream state never mixes processes and preserves non-interactive output. | `EVID-01`, `EVID-02` |
| `CP-03` | Manual terminal smoke confirms terminal restoration. | `EVID-03` |
| `CP-04` | Root README, architecture, ADR, feature docs, and behavior converge. | `EVID-04` |
| `CP-05` | Full local and CI validation are green. | `EVID-05` |

## Approval gates

- `AG-01` User instruction of 2026-07-23 authorizes the FPF decision `DL-03` and creation of the downstream feature documents.
- `AG-02` Any manual-only critical-path gap needs a named human approver, procedure, and reference before closure.

## Stop conditions / fallback

- Stop and update `design.md` if implementation needs a public flag, changes non-interactive stdout, preserves raw ANSI, or cannot meet `CTR-01` with ADR-001.
- Stop before source changes if `x/term` cannot support a required supported platform without a new portability decision.
- On terminal setup failure, restore if needed and fall back to current permanent output; do not fail or change workflow semantics solely because the view is unavailable.

## Ready for acceptance

All `CHK-*` have concrete passing `EVID-*`; deterministic and race tests are green; the manual smoke records open, scroll, resize, stage transition, completion, interrupt, and shell restoration; public and derived docs agree; and no critical/important convergence finding remains.
