---
title: "FT-039: Implementation plan"
doc_kind: feature
doc_function: derived
purpose: "Execution sequence and verification for deterministic publication and CI orchestration."
derived_from:
  - brief.md
  - design.md
status: active
audience: humans_and_agents
---

# FT-039: Implementation plan

| Step | Scope | Evidence / stop condition |
| --- | --- | --- |
| `STEP-01` | Accept ADR-002, remove Finalize settings/contract, add `ci-timeout` resolution and config/help migration. | Precedence and removal tests; stop if any obsolete setting remains operational. |
| `STEP-02` | Implement host-owned commit eligibility, direct-ref push, provider-bound PR reuse/create, and exact-SHA check-run polling. | Fake runner covers dirty baseline, detached/ambiguous identity, direct-ref push, PR ambiguity, URL parsing, no checks, green, first failure, stale SHA, transient/permanent provider failure, deadline, and cancellation. |
| `STEP-03` | Replace the workflow state/event transition with `publish` then `ci`; retain the bounded Fix-CI → Review loop. | Workflow tests cover success, skipped CI, failed CI recovery, exhausted recovery, timeout exit `2` without Fix CI, and exit `130`. |
| `STEP-04` | Verify a linked worktree publication path from the host process and document output/rollout/backout. | A linked-worktree test proves no Codex workspace write is needed for shared Git metadata; root README and canonical owners agree. |
| `STEP-05` | Run validation profile `standard`. | `go test ./...`, `go vet ./...`, `make docs-lint`, and `git diff --check`; record any environment-unavailable command as a gap. |
