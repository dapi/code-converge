---
title: "FT-028: Decision Log"
doc_kind: feature
doc_function: supporting
purpose: "Auditable routing, FPF reasoning, review-improve records, and human gates for issue #28. It does not own feature scope, selected design, or execution sequencing."
derived_from:
  - brief.md
  - ../../flows/routing.md
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../FT-009/design.md
  - ../../../README.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_028_scope
  - selected_terminal_control_mechanism
  - implementation_sequence
---

# FT-028: Decision Log

## Decisions

### `DL-01` — Feature Flow routing override

- **Status:** resolved by explicit user instruction.
- **Question:** Which delivery flow governs issue #28?
- **Facts:** The issue itself says Bug Fix Flow because observed behavior contradicts an accepted contract. Canonical task routing likewise selects Bug Fix Flow for such a defect. The user explicitly instructed this worktree to take issue #28 through Feature Flow. The requested scope is one independently verifiable delivery unit affecting stdout/concurrency behavior; AGENTS.md requires a feature package for CLI/stdout/architecture changes.
- **Alternatives:** Bug Fix Flow; Feature Flow; Human Routing.
- **FPF reasoning:** The routing taxonomy classifies the work as a bug, but the governing user instruction selects the lifecycle representation. The unit remains bounded and does not require an epic. Treating the explicit instruction as a process override changes no product behavior and retains Feature Flow's stronger problem/design/plan gates for a concurrency-sensitive stdout correction.
- **Result:** Use FT-028 under Feature Flow. `brief.md` records this rationale; the issue's defect scope and acceptance criteria remain authoritative.
- **Confidence:** high.

### `DL-02` — Terminal-safe clearing mechanism

- **Status:** resolved by the user's authorization to choose with FPF.
- **Question:** Which terminal-control mechanism can reliably clear all owned interactive liveness display rows after resize/reflow without changing the public liveness contract?
- **Facts:** Issue #28 reports retained frames after resize, reflow, or scrolling and requires root-cause confirmation before choosing a fix. `internal/event.Logger.writeTransient` writes `\r` + `CSI 2K` + an unbounded frame; `clearLocked` writes only `\r` + `CSI 2K`. The logger tracks a boolean transient state, not the printable width, physical row footprint, terminal size, or cursor position. Existing tests assert byte sequences and stop/join behavior but do not model wrapping or reflow. FT-009 requires an in-place interactive line, clearing before permanent output/diagnostics, and no late writes. No current document names supported terminal emulators, resize semantics, width source, Unicode-width policy, or an acceptable degradation rule.
- **Alternatives:**
  - **A:** Add a width/footprint-aware renderer that tracks and clears every owned physical row across resize/reflow.
  - **B:** Use a constrained single-row rendering policy, with an explicit safe fallback when the terminal dimensions cannot keep the frame unwrapped.
  - **C:** Keep the current escape sequence and add only synchronization tests.
  - **D:** Remove interactive liveness or replace it with newline records.
- **FPF reasoning:** Bounded contexts separate the accepted product contract (FT-009/README), renderer state, and terminal display semantics. The abductive hypothesis is A: stale rows are owned display footprint that `CSI 2K` cannot clear because it erases a line, not a wrapped footprint. Deduction predicts that (1) an old frame that spans N physical rows must be cleared row-by-row before permanent output, and (2) a resize must recompute N from the current terminal width and the stored printable footprint. C cannot address that missing state; D contradicts `REQ-02`; B cannot repair a frame that already wrapped before a later resize. Induction is `CHK-01`'s deterministic terminal model plus `CHK-05` manual evidence. XTerm documents `CSI 2K` as *Erase in Line* and supports automatic wrapping of long lines, which independently supports the hypothesis but does not broaden the supported-terminal claim. [XTerm control sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
- **Result:** Select A. The renderer records the printable-cell footprint of its last transient frame and, while holding the existing presentation mutex, clears every row it owns from its current final row to its first row before redraw, permanent stdout, or a diagnostic. It queries the current width on each clear/redraw; an unavailable or invalid width is a liveness operational failure rather than an unsafe best-effort cursor sequence. `design.md` owns the exact invariant and `implementation-plan.md` owns realization/testing.
- **Confidence:** medium. The causal mechanism is high-confidence from source plus terminal semantics; correctness across the selected POSIX-TTY width model must be raised through the required deterministic and manual evidence.

## Review-Improve Cycles

### Cycle 1 — bootstrap package integrity review

- **Review scope:** issue #28; task routing and Feature Flow; FT-009 `CTR-04`/`INV-04`; root README liveness contract; `internal/event` and workflow liveness boundary; the newly instantiated FT-028 core documents.
- **Critical:** none.
- **Important:** `IMP-01` a selected terminal-clearing strategy would be required for downstream design and execution, but the available documents provide no supported terminal semantics, width/reflow model, or acceptable fallback; choosing one would invent a material contract.
- **FPF closures:** `DL-01` closes the process-routing conflict. `DL-02` initially identified insufficient evidence and is resolved by the user's explicit authorization to choose through FPF.
- **Changes:** created bootstrap-safe `README.md`, canonical `brief.md`, and this decision log; added the issue-derived verification inventory. Downstream artifacts are created in Cycle 2 after the authorized decision.
- **Minor:** current tests do not model reflow, but it is included in `IMP-01` and not independently remediated.
- **Human gate:** initially yes; resolved by the user's subsequent instruction.

### Cycle 2 — FPF solution convergence review

- **Review scope:** FT-028 after the authorized FPF choice; issue #28, FT-009 contract, current renderer, and Feature Flow downstream gates.
- **Critical:** none.
- **Important:** `IMP-02` `CHK-01` still named the removed `DEC-01`, contradicting the resolved decision and making the verify contract appear blocked.
- **FPF closures:** `DL-02` selects and bounds the footprint-aware clearing hypothesis; its deductions are explicitly mapped to deterministic and manual evidence instead of being treated as proven portability.
- **Changes:** resolved `DEC-01` by removing it from the problem owner; added `design.md` and `implementation-plan.md`; indexed them from the feature README; corrected `CHK-01` to reference the selected mechanism.
- **Minor:** the existing contract does not state a supported terminal matrix; design constrains the implementation to a POSIX TTY width source and records failed width discovery as operational failure.
- **Human gate:** no.

### Cycle 3 — final package convergence review

- **Review scope:** all FT-028 artifacts, canonical-ID traceability, selected design versus execution plan, and documentation-link/diff hygiene.
- **Critical:** none.
- **Important:** none.
- **FPF closures:** none required; `DL-02` remains explicitly evidence-bounded.
- **Changes:** none.
- **Minor:** none changed.
- **Verification:** `make docs-lint` and `git diff --check` passed.
- **Human gate:** no; review-improve stopped early by rule.

### Plan grounding review — project baseline

- **Review scope:** `implementation-plan.md` against `internal/app`, `internal/event`, `internal/workflow`, their tests, `go.mod`, build targets, and testing policy.
- **Important:** the prior plan named a terminal-width reader but did not ground its dependency, injection seam, platform support, or deterministic resize case in the repository.
- **FPF closure:** the footprint-clearing design remains an abductive hypothesis. The plan now states its deductive prediction (80-column frame reflowed to 40 columns clears every owned row) and links the deterministic model and manual terminal procedure as separate inductive evidence.
- **Changes:** added concrete source/test touchpoints, the existing `IsTerminal` seam as the reuse pattern, Linux/macOS release scope, deliberate dependency/build validation, explicit `80 → 40` scenario, and stop condition.
- **Human gate:** no; this grounding does not change scope or public contract.

## Implementation Review-Fix Loop

### Iteration 1 — footprint implementation and deterministic reflow

- **Critical:** none.
- **High:** the first regression covered only clearing at completion; it did not prove that a width change between liveness frames clears the old multi-row footprint before redraw.
- **Fix:** the deterministic test now renders a long frame at 80 columns, changes the injected width to 40 columns, drives a tick, and asserts row-by-row clearing before the next frame. It also preserves the completion boundary assertion.
- **Verification:** `go test ./...`; `go test -race ./internal/event ./internal/workflow`; `go vet ./...`; `go test ./tools/build-dist`; Linux/macOS amd64/arm64 cross-builds; `make docs-lint`; `git diff --check`.

### Iteration 2 — local convergence

- **Critical:** none.
- **High:** none after self-review of the affected app/event/workflow paths, dependency diff, deterministic reflow coverage, and validation output.
- **Result:** local checks are green. Hosted CI and independent PR review remain required before `delivery_status: done`.

## Human Gate

No unresolved question currently requires a human decision. `HG-01` was resolved by the user's instruction to use FPF for the choice; `DL-02` records the selected scope and the evidence required to validate it.
