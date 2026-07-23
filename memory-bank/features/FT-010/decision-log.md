---
title: "FT-010: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Auditable FPF reasoning, review-improve cycles, and human gates for FT-010."
derived_from:
  - brief.md
  - https://github.com/dapi/code-converge/issues/10
status: active
audience: humans_and_agents
---

# FT-010: Decision Log

## Artifact contract

- **Role:** provenance companion for FPF decisions, document review, and gate status.
- **Owns:** source facts, alternatives, reasoning, confidence, document conflicts, and human gates.
- **Must not define:** requirements, the public CLI/terminal contract, selected solution, or execution sequence. Accepted facts must move to `brief.md`, `design.md`, or the root README.

## FPF method

The review separates four bounded decision contexts: public activation and keys, safe presentation of an untrusted process stream, operator navigation/state semantics, and terminal-capability implementation. It evaluates each against source facts only. The current sources establish required outcomes but intentionally leave the choices among several externally observable alternatives open; therefore the framework preserves the unknown instead of inventing a contract.

## Decision entries

### `DL-01` — Route and assurance floor

- **Status:** resolved; promoted to `brief.md`.
- **Question:** Which delivery flow and validation profile apply?
- **Facts:** Issue #10 changes the operator-facing CLI interaction model and requires an interactive terminal presentation while preserving non-interactive stdout. The routing rules exclude Small Change when CLI behavior or stdout contracts change. The issue does not record financial, security/auth, persistent-data, migration, concurrency, or cross-system integration triggers.
- **Alternatives:** Small Change/low-risk; Feature Flow/standard; Feature Flow/high-risk.
- **FPF reasoning:** Lifecycle route and validation depth are separate decisions. This is one independently verifiable change to observable executable behavior and must retain a compatibility regression path, which needs Feature Flow and `standard` evidence. No sourced trigger justifies `high-risk`.
- **Result:** `FT-010` follows Feature Flow with `standard` validation and no downgrade.
- **Confidence:** high; direct issue, routing, and validation-policy facts.

### `DL-02` — Blocked interactive-terminal contract

- **Status:** superseded by resolved `DL-03`; it remains as the provenance of `HG-01` and `DEC-01`.
- **Question:** What exact public contract governs interactive eligibility/activation, closing, lower-pane output representation, scrollback, inactive-stage behavior, and minimum terminal capabilities/library?
- **Facts:** The issue requires `i` to open a split view during an interactive run, concurrent workflow/agent updates, and clean restoration. It explicitly lists as open questions whether `i` toggles or a separate close key is required; raw versus sanitized output; fixed/scrollable/complete upper history; `i` with no active agent; default-on-TTY versus an explicit flag; and a terminal library/minimum capabilities. The issue requires definitions for resize, scrolling, long lines, ANSI/control sequences, completion, errors, and inactive stages. The root README describes current human and `kv` logging plus liveness only; it has neither an interactive input contract nor a pane/terminal-capability contract. The runner currently returns buffered stdout/stderr after completion, and architecture requires raw agent output to remain outside workflow stdout.
- **Alternatives:**
  - **A:** Enable automatically when the required terminal streams are TTYs; let `i` toggle the view; render a sanitized stream; expose bounded scrollback and an explicit inactive/completion state; use a selected terminal implementation with documented fallback.
  - **B:** Require an explicit interactive flag; use a distinct close key; choose raw or ANSI-preserving output and a defined scrollback model; document a capability floor and fallback.
  - **C:** Another coherent contract that answers every listed issue question and preserves non-interactive stdout isolation.
- **FPF reasoning:** The bounded contexts are coupled but not interchangeable: activation decides who receives the new UI; key behavior decides the user interaction; stream representation decides safety/fidelity; scrollback and inactive behavior decide observable state; the capability/library choice decides portability and cleanup mechanics. Current documents prove the non-interactive preservation constraint, but none selects a compatibility/default policy, a terminal-safety policy, or a capability floor. Choosing A or B would create new product requirements, public documentation, and test obligations unsupported by evidence. This is not a missing implementation detail that an architecture document can settle downstream.
- **Result at the time:** Stop before `design.md`, `implementation-plan.md`, implementation, or updates to the root public contract. `DL-03` subsequently selected the coherent contract under the user's decision authority.
- **Confidence:** high that a product/contract decision is missing; direct issue wording and current README/architecture absence.

### `DL-03` — FPF resolution of the interactive-terminal contract

- **Status:** resolved; promoted to `design.md`, with reusable runtime boundary accepted in `ADR-001`.
- **Question:** Which smallest coherent terminal contract satisfies Issue 10 without changing the non-interactive workflow contract?
- **Facts:** The issue title says the view is toggled by `i`; the issue requires live split panes, scroll/resize/long-line/control-sequence/inactive/completion/error semantics and clean restoration, while preserving non-interactive stdout. The root README already scopes transient liveness to human mode on interactive stdout and requires no ANSI in non-TTY output. The architecture isolates raw Codex output from workflow stdout and currently has no TUI runtime. The CLI currently has a Unix/Windows process split and no third-party dependencies.
- **Candidates:** (A) auto-enable on eligible terminals, `i` toggle, sanitized bounded-scrollback panes, explicit inactive/completion states, and a narrow cross-platform terminal primitive; (B) new opt-in flag, separate close key, raw/ANSI-preserving stream, and a full TUI framework; (C) no interactive view outside explicitly configured terminals.
- **FPF filters:**
  - **Consistency:** A preserves the existing human/TTY distinction, non-interactive `kv` contract, and raw-output isolation; B adds a configuration/public-format surface; C fails the issue's interactive goal.
  - **Parsimony:** A uses the issue's named key and no new persistent setting; B adds both user-facing and framework surface not required by the source facts.
  - **Probeability:** A yields deterministic eligibility, stream-identity, buffer, cleanup, and non-interactive test matrices; raw ANSI and unbounded history make B's safety and memory behavior less determinate.
  - **Scope fit:** A changes only the terminal presentation boundary and its runner feed; it does not change workflow decisions, output schema, or agent policy.
- **Result:** Accept A. The view is eligible only in `human` mode with terminal stdin/stdout and non-dumb `TERM`; `i` toggles it. Two independently bounded 2,000-line panes support `Tab`, arrows/Page, and `End`; the agent stream is sanitized plain text, tagged by identity, and never mixed or forwarded to workflow stdout. Inactive/completion/error states are explicit. `golang.org/x/term` supplies only cross-platform terminal mode/capability primitives; local code owns presentation. `ADR-001` records the reusable part.
- **Confidence:** medium-high. The outcome/constraints are direct source facts; exact buffer size and dependency are selected by the stated FPF filters under the user's instruction to decide, and are explicitly testable/revisable design choices.

`HG-01` is closed by `DL-03`.

## Human gates

### `HG-01` — Accept the interactive-terminal contract

- **Status:** closed by `DL-03` on 2026-07-23.

- **Question:** Which exact contract should FT-010 implement for (1) activation eligibility/default versus explicit opt-in, (2) close behavior, (3) safe agent-stream representation, (4) pane history/scrolling and no-active/completed/error states, and (5) minimum terminal capabilities and library/fallback policy?
- **Available facts:** Issue #10 requires `i` to open a live split view without interrupting the workflow; workflow records and active agent output remain separated; resize, scrolling, long lines, ANSI/control sequences, completion/errors/inactive stages, and cleanup must be defined. Non-interactive machine-readable stdout and raw-output isolation are preserved. Existing public docs specify neither input activation nor terminal capability semantics; current runner output is buffered after process completion.
- **Options:** A — automatic eligible-TTY activation with `i` toggle, sanitized bounded-scrollback view and documented fallback; B — explicit opt-in with a distinct close key and a separately selected raw/sanitized/ANSI policy; C — another complete contract specifying each listed dimension.
- **Risk of an incorrect choice:** A wrong default/key policy can disrupt existing interactive operators; unsafe stream rendering can execute or misrender control data; an unsuitable capability/library choice can leave terminals corrupted or make the feature unavailable in supported environments; an undefined history/state policy prevents reliable tests and documentation.
- **Resolution:** The user's instruction to use FPF for the decision authorized `DL-03`; the selected contract is now canonical in `design.md`, ADR-001, and the root README.
