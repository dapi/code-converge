---
title: "FT-028: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Execution plan for FT-028 footprint-aware liveness clearing without redefining its canonical problem or solution facts."
derived_from:
  - brief.md
  - design.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_028_scope
  - ft_028_selected_design
  - ft_028_acceptance_criteria
  - ft_028_validation_profile
---

# FT-028: Implementation Plan

## Grounding and test strategy

| Path | Role / reuse |
| --- | --- |
| `internal/event/event.go` | `Logger.mu` already serializes `Emit`, `Diagnostic`, `writeTransient`, and `clearLocked`; it currently stores only `transient bool` and emits `\r` + `CSI 2K`. Extend this owner with `SOL-01`–`SOL-03`, not workflow state. |
| `internal/event/event_test.go` | Existing `signalWriter`, injected ticker, `TestTransientClearedBeforePermanentAndDiagnostic`, and `TestLivenessWriterFailureCancelsStage` provide deterministic writer/lifecycle patterns. Add a terminal-footprint model here. |
| `internal/app/app.go` / `app_test.go` | `App.IsTerminal` is already an injectable seam; production terminal detection accepts only `*os.File` character devices, while buffer-backed tests are non-TTY. Add a parallel injectable width-reader seam and prove that it is wired into the logger only for interactive human output. |
| `internal/workflow/workflow.go` / `workflow_test.go` | Each Codex-backed stage already executes `Stop()` before its next permanent event or diagnostic. Preserve this ordering; workflow should not compute terminal rows. |
| `go.mod`, `tools/build-dist/main.go`, `.github/workflows/release.yml` | The module has no dependencies today and release artifacts target `darwin/{amd64,arm64}` and `linux/{amd64,arm64}`. Any width implementation must build for both OSes and be added deliberately to module/dependency evidence. |

The `high-risk` profile from `brief.md` requires affected unit/contract/concurrency checks, race testing, and manual evidence for terminal behavior not represented by the deterministic model. `go test ./...`, `go test -race ./internal/event ./internal/workflow`, `go vet ./...`, `make docs-lint`, and `git diff --check` are required before handoff. `go test ./tools/build-dist` is required if a module dependency or platform build surface changes.

## Preconditions

| ID | Canonical refs | Required state |
| --- | --- | --- |
| `PRE-01` | `SD-01`, `SD-04`, `CTR-01` | A POSIX-TTY width reader can be injected and faked in tests; its production dependency/build path is known to compile for all four release targets. |

## Design realization mapping

| Solution refs | Target / steps | Checks / evidence |
| --- | --- | --- |
| `SOL-01`, `SD-03`, `CTR-01`, `INV-01`–`INV-02` | `internal/event`; `STEP-01`, `STEP-03` | `CHK-01`, `CHK-02`, `CHK-04`; `EVID-01`, `EVID-02`, `EVID-04` |
| `SOL-02`–`SOL-03`, `SD-01`, `SD-04`, `INV-03`, `FM-01` | `internal/event` plus app injection; `STEP-01`–`STEP-03` | `CHK-01`, `CHK-03`, `CHK-05`; `EVID-01`, `EVID-03`, `EVID-05` |
| `CTR-02`, `FM-02`, `RB-01` | event/app/workflow regression; `STEP-04` | `CHK-03`, `CHK-04`; `EVID-03`, `EVID-04` |

## Steps

| Step | Goal | Touchpoints | Verifies |
| --- | --- | --- | --- |
| `STEP-01` | Choose and add a cross-compiled POSIX terminal-size dependency/implementation; add `App` width-reader injection mirroring `IsTerminal`, with no new flag/configuration. Prove the production reader is called only for interactive human stdout. | `go.mod`, `go.sum` if created, `internal/app/app.go`, `internal/app/app_test.go`, `tools/build-dist` tests | `CHK-03` |
| `STEP-02` | Add the stored frame footprint and row-count calculation under `Logger.mu`. Start each frame at column zero, reserve the final column, and define its printable-cell calculation before writing cursor controls. | `internal/event/event.go`, `internal/event/event_test.go` | `CHK-01`, `CHK-04` |
| `STEP-03` | Implement row-aware clear/repaint: clear the stored footprint from final row to first before redraw, permanent stdout, or diagnostics; propagate unavailable width and writer errors through the existing liveness error path. | `internal/event/event.go`, `internal/event/event_test.go` | `CHK-01`, `CHK-02`, `CHK-04` |
| `STEP-04` | Add deterministic terminal-model tests: width `80 → 40` after a frame has wrapped, two or more rows, completion, diagnostic, cancellation, and first writer failure. Retain existing non-TTY/heartbeat/kv byte assertions. | `internal/event/event_test.go`, `internal/app/app_test.go`, affected workflow tests | `CHK-01`–`CHK-04` |
| `STEP-05` | Run repository gates and a manual interactive procedure: run `code-converge --mode best`, resize while a review is active, then capture completion/diagnostic output and verify no retained frame. Update README only if the accepted public wording needs clarification. | tests, terminal transcript, README if needed | `CHK-03`–`CHK-05` |

## Stop conditions

- `STOP-01`: if a terminal model or manual evidence shows the algorithm can erase unrelated history or cannot clear its owned footprint, stop implementation, restore the last safe renderer state, and return to `design.md`.
- `STOP-02`: if a required width implementation cannot compile for every released Linux/macOS target without a new public configuration/contract, stop and update `design.md`/`brief.md` before adding that surface.

## Review and grounding record

- **Grounding facts:** current interactive detection is `App.isTerminal`; current logger lifecycle and serialization are local to `internal/event`; workflow already provides the required stop-before-next-output barrier; release scope is Linux and macOS, not Windows.
- **FPF chain:** `SOL-02` is an abductive hypothesis. Its deduction is that a stored frame rendered at width 80 and then reflowed at width 40 needs every computed owned row cleared before the permanent boundary. `CHK-01` must model and refute/corroborate that prediction; `CHK-05` is separate empirical evidence for a real terminal.
- **Resolved plan findings:** made the width-reader dependency, four-target build check, exact file touchpoints, state ownership, deterministic `80 → 40` scenario, and manual command explicit. No plan-local open question remains.
