---
title: "FT-028: Fix stale interactive liveness lines"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem-space and verification owner for issue #28: stale transient human liveness frames must not remain in interactive terminal history."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../../engineering/testing-policy.md
  - ../../features/FT-009/design.md
  - ../../../README.md
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - implementation_sequence
---

# FT-028: Fix stale interactive liveness lines

## What

### Problem

Issue #28 reports that interactive human stdout may leave one or more `Reviewing...` liveness frames as permanent-looking terminal-history lines before a later permanent completion record. This contradicts the accepted liveness contract in the root [README](../../../README.md) and FT-009 `CTR-04`/`INV-04`: one stage has one transient in-place line, cleared before permanent stdout or stderr diagnostic output.

The issue states that workflow execution is not duplicated; the defect is limited to presentation. The current renderer writes an unbounded frame preceded by carriage return plus `CSI 2K`; it retains only a boolean transient state and no rendered-width or terminal-layout state. That source fact is consistent with the issue's resize, reflow, and scrolling reproduction context, but does not by itself establish a portable terminal-control remedy.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Stale interactive liveness frames after a stage boundary | Issue #28 reports one or more retained frames | None for every covered stage completion and diagnostic boundary | Deterministic terminal-model regression test, plus explicitly recorded manual terminal evidence if the selected terminal behavior cannot be automated |
| `MET-02` | Writes after `Liveness.Stop()` returns | Existing contract requires none | Zero | Deterministic concurrency test with controlled ticks and writer |

### Scope

- `REQ-01` In interactive human mode, clear the active liveness presentation before every subsequent permanent stdout record and stderr diagnostic so no stale `Reviewing...` (or equivalent stage) frame remains in terminal history.
- `REQ-02` Preserve the existing human and `kv` public output contracts and retain interactive liveness; the defect fix must not alter workflow transitions or duplicate stages.
- `REQ-03` Preserve the stop/join ordering guarantee: no liveness goroutine writes after `Stop()` completes, including completion, failure, cancellation, and liveness-writer-failure paths.
- `REQ-04` Confirm the failure mechanism before selecting a terminal-rendering change, and add a regression test for that confirmed mechanism when technically possible; otherwise record the terminal limitation and manual evidence.

### Non-Scope

- `NS-01` Changing the CLI, configuration names/defaults, `kv` schema, heartbeat semantics, stage wording, or workflow result/exit-code behavior.
- `NS-02` Removing interactive liveness or replacing it with newline heartbeats as a workaround.
- `NS-03` Claiming compatibility with terminal control semantics, resize/reflow behavior, or a terminal matrix that is not supported by recorded evidence.

### Constraints / assumptions

- `ASM-01` The root README remains the public stdout-contract owner; this feature may update it only if accepted behavior needs a contract clarification, not to redefine the contract by documentation alone.
- `ASM-02` FT-009 remains the canonical owner of the existing liveness contract. FT-028 is a corrective delivery unit and must not contradict `CTR-04` or `INV-04`.
- `CON-01` The issue requires a confirmed failure mechanism before choosing the fix and explicitly names terminal resize, reflow, and scrolling as relevant conditions.
- `CON-02` Concurrency semantics and a compatibility-sensitive stdout contract activate the `high-risk` validation profile.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The fix changes concurrent terminal-rendering mechanics, requires an explicit footprint/clear trade-off, and must preserve a public stdout contract. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | Feature-local FPF and the material terminal-semantics uncertainty need an auditable owner. | `decision-log.md` |
| `runtime-surfaces.md` | omitted | Current affected runtime surface is compact and grounded directly in issue #28, FT-009, `internal/event`, and `internal/workflow`; a separate inventory would duplicate those facts. | none |
| `design.md` | selected | The footprint-aware rendering and clearing contract needs a solution-space owner. | `design.md` |
| `implementation-plan.md` | selected | Active brief and design now provide the required upstream facts for executable sequencing. | `implementation-plan.md` |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `high-risk` | The change targets concurrent liveness/stop ordering and a compatibility-sensitive public stdout contract. Validation must cover affected concurrency, contract, regression, and failure paths. | none |

## Verify

### Exit criteria

- `EC-01` Covered interactive liveness frames are cleared before every permanent stdout record and stderr diagnostic; no stale frame remains after a completion, failure, or cancellation boundary.
- `EC-02` `Liveness.Stop()` joins the worker and prevents later liveness writes.
- `EC-03` Human/`kv` public contracts and workflow behavior remain unchanged except for the corrective rendering behavior required by `REQ-01`.
- `EC-04` The confirmed failure mechanism has deterministic regression coverage, or the evidence records why automation cannot model the terminal behavior and supplies the approved manual procedure/result.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `CON-01`, `CON-02` | `EC-01`, `SC-01`, `SC-02` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `REQ-02` | `ASM-01`, `ASM-02`, `CON-02` | `EC-03`, `SC-03` | `CHK-03` | `EVID-03` |
| `REQ-03` | `CON-02` | `EC-02`, `SC-04` | `CHK-04` | `EVID-04` |
| `REQ-04` | `CON-01` | `EC-04`, `SC-01` | `CHK-01`, `CHK-05` | `EVID-01`, `EVID-05` |

### Acceptance scenarios

- `SC-01` A deterministic terminal model reproduces the confirmed wrapped/reflowed-frame mechanism; ending the stage clears every owned transient frame before the completion line.
- `SC-02` An active interactive stage followed by an error diagnostic clears its transient frame before the diagnostic is emitted.
- `SC-03` Human non-interactive/heartbeat and `kv` paths retain their existing newline/ANSI/schema behavior; the workflow produces the same permanent events and terminal result.
- `SC-04` Completion racing a liveness tick, context cancellation, and liveness-writer failure leave no late transient write after `Stop()`.

### Negative and edge cases

- `NEG-01` A terminal resize or reflow must not cause a previously drawn owned frame to survive the next clear boundary.
- `NEG-02` The fix must not emit cursor-control or ANSI sequences into non-interactive human output or `kv` output.
- `NEG-03` A writer failure during rendering remains an operational failure and does not introduce a second liveness write.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `SC-01`, `NEG-01`, `REQ-04` | Targeted deterministic `internal/event` regression test for the selected footprint-aware mechanism. | Reproduces the confirmed mechanism and proves no retained owned frame after clear. | `artifacts/ft-028/verify/chk-01/` |
| `CHK-02` | `SC-02`, `EC-01` | Targeted `internal/event` test for transient-to-diagnostic ordering. | Clear occurs before diagnostic and no stale frame remains. | `artifacts/ft-028/verify/chk-02/` |
| `CHK-03` | `SC-03`, `NEG-02`, `EC-03` | Affected event/app/workflow contract tests. | Existing permanent human, heartbeat, `kv`, and workflow semantics pass unchanged. | `artifacts/ft-028/verify/chk-03/` |
| `CHK-04` | `SC-04`, `NEG-03`, `EC-02` | Deterministic tick, cancellation, and failing-writer tests; race run for affected package. | Stop/join barrier and first-error behavior hold with no late writes or race report. | `artifacts/ft-028/verify/chk-04/` |
| `CHK-05` | `EC-04` | If `CHK-01` cannot represent the selected terminal behavior, execute the human-approved terminal procedure. | Recorded terminal, dimensions/actions, observed output, and result establish the stated limitation or acceptance. | `artifacts/ft-028/verify/chk-05/` |

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | Deterministic reproduction/regression test log | targeted Go test | `artifacts/ft-028/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Clear-before-diagnostic test log | targeted Go test | `artifacts/ft-028/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Contract-regression suite log | affected Go tests | `artifacts/ft-028/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Concurrency/race test log | affected Go tests with race detector | `artifacts/ft-028/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Manual terminal evidence or explicit automation-limitation record | human-approved procedure | `artifacts/ft-028/verify/chk-05/` | `CHK-05` |
