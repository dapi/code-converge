---
title: "FT-028: Design"
doc_kind: feature
doc_function: canonical
purpose: "Solution-space owner for footprint-aware interactive liveness clearing in FT-028."
derived_from:
  - brief.md
  - decision-log.md
  - ../FT-009/design.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_028_scope
  - ft_028_acceptance_criteria
  - implementation_sequence
---

# FT-028: Design

## Design Pack

| Artifact | Role | Owns |
| --- | --- | --- |
| `design.md` | Feature-local solution owner | `SOL-*`, `C4-*`, `SD-*`, `CTR-*`, `INV-*`, `FM-*`, `RB-*` |

## Context

`CSI 2K` erases one current terminal line, while a liveness frame can occupy several physical rows after wrapping or reflow. The existing logger stores only `transient: bool`, so it cannot clear the whole footprint it owns. `DL-02` selects a footprint-aware presentation coordinator without changing the permanent output contract.

## C4 Applicability

`C4-04: Code` is required. The change introduces a non-trivial state/locking contract inside the existing `internal/event.Logger`; it changes neither containers nor external interfaces. No separate diagram is needed: the affected code-level relationship is bounded to `Logger → transient footprint → terminal width source → stdout`.

## Selected Design

- `SOL-01` Extend the existing mutex-protected logger state with the printable-cell footprint of the last successfully written transient frame and an injected terminal-width reader for interactive stdout.
- `SOL-02` Before any redraw, permanent stdout record, or stderr diagnostic, clear the entire stored footprint: derive its current physical-row count from the current terminal width, erase each owned row from the frame's final row back to its first row, and leave the cursor at the first row for the next write.
- `SOL-03` Render transient frames with a reserved final cell so a frame never ends at the right margin; calculate cell width consistently with the footprint used for clearing.

### Accepted local decisions

- `SD-01` Width is queried immediately before each transient clear/redraw, not cached across a resize. The supported scope is interactive POSIX stdout for which a positive terminal column count is available.
- `SD-02` If a required width cannot be obtained or a footprint cannot be represented safely, return the liveness write error through the existing cancel/Stop path; do not emit an unsafe cursor sequence or silently switch to a newline heartbeat.
- `SD-03` The existing `Logger.mu` remains the single presentation coordinator; footprint mutation, clearing, permanent writes, and diagnostics occur under it.
- `SD-04` `internal/app` exposes an injectable width-reader seam alongside its existing `IsTerminal` seam. The production implementation obtains a current column count only from interactive `*os.File` stdout, using a Go dependency that supports both released POSIX targets (Linux and macOS); tests supply widths directly and do not require a real terminal.

## Contracts and invariants

- `CTR-01` A successful transient write atomically records its printable cell count and marks the transient active. A clear consumes that exact record and clears all rows derived from it at the terminal's current positive width.
- `CTR-02` The terminal-width reader is used only in interactive human transient mode; non-interactive human, heartbeat, and `kv` byte contracts remain unchanged.
- `INV-01` A permanent stdout record or stderr diagnostic is never written while an owned transient footprint remains active.
- `INV-02` No transient write occurs after `Liveness.Stop()` has joined the worker.
- `INV-03` The renderer never deliberately writes a frame into the terminal's final column.

## Failure modes and backout

| ID | Failure mode / action |
| --- | --- |
| `FM-01` | Width lookup is unavailable, non-positive, or frame measurement is unsafe: cancel the stage through the existing liveness error path and return operational failure. |
| `FM-02` | A clear/write fails: preserve first-error semantics, stop/join the worker, and do not issue later liveness writes. |
| `RB-01` | Backout is a normal code revert; no migration, persisted state, config, or release rollout is introduced. |

## Architecture coverage and design verification

| Aspect | Decision | Evidence |
| --- | --- | --- |
| Components/connectors | `internal/app` supplies a width reader; `internal/event` owns frame measurement and clearing; workflow remains unchanged except existing error handling. | `CHK-01`–`CHK-04` |
| Configuration | N/A; no new public setting. | `CHK-03` |
| Behavioral semantics | Covered by `CTR-01`–`CTR-02` and `INV-01`–`INV-03`. | `CHK-01`, `CHK-02`, `CHK-04` |
| Quality/evolution | Concurrency and compatibility are high-risk; the state is local and revertible. | race run, `CHK-03`–`CHK-05` |

## Traceability

| Requirement | Solution refs | Contracts / invariants | Failure / backout |
| --- | --- | --- | --- |
| `REQ-01`, `REQ-04` | `SOL-01`–`SOL-03`, `C4-04`, `SD-01` | `CTR-01`, `INV-01`, `INV-03` | `FM-01`, `RB-01` |
| `REQ-02` | `SOL-01`, `SD-02` | `CTR-02` | `FM-01`, `RB-01` |
| `REQ-03` | `SOL-01`, `SD-03` | `INV-02` | `FM-02` |
