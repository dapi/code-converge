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
| Codex boundary (`internal/codex`) | Schema-constrained command invocation with a prepared review target, strict final-response-file classification, strict finalization response parsing | Exit-code policy and workflow stdout formatting |
| Repository status and review discovery (`internal/repository`) | Git status query, local findings-fix checkpoint commit, deterministic base discovery and a disposable merge-base-to-worktree index snapshot | Workflow transition policy, remote publication, and Codex-output interpretation |
| Workflow orchestration (`internal/workflow`) | State transitions, budgets, stage timing and exit outcomes | Subprocess mechanics |
| Process runner (`internal/runner`) | Working directory, context cancellation, captured stdin/stdout/stderr and exit status | Agent-report interpretation |
| Progress presentation (`internal/event`) | Structured/human stdout rendering, duration/count formatting, terminal liveness and serialized clearing/diagnostic coordination defined by the root README | Workflow decisions and raw agent output |

Review uses `codex exec` with a caller-supplied strict schema and per-invocation final-message file. Repository discovery resolves the intended base and refreshes a private `GIT_INDEX_FILE` snapshot before each review so committed, staged, unstaged and untracked changes are one merge-base-to-worktree target without changing the real index or worktree. The adapter passes that path both as process environment and as an invocation-local Codex spawned-tool override. After a zero process exit, the Codex boundary classifies only the exact validated structured response file; terminal streams, prose, missing/invalid files and non-zero invocations cannot select a result. Before automatic remediation, the repository collaborator checks whether a checkpoint can safely be attributed to the fix stage. A dirty baseline continues remediation but skips checkpointing; a clean baseline may create a local checkpoint commit and never publishes it. After a clean classification, repository status or a run-local checkpoint determines whether finalization is applicable. Finalization keeps the exact verdict contract from the root README because that verdict controls workflow transitions and is the only publication path.

External process execution is a trust boundary. The runner preserves the operator's invocation directory, captures stdin/stdout/stderr, propagates context cancellation, and never forwards raw Codex output to workflow stdout. Code-Converge does not add a timeout or override Codex sandbox, approval, or network configuration. Publication behavior remains hosting-provider-neutral.

Configuration has one resolver. The public option names, sources, and defaults are owned by [`../../README.md`](../../README.md); operational prerequisites are in [`../ops/config.md`](../ops/config.md). Initial delivery rationale is traceable through [`../features/FT-002/design.md`](../features/FT-002/design.md); human/structured presentation and liveness concurrency are traceable through [`../features/FT-009/design.md`](../features/FT-009/design.md).
