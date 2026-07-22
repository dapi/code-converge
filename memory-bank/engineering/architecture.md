---
title: Engineering Architecture
doc_kind: engineering
doc_function: canonical
purpose: Architecture ownership for the code-converge CLI.
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

The product is a dependency-free Go CLI that coordinates a sequential state machine. The module is `github.com/dapi/code-converge`; concrete package boundaries implement the responsibility split below.

| Responsibility | Owns | Must remain separate from |
| --- | --- | --- |
| CLI boundary (`cmd/code-converge`, `internal/app`) | Argument parsing, command selection, signal context, dependency wiring | Workflow transition policy and agent-report interpretation |
| Configuration resolution (`internal/config`) | Settings sources, precedence, source metadata, validated Git root | Ad hoc per-stage configuration lookup |
| Codex boundary (`internal/codex`) | Command invocation, plain-text and strictly structured review-report classification, strict finalization response parsing | Exit-code policy and workflow stdout formatting |
| Repository status (`internal/repository`) | Git status query for staged, unstaged and untracked changes | Workflow transition policy and Codex-output interpretation |
| Workflow orchestration (`internal/workflow`) | State transitions, budgets, stage timing and exit outcomes | Subprocess mechanics |
| Process runner (`internal/runner`) | Working directory, context cancellation, captured stdin/stdout/stderr and exit status | Agent-report interpretation |
| Progress presentation (`internal/event`) | Structured/human stdout rendering, duration/count formatting, terminal liveness and serialized clearing/diagnostic coordination defined by the root README | Workflow decisions and raw agent output |

Review uses normal `codex review` output without requiring a caller-supplied schema. The Codex boundary recognizes the established plain-text reports and the exact validated structured response shape from supported Codex CLI output; ambiguous, malformed or non-zero output becomes an operational failure. After a clean classification, the repository-status collaborator determines whether finalization is applicable. Finalization keeps the exact verdict contract from the root README because that verdict controls workflow transitions.

External process execution is a trust boundary. The runner preserves the operator's invocation directory, captures stdin/stdout/stderr, propagates context cancellation, and never forwards raw Codex output to workflow stdout. Code-Converge does not add a timeout or override Codex sandbox, approval, or network configuration. Publication behavior remains hosting-provider-neutral.

Configuration has one resolver. The public option names, sources, and defaults are owned by [`../../README.md`](../../README.md); operational prerequisites are in [`../ops/config.md`](../ops/config.md). Initial delivery rationale is traceable through [`../features/FT-002/design.md`](../features/FT-002/design.md); human/structured presentation and liveness concurrency are traceable through [`../features/FT-009/design.md`](../features/FT-009/design.md).
