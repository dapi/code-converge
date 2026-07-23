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
| Codex boundary (`internal/codex`) | Command invocation with a prepared review target, plain-text and strictly structured review-report classification, strict finalization response parsing | Exit-code policy and workflow stdout formatting |
| Repository status and review discovery (`internal/repository`) | Git status query plus deterministic base discovery and a disposable merge-base-to-worktree index snapshot | Workflow transition policy and Codex-output interpretation |
| Workflow orchestration (`internal/workflow`) | State transitions, budgets, stage timing and exit outcomes | Subprocess mechanics |
| Process runner (`internal/runner`) | Working directory, context cancellation, captured stdin/stdout/stderr and exit status | Agent-report interpretation |
| Progress presentation (`internal/event`) | Structured/human stdout rendering, duration/count formatting, terminal liveness and serialized clearing/diagnostic coordination defined by the root README | Workflow decisions and raw agent output |

Review uses normal `codex review --base` output without requiring a caller-supplied schema. Repository discovery ties provider PR candidates to one host-aware identity, retaining a non-default URL port, from every configured push URL. Every URL must identify that same provider destination before it is authoritative; conflicting valid destinations fail before provider discovery, while an all-non-provider or mixed provider/non-provider set leaves local discovery available. It falls back to `origin` when the normal Git push-remote settings are absent, scopes `gh` to the verified identity, retains the queried PR owner to resolve its remote-tracking base, and refreshes a private `GIT_INDEX_FILE` snapshot before each review so committed, staged, unstaged and untracked changes are one review target without changing the real index or worktree. The Codex boundary forces a wrapper-prefixed `PATH` through `shell_environment_policy.set` and disables login-shell startup for the review, so profile initialization cannot discard the scoped transport. Its private root, index and Git executable are sidecar data next to the wrapper, so an `include_only` policy that permits `PATH` cannot discard the scoped transport. `GIT_INDEX_FILE` is never exported to Codex. The PATH wrapper directory contains only the symlinked `git` helper, which runs from the installed executable rather than the temporary directory; all `git-*` helpers are linked into a separate child-only `GIT_EXEC_PATH` directory, and setup fails before review if either temporary directory contains a platform path-list separator. The helper resolves documented Git global options including both `--namespace` and `--attr-source` forms plus `--list-cmds=<group>`, while unknown or malformed options fail closed. It applies the private index only after it confirms the reviewed repository; other targets, repository-creation commands, and unclassifiable external subcommands use their normal index. It sets `GIT_EXEC_PATH` only within its child Git process so aliases and hooks continue through the wrapper without exposing that setting to Codex policy. All policy selections other than the review-only `PATH` and login-shell override remain intact. It recognizes the established plain-text reports and the exact validated structured response shape from supported Codex CLI output; ambiguous, malformed or non-zero output becomes an operational failure. After a clean classification, the repository-status collaborator determines whether finalization is applicable. Finalization keeps the exact verdict contract from the root README because that verdict controls workflow transitions.

External process execution is a trust boundary. The runner preserves the operator's invocation directory, captures stdin/stdout/stderr, propagates context cancellation, and never forwards raw Codex output to workflow stdout. Code-Converge does not add a timeout or override Codex sandbox, approval, or network configuration. Publication behavior remains hosting-provider-neutral.

Configuration has one resolver. The public option names, sources, and defaults are owned by [`../../README.md`](../../README.md); operational prerequisites are in [`../ops/config.md`](../ops/config.md). Initial delivery rationale is traceable through [`../features/FT-002/design.md`](../features/FT-002/design.md); human/structured presentation and liveness concurrency are traceable through [`../features/FT-009/design.md`](../features/FT-009/design.md).
