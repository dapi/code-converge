---
title: "ADR-002: Deterministic delivery orchestration"
doc_kind: adr
doc_function: canonical
purpose: "Records the reusable ownership boundary between Code Converge and Codex for delivery lifecycle operations."
derived_from:
  - ../features/FT-039/brief.md
  - ../features/FT-039/design.md
status: active
decision_status: accepted
date: 2026-07-31
audience: humans_and_agents
must_not_define:
  - implementation_plan
---

# ADR-002: Deterministic delivery orchestration

## Context

Commit/push/PR/CI decisions were delegated to a Codex finalization session. That couples deterministic host operations to model-session duration and sandbox permissions, including linked-worktree Git metadata outside a model workspace.

## Decision

Code Converge owns deterministic repository and delivery lifecycle orchestration: repository inspection, safe checkpoint/commit decisions, remote/branch resolution, push, pull-request discovery/creation, and CI polling/classification. Codex owns review, code modification, and diagnosis/remediation of findings or failed CI.

Deterministic operations may run `git` and `gh` child processes, but Code Converge constructs, observes, retries and classifies them. The Finalize Codex stage and its configuration are removed.

## Consequences

Publication and CI lifetime are governed by the CLI deadline and cancellation context, not a model turn. Linked-worktree mutations occur in the host process. Users must remove obsolete finalize settings.

## Alternatives

- Increase Codex finalizer timeout: rejected; it does not solve sandbox ownership or deterministic classification.
- Grant Codex broad filesystem access: rejected; it needlessly widens model command authority.

## Related links

- [FT-039 brief](../features/FT-039/brief.md)
- [FT-039 design](../features/FT-039/design.md)
