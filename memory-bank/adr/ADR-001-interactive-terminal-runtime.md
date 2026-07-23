---
title: "ADR-001: Interactive terminal runtime"
doc_kind: adr
doc_function: canonical
purpose: "Records the reusable terminal input, capability, and restoration boundary for interactive CLI presentation."
derived_from:
  - ../features/FT-010/brief.md
  - ../features/FT-010/design.md
status: active
decision_status: accepted
date: 2026-07-23
audience: humans_and_agents
must_not_define:
  - current_system_state
  - implementation_plan
---

# ADR-001: Interactive terminal runtime

## Context

FT-010 needs to receive a single-key command while preserving the current CLI's deterministic non-interactive stdout contract and restoring the terminal on every exit path. The existing CLI has no input/pane runtime and is dependency-free; its `process_unix.go` and `process_windows.go` portability split shows that direct terminal handling would otherwise recreate platform-specific code.

## Decision drivers

- Interactive behavior must be absent when either terminal endpoint is unavailable or unsuitable.
- The input/runtime abstraction must be deterministic under fakes and must restore terminal state after normal completion, cancellation, panic, and setup failure.
- The change must not make terminal state or raw agent bytes part of workflow stdout.
- The chosen boundary should be reusable by later terminal-only operator surfaces without introducing a complete TUI framework.

## Considered options

| Option | Advantages | Disadvantages | Decision |
| --- | --- | --- | --- |
| Standard-library/platform syscalls | Keeps third-party dependencies at zero. | Duplicates raw-mode and restoration behavior across supported operating systems and makes testing/cleanup ownership less clear. | Rejected. |
| `golang.org/x/term` plus an in-repository renderer | Provides cross-platform terminal mode/capability primitives while keeping layout, buffers, event mapping, and output policy local. | Adds one focused module dependency and requires an explicit capability/fallback policy. | Accepted. |
| Full terminal UI framework | Provides widgets and input routing. | Adds an opinionated runtime and larger dependency/API surface than the two-pane feature needs. | Rejected. |

## Decision

Use `golang.org/x/term` only for terminal detection and raw-mode restoration. Keep the presentation coordinator, pane state, stream sanitizer, layout, key mapping, and test fakes in repository-owned packages. Treat interactive presentation as eligible only when `log-format=human`, stdin and stdout are terminals, and `TERM` is neither empty nor `dumb`; otherwise retain the existing permanent human output without an error or prompt. The runtime owns restoration with a single idempotent cleanup path, including panic recovery at the application boundary.

## Consequences

### Positive

- One narrow cross-platform dependency replaces duplicated terminal-mode mechanics.
- Interactive and non-interactive contracts remain cleanly separated.
- A small, testable terminal boundary can be reused without adopting a general TUI framework.

### Negative

- The project is no longer strictly dependency-free once FT-010 is implemented.
- Unsupported terminals receive no interactive view and must use the existing output.

### Neutral / organizational

- Implementation must update `go.mod`, root README, and architecture documentation atomically with the feature.
- The chosen dependency must be covered by the repository's normal module and CI verification.

## Risks and mitigation

- Raw mode can leave a terminal unusable: enforce one idempotent restore path and deterministic cleanup tests.
- Terminal capability detection can be wrong: use conservative eligibility and preserve existing output as fallback.
- Agent output can contain terminal controls: sanitize before it enters pane state; never pass it to workflow stdout.

## Follow-up

- FT-010 `design.md` defines the feature-specific interactions, stream contract, and failure modes.
- FT-010 `implementation-plan.md` sequences the dependency addition, runtime wiring, deterministic tests, and manual smoke evidence.

## Related links

- [FT-010 brief](../features/FT-010/brief.md)
- [FT-010 design](../features/FT-010/design.md)
- [FT-010 implementation plan](../features/FT-010/implementation-plan.md)
