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

The confirmed product shape is a local CLI that coordinates a sequential state machine. No Go module, package layout, dependency set, or process-runner abstraction has been selected yet. The table below records conceptual responsibilities required by the public contract; it does not prescribe package paths or authorize implementation.

| Responsibility | Owns | Must remain separate from |
| --- | --- | --- |
| CLI boundary | Argument parsing and command selection | Workflow transition policy and agent-report interpretation |
| Configuration resolution | Settings sources, precedence, and source metadata | Ad hoc per-stage configuration lookup |
| Codex boundary | Command invocation, ordinary review-report classification, and finalization response parsing | Exit-code policy and workflow stdout formatting |
| Workflow orchestration | State transitions and stage timing | Subprocess mechanics |
| Event rendering | The one-line stdout encoding defined by the root README | Workflow decisions and raw agent output |

Review uses normal `codex review` output without requiring JSON or a caller-supplied schema. The Codex boundary must recognize an explicitly clean report or concrete finding entries; ambiguous or non-zero output becomes an operational failure. Finalization keeps the exact verdict contract from the root README because that verdict controls workflow transitions.

External process execution is a trust boundary. The implementation must control the working directory and capture stdin/stdout/stderr so raw Codex output cannot bypass the event renderer; credentials must never be logged. Timeout, cancellation, sandbox, network, and concrete runner policies remain undecided and must be resolved by the independently scoped implementation work. Publication behavior remains hosting-provider-neutral.

Configuration has one resolver. The public option names, sources, and defaults are owned by [`../../README.md`](../../README.md); operational prerequisites are in [`../ops/config.md`](../ops/config.md). Exact package topology and dependency choices remain open until implementation is explicitly scoped.
