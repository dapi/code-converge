---
title: Git Workflow
doc_kind: engineering
doc_function: convention
purpose: Confirmed code-converge Git branch and verification expectations, with unspecified policies made explicit.
derived_from:
  - ../dna/governance.md
  - testing-policy.md
  - ../../README.md
status: active
audience: humans_and_agents
---

# Git Workflow

## Default Branch

The project's current remote default branch is `master` as of the Memory Bank integration. A branch migration has not been proposed.

## Commits

- Use a concise human-readable subject that describes the intentional change.
- Conventional Commits, issue references, and auto-close keywords are not required by any current project policy.
- Squash, merge-commit, and rebase policy is not defined; do not claim one in automation.
- Do not combine unrelated user changes merely to produce a single commit.

## Hosted Change Requests

- Run applicable canonical checks from [`testing-policy.md`](testing-policy.md) before publication and record any unavailable check as a gap.
- Use a short subject that identifies the delivered outcome.
- Record what changed, verification evidence, and remaining risks or manual gaps in the hosted change request.
- When the target repository has applicable required CI, the `code-converge` workflow treats its green result as part of successful finalization. When it does not, the CI step is not applicable. Branch protection and required-check configuration belong to the target repository and hosting provider.

## Worktrees

No repository-specific worktree workflow or directory convention is defined. A task may use a worktree when explicitly requested, but must not assume bootstrap commands that do not exist.
