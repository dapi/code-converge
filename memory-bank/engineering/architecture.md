---
title: Engineering Architecture
doc_kind: engineering
doc_function: canonical
purpose: Architecture ownership for the reviewer CLI.
derived_from:
  - ../domain/model.md
  - ../domain/states.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - architecture_patterns
  - module_boundaries
---

# Engineering Architecture

The product is a dependency-free Go CLI that coordinates a sequential state machine. The module is `github.com/dapi/reviewer`; concrete package boundaries implement the responsibility split below.

| Responsibility | Owns | Must remain separate from |
| --- | --- | --- |
| CLI boundary (`cmd/reviewer`, `internal/app`) | Argument parsing, command selection, signal context, dependency wiring | Workflow transition policy and agent-report interpretation |
| Configuration resolution (`internal/config`) | Settings sources, precedence, source metadata, validated Git root | Ad hoc per-stage configuration lookup |
| Codex boundary (`internal/codex`) | Command invocation, ordinary review-report classification, strict finalization response parsing | Exit-code policy and workflow stdout formatting |
| Workflow orchestration (`internal/workflow`) | State transitions, budgets, stage timing and exit outcomes | Subprocess mechanics |
| Process runner (`internal/runner`) | Working directory, context cancellation, captured stdin/stdout/stderr and exit status | Agent-report interpretation |
| Event rendering (`internal/event`) | The one-line stdout encoding defined by the root README | Workflow decisions and raw agent output |

Review uses normal `codex review` output without requiring JSON or a caller-supplied schema. The Codex boundary must recognize an explicitly clean report or concrete finding entries; ambiguous or non-zero output becomes an operational failure. Finalization keeps the exact verdict contract from the root README because that verdict controls workflow transitions.

External process execution is a trust boundary. The runner preserves the operator's invocation directory, captures stdin/stdout/stderr, propagates context cancellation, and never forwards raw Codex output to workflow stdout. Reviewer does not add a timeout or override Codex sandbox, approval, or network configuration. Publication behavior remains hosting-provider-neutral.

Configuration has one resolver. The public option names, sources, and defaults are owned by [`../../README.md`](../../README.md); operational prerequisites are in [`../ops/config.md`](../ops/config.md). Feature-local rationale and contract details are traceable through [`../features/FT-002/design.md`](../features/FT-002/design.md).
