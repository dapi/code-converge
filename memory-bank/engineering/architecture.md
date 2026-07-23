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

The product is a Go CLI that coordinates a sequential state machine. It uses `golang.org/x/term` only for the interactive terminal capability/raw-mode boundary accepted in [`ADR-001`](../adr/ADR-001-interactive-terminal-runtime.md); all workflow, presentation, and layout policy remains repository-owned. The module is `github.com/dapi/code-converge`; concrete package boundaries implement the responsibility split below.

| Responsibility | Owns | Must remain separate from |
| --- | --- | --- |
| CLI boundary (`cmd/code-converge`, `internal/app`) | Argument parsing, command selection, signal context, dependency wiring | Workflow transition policy and agent-report interpretation |
| Configuration resolution (`internal/config`) | Settings sources, precedence, source metadata, validated Git root | Ad hoc per-stage configuration lookup |
| Codex boundary (`internal/codex`) | Schema-constrained command invocation with a prepared review target, strict final-response-file classification, strict finalization response parsing | Exit-code policy and workflow stdout formatting |
| Repository status and review discovery (`internal/repository`) | Git status query, local findings-fix checkpoint commit, deterministic base discovery and a disposable merge-base-to-worktree index snapshot | Workflow transition policy, remote publication, and Codex-output interpretation |
| Workflow orchestration (`internal/workflow`) | State transitions, budgets, stage timing and exit outcomes | Subprocess mechanics |
| Process runner (`internal/runner`) | Working directory, context cancellation, captured stdin/stdout/stderr, live observer chunks, exit status and private-stage context | Agent-report interpretation or terminal layout |
| Diagnostic session records (`internal/session`) | Private best-effort Codex invocation records, redaction, permissions and bounded retention cleanup | Workflow result policy and public stdout schema |
| Progress presentation (`internal/event`, `internal/terminal`) | Structured/human stdout rendering, duration/count formatting, terminal liveness, interactive pane state, raw-mode restoration and serialized clearing/diagnostic coordination defined by the root README | Workflow decisions, agent-report interpretation and raw output forwarding to workflow stdout |

Review uses `codex exec` with a caller-supplied strict schema and per-invocation final-message file. Repository discovery ties provider PR candidates to one host-aware identity, retaining a non-default URL port, from every configured push URL. Every URL must identify that same provider destination before it is authoritative; conflicting valid destinations fail before provider discovery, while an all-non-provider or mixed provider/non-provider set leaves local discovery available. It falls back to `origin` when the normal Git push-remote settings are absent, scopes `gh` to the verified identity, retains the queried PR owner to resolve its remote-tracking base, and refreshes a private index snapshot before each review so committed, staged, unstaged and untracked changes are one merge-base-to-worktree target without changing the real index or worktree.

The Codex boundary forces a wrapper-prefixed `PATH` plus neutral `SHELL`, `ZDOTDIR`, `BASH_ENV`, and `ENV` values through `shell_environment_policy.set`, disables login-shell startup, and removes inherited Git repository/index/config transports and exported shell functions for the review. Login and non-login startup files or caller state therefore cannot discard, replace, or redirect the scoped transport. Its private root, index and Git executable are sidecar data next to the wrapper, so an `include_only` policy that permits `PATH` needs no additional helper variables. `GIT_INDEX_FILE` is never exported to Codex. The PATH wrapper directory contains only the symlinked `git` helper, which runs from the installed executable rather than the temporary directory; all `git-*` helpers are linked into a separate child-only `GIT_EXEC_PATH` directory, and setup fails before review if either temporary directory contains a platform path-list separator or any sidecar path cannot be represented losslessly as UTF-8. The helper resolves documented Git global options including both `--namespace` and `--attr-source` forms plus `--list-cmds=<group>`, while unknown or malformed options fail closed. It rejects reviewed-root commands and aliases that explicitly enable split-index before Git can create shared-index state. It applies the private index only after confirming the reviewed repository; other targets, repository-creation commands, and unclassifiable external subcommands use their normal index. It sets `GIT_EXEC_PATH` only within its child Git process so aliases and hooks continue through the wrapper without exposing that setting to Codex policy. Commands classified outside the review index carry a child-only no-index marker so helpers such as `git-submodule` cannot re-enable the scoped index through a descendant wrapper; any inherited copy of that marker is removed before Codex starts. All other user policy selections remain intact.

After a zero process exit, the Codex boundary classifies only the exact validated structured response file; terminal streams, prose, missing or invalid files, and non-zero invocations cannot select a result. Before automatic remediation, the repository collaborator checks whether a checkpoint can safely be attributed to the fix stage. A dirty baseline continues remediation but skips checkpointing; a clean baseline may create a local checkpoint commit and never publishes it. After a clean classification, repository status or a run-local checkpoint determines whether finalization is applicable. Finalization keeps the exact verdict contract from the root README because that verdict controls workflow transitions and is the only publication path.

External process execution is a trust boundary. The runner preserves the operator's invocation directory, captures stdin/stdout/stderr, emits optional live source-labelled chunks only to the interactive presentation observer, propagates context cancellation, and never forwards raw Codex output to workflow stdout. Code-Converge does not add a timeout or override Codex sandbox, approval, or network configuration. Publication behavior remains hosting-provider-neutral.

When enabled, the app decorates only Codex-backed runner calls with `internal/session`: private local records capture redacted invocation diagnostics without becoming review result data or workflow events. The human renderer may publish one path-only session handoff after the directory exists; `kv` remains the stable event stream and receives no extra record.

Configuration has one resolver. The public option names, sources, and defaults are owned by [`../../README.md`](../../README.md); operational prerequisites are in [`../ops/config.md`](../ops/config.md). Initial delivery rationale is traceable through [`../features/FT-002/design.md`](../features/FT-002/design.md); human/structured presentation and liveness concurrency are traceable through [`../features/FT-009/design.md`](../features/FT-009/design.md).
