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

Backout is a source revert; no remote data migration exists.
