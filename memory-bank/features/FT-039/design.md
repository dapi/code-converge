---
title: "FT-039: Design"
doc_kind: feature
doc_function: canonical
purpose: "Selected deterministic publication and CI orchestration design for GH-39."
derived_from:
  - brief.md
  - ../../engineering/architecture.md
  - ../../adr/ADR-002-deterministic-delivery-orchestration.md
status: active
audience: humans_and_agents
must_not_define:
  - ft_039_scope
  - ft_039_acceptance_criteria
  - implementation_sequence
---

# FT-039: Design

## Design pack

| Artifact | Role | Owns |
| --- | --- | --- |
| `design.md` | Feature solution | `SOL-*`, contracts, invariants and failure modes |
| [ADR-002](../../adr/ADR-002-deterministic-delivery-orchestration.md) | Reusable architectural rule | Ownership boundary |

## C4 applicability

`C4-00`: not required. Existing CLI, workflow, repository and runner components retain their boundaries; GitHub CLI is an existing child-process connector.

## Selected solution

- `SOL-01`: Replace `Agent.Finalize` with `Repository.Publish(ctx)`. It commits only when status is clean, detects no-op commits, resolves current branch/remote, then uses `gh` to reuse/create one open PR.
- `SOL-02`: `Repository.WaitCI(ctx, publishedSHA, timeout)` polls provider data for the exact head revision. It retries transient failures within the deadline, returns failure immediately, green only when all checks are terminal accepted states, skipped when there are no checks, and timeout otherwise.
- `SOL-03`: Workflow emits repository-owned publication and CI outcomes. `failed` starts Fix CI; `timeout` is operational.
- `SOL-04`: Remove finalization model/effort/prompt CLI/env/file/profile/config settings. This is a breaking removal, not a deprecated no-op.

## Architecture coverage

| Aspect | Status | Notes |
| --- | --- | --- |
| Components | covered | workflow selects transitions; repository executes Git/GitHub commands; event renders output. |
| Connectors | covered | synchronous `git` and `gh` child processes; JSON is parsed/classified locally. |
| Configuration | covered | `ci-timeout` follows the common resolver. |
| Behavioral semantics | covered | contracts, invariants and failure modes below. |
| Quality/evolution | covered | deadline, retry, cancellation and explicit breaking migration. |

## Contracts, invariants and failures

- `CTR-01`: Publication returns commit, push, PR and head SHA. A remote head observed after a push is success even if local tracking-ref refresh reports an error.
- `CTR-02`: CI only classifies checks for the exact published SHA; stale data is retried until deadline.
- `INV-01`: A dirty worktree is never automatically committed at publication.
- `INV-02`: Success requires every applicable check to be terminal `success|skipped|neutral`.
- `INV-03`: The first applicable failure enters Fix CI; timeout never does.
- `INV-04`: Context cancellation reaches active child processes and yields exit 130.
- `FM-01`: Ambiguous remote, branch or PR identity; provider auth/protocol error → operational failure.
- `FM-02`: CI deadline → `ci=timeout`, operational exit 2.
- `FM-03`: No applicable checks → `ci=skipped`.

## Public and agent-contract compatibility

- `--ci-timeout`, `CODE_CONVERGE_CI_TIMEOUT`, and `.code-converge/ci-timeout` resolve under the established CLI > project > user > environment > default precedence. The built-in default is `60m`; values below one second are configuration errors.
- `finalize-model`, `finalize-reasoning-effort`, `finalize-prompt-file`, their environment variables, profile entries, configuration files, help entries, and `config` output are removed. This is an intentional breaking migration: obsolete environment and configuration-file settings fail configuration with an actionable removal diagnostic; they are never silently ignored.
- Codex receives only Review, Fix findings, and Fix CI prompts. There is no finalization prompt, output schema, verdict, or Codex process after a clean review.
- The public state/event replacement is `publish` followed by `ci`. Publication emits deterministic `commit`, `push`, and `change_request` step outcomes. CI emits `success`, `skipped`, `failed`, or `timeout`; `timeout` produces `run_completed status=ci_timeout exit_code=2`.

## Provider connector and temporal semantics

- Publication resolves the current branch and one push remote; detached HEAD, missing or ambiguous remotes, malformed PR data, and multiple matching open PRs are operational errors. It uses `git push <remote> HEAD:refs/heads/<branch>` so local tracking-ref maintenance is not used as proof of remote publication.
- PR discovery, creation, and CI queries are bound to the resolved push remote's uniquely parsed GitHub `owner/repository` identity. A pre-existing matching open PR is reused; exactly one newly created PR URL is accepted.
- The CI deadline starts after the published SHA is known. Every poll collects every page from GitHub's check-runs endpoint for that SHA, never a branch or latest workflow run. Empty check-runs are `skipped`; only completed `success`, `skipped`, and `neutral` conclusions are accepted.
- Retryable transport/provider failures back off within the same deadline. Authentication, authorization, identity, malformed-protocol, and malformed-data failures are operational immediately. The first completed unaccepted check conclusion returns `failed` without waiting for other checks.
- The workflow's cancellation context reaches every `git` and `gh` child process. Cancellation wins over classification and produces exit `130`; deadline expiry is distinct from cancellation and CI failure.

## Rollout and backout

This is a breaking configuration migration. Release notes and root help direct operators to delete obsolete Finalize settings before upgrading. No remote data migration is required. Backout is a source revert; an already-published branch or PR is not deleted automatically.
