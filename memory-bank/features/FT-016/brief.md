---
title: "FT-016: Review branch commits and working-tree changes against the intended PR base"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem, blocker and verify owner for extending review scope from the intended PR base through the current working tree."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../engineering/architecture.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/16
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-016: Review branch commits and working-tree changes against the intended PR base

## What

### Problem

The current review adapter invokes `codex review --uncommitted`. As documented and implemented, that covers staged, unstaged and untracked worktree files, but not commits already made on the current branch. Consequently, a clean worktree can be classified clean although the intended pull request contains branch commits. Hard-coding `origin/main` would not represent pull requests targeting another branch.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Committed-change coverage | A clean worktree invokes `--uncommitted` only | A selected, resolvable base yields one review of branch commits plus staged, unstaged and untracked worktree changes | Deterministic fake Git/Codex integration tests |
| `MET-02` | Base-selection safety | No base is selected or reported | Explicit and deterministic discovery selects a single base with source; ambiguity fails before Codex invocation | Configuration and discovery matrix |
| `MET-03` | Side-effect safety | Review does not need to change Git state | Discovery and review do not modify the worktree, real index, remote refs or hosting objects | Fake process assertions and negative tests |

### Scope

- `REQ-01` By default, review the complete proposed change from the merge-base with the intended pull-request base through the current working-tree state, including committed branch changes and staged, unstaged and untracked files exactly once.
- `REQ-02` Resolve one intended base deterministically, with an explicit override, unique open-PR discovery through an optional provider adapter, branch-specific merge intent including `branch.<current>.gh-merge-base`, and an unambiguous repository/remote default branch.
- `REQ-03` Apply one exact precedence order and expose the selected base, merge-base, review scope and discovery source without leaking raw agent output into the structured event stream.
- `REQ-04` Fail safely with an actionable diagnostic before review when no candidate is usable or candidates are ambiguous; define behavior for detached HEAD, unavailable provider/authentication, stale or missing base refs, forks/remotes ambiguity and an already-merged branch.
- `REQ-05` Preserve a usable local fallback without `gh` or provider authentication; provider discovery is optional and must not create or modify a pull request.
- `REQ-06` Do not modify the worktree or real Git index to include untracked files, and do not implicitly fetch remote refs unless the accepted public contract explicitly permits it.
- `REQ-07` Preserve fail-closed review classification and the bounded review/fix/finalize workflow outside the selected review-input scope.
- `REQ-08` Update the root README and dependent Memory Bank owners atomically with the implementation, and retain selected validation evidence.

### Non-Scope

- `NS-01` Creating, modifying, closing or merging a pull request; changing a remote repository or performing a network fetch as an undocumented side effect.
- `NS-02` Replacing Codex, Git, the existing finalization agent or the hosting-provider-neutral publication model.
- `NS-03` Changing finding classification, severity buckets, review/fix budgets, finalization verdict meanings or raw-output isolation.
- `NS-04` Inferring a PR target from an ambiguous set of remotes, open pull requests or stale refs without a documented deterministic rule or explicit user choice.

### Constraints / Assumptions

- `CON-01` Issue #16 requires one review cycle covering committed, staged, unstaged and untracked changes without duplicate findings; a native base review alone must not silently omit untracked files.
- `CON-02` The root README owns public flags, configuration names, precedence, event schema and exit semantics. Existing configuration resolution has one precedence model: CLI, project, user, environment, built-in default.
- `CON-03` The current architecture separates app/configuration, Codex invocation, repository status, workflow state policy, process running and progress presentation. The current Codex adapter invokes `codex review --uncommitted`.
- `CON-04` `gh` is neither a present product prerequisite nor a required provider: unavailable provider discovery must leave deterministic local resolution usable when Git configuration is sufficient.
- `CON-05` The issue expressly leaves public names/precedence, default-scope compatibility transition, combined-input mechanism, provider adapter and merged-branch behavior for an explicit design decision.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The feature changes CLI/configuration and stdout contracts; introduces Git/base-resolution and optional-provider boundaries; and needs explicit failure, compatibility and no-mutation semantics. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | FPF closure, conflict provenance and human gate must be auditable without defining canonical semantics. | feature-local provenance |
| `design.md` | selected | `DL-03` resolves the public contract; Git/provider and combined-input boundaries require a solution owner. | `design.md` |
| `implementation-plan.md` | selected | Execution needs coordinated configuration, workflow, Git/provider and documentation sequencing. | `implementation-plan.md` |
| Separate provider/Git contract | assess in design | Need depends on the selected combined-input approach. | `design.md` |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `high-risk` | The remediation changes the shell-environment security boundary and provider-identity trust decision. User directive of 2026-07-22 explicitly authorizes fixing the reported security and repository-selection findings; all affected deterministic security/failure paths, full local suites and independent review remain required. | user directive: `fix findings` |

## Blocking Decisions

| Decision ID | Question | Why it blocks | Owner / state |
| --- | --- | --- | --- |
| `DEC-01` | Which public option/configuration names, exact base-candidate precedence, default review scope and compatibility policy are accepted? | These facts determine the observable CLI/config contract, event source values, documentation and test matrix. | Resolved by `DL-03`: `--review-base` / `review-base`, direct `branch-and-worktree` default, explicit → open PR → branch merge intent → remote default; merged branch retains worktree review |

## Verify

### Exit Criteria

- `EC-01` A committed-only branch with a clean worktree is reviewed against the selected base rather than classified as a no-change review.
- `EC-02` A combined committed, staged, unstaged and untracked state is presented to exactly one review without omission or duplicate findings.
- `EC-03` Explicit override, unique open PR, branch merge intent and unambiguous default-base fallback follow the accepted precedence and report the selected source.
- `EC-04` Ambiguous/missing/stale candidates, detached HEAD and already-merged behavior follow the accepted fail-safe contract before Codex review; unavailable `gh` does not break sufficient local resolution.
- `EC-05` Git/hosting discovery and review preserve worktree, real index and remote state; no implicit network fetch or hosting-object mutation occurs.
- `EC-06` Root README, domain/architecture owners, feature design/plan and evidence agree on the delivered contract; required local and CI checks pass.

### Acceptance Scenarios

- `SC-01` Given a selected base and a clean worktree containing commits since its merge-base, the default review covers that branch delta.
- `SC-02` Given committed, staged, unstaged and untracked changes, one review covers the complete merge-base-to-worktree change exactly once.
- `SC-03` Given a unique open PR whose head branch and repository match the current branch's configured provider identity (or conventional `origin` fallback), discovery selects its non-default target even when the PR is owned by an upstream remote rather than the fork push remote; an unrelated fork using the same branch name fails safely, and before a PR exists an accepted branch merge-base setting wins over the accepted default-base fallback.
- `SC-04` Given an explicit override, it wins according to the established configuration precedence and `code-converge config` identifies its source.
- `SC-05` Given multiple conflicting PR/base candidates, a detached HEAD, missing/stale ref or an already-merged branch, the run follows the accepted diagnostic/terminal policy without guessing.
- `SC-06` Given unavailable `gh`/authentication and sufficient local Git configuration, review resolves locally and provider discovery creates or modifies no hosting object.
- `SC-07` Given any successful discovery, operational progress identifies scope, base, merge-base and source using only structured-safe values; raw Codex output remains isolated.

### Negative Coverage

- `NEG-01` No native base-only invocation is accepted as proof of complete scope if it omits untracked files.
- `NEG-02` No ambiguous remote/PR/base candidate is silently selected; closed or merged PRs are not current targets.
- `NEG-03` No discovery/review path modifies the real index/worktree, fetches implicitly, or creates/updates a PR.
- `NEG-04` A discovery failure never becomes a clean classification or a finalization attempt.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `EC-01`–`EC-05`, `SC-01`–`SC-06`, `NEG-01`–`NEG-04` | Deterministic table-driven Git/base-resolution and workflow tests with fake runner/adapter | Full candidate, error, merge-base and no-mutation matrix passes without real Git remote/provider mutation. | `artifacts/ft-016/verify/chk-01/` |
| `CHK-02` | `EC-02`, `EC-05`, `SC-02`, `NEG-01`, `NEG-03` | Fake-executable Codex/Git integration tests | Captured invocation/input covers combined scope once and leaves real Git state untouched. | `artifacts/ft-016/verify/chk-02/` |
| `CHK-03` | `EC-03`, `EC-04`, `EC-06`, `SC-03`–`SC-07` | Config/app/event golden tests and semantic contract read-through | Precedence/source/event diagnostics agree with the accepted public contract. | `artifacts/ft-016/verify/chk-03/` |
| `CHK-04` | `EC-06` | `make docs-lint` | README, dependent canonical owners and feature docs link and agree. | `artifacts/ft-016/verify/chk-04/` |
| `CHK-05` | `EC-06` | `go test ./... && go vet ./... && git diff --check` plus required CI | Full local verification and required CI are green. | `artifacts/ft-016/verify/chk-05/` |

### Evidence Contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | Git/base candidate and failure matrix | deterministic test runner | `artifacts/ft-016/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Fake-executable combined-scope/no-mutation report | deterministic test runner | `artifacts/ft-016/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Config/app/event golden-test and semantic review report | test runner/reviewer | `artifacts/ft-016/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Documentation lint and contract convergence record | test runner/reviewer | `artifacts/ft-016/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Full local verification and required-CI record | test runner/CI | `artifacts/ft-016/verify/chk-05/` | `CHK-05` |

### Execution Evidence Status

| Evidence ID | Status | Concrete carrier |
| --- | --- | --- |
| `EVID-01` | pass | `TestReviewScopeExplicitBaseBuildsPrivateSnapshot`, `TestReviewScopeMakesTemporaryDirectoryAbsolute`, `TestReviewScopeRelocatesTemporaryDirectoryInsideReviewedWorktree`, `TestRepositoryIdentity`, `TestReviewScopeFindsForkPullRequestFromUpstreamRepository`, `TestReviewScopeUsesUpstreamTrackingBaseForForkPullRequest`, `TestReviewScopeRejectsMultipleOpenPRs`, `TestReviewScopeRejectsPRFromUnrelatedForkWithSameBranchName`, `TestReviewScopeQueriesProviderAgainstPushRepositoryHost`, `TestReviewScopeRejectsConflictingPushRepositoryIdentities`, `TestReviewScopeFallsBackWhenCurrentProviderCannotBeEstablished`, `TestReviewScopeFallsBackFromNonProviderPushRemote`, `TestReviewScopeFallsBackFromMixedProviderAndNonProviderPushURLs`, `TestReviewScopeUsesOriginWhenBranchHasNoConfiguredPushRemote`, `TestReviewScopeUsesUniqueRemoteTrackingPRBase`, `TestReviewScopeUsesSlashContainingPRBase`, `TestReviewScopeRejectsStaleProviderBase`, `TestReviewScopePreservesCommittedChangesOutsideSparseCheckout`, `TestReviewScopeFallsBackForGitWithoutSparseAddOutsideSparseCheckout`, `TestReviewScopeRequiresSparseAddForSparseCheckout`, `TestReviewScopePrivateIndexDoesNotLeakOutsideReviewedRepository`, `TestReviewScopePrivateIndexSurvivesScopedShellEnvironment`, `TestReviewScopeRejectsTemporaryPathWithPathListSeparator`, `TestGitCreationAndTargetParsingConsumeNamespaceValue`, `TestSplitGitGlobalOptionsRecognizesSupportedTargetNeutralFlags`, `TestSplitGitGlobalOptionsRecognizesDocumentedValueFlags`; `go test ./internal/repository`, including split-index rejection, external-subcommand and submodule-descendant isolation, older-Git snapshot fallback and documented-global-option coverage. |
| `EVID-02` | pass | `TestAdapterInvocations`, `TestScopedReviewArgsDisableLoginShellToPreserveWrapperPath`, `TestScopedReviewArgsRequireScopedGitEnvironment`, `TestReviewScopePrivateIndexSurvivesScopedShellEnvironment`, `TestReviewScopeRemovesInheritedGitTransportEnvironment`, `TestLinkGitHelpersReservesScopedConfigurationFile`, app fake-runner integration, `TestExecPassesInvocationEnvironment`, `TestExecOverridesInheritedEnvironment` and `TestExecUnsetsInheritedEnvironment`; `go test ./internal/codex ./internal/app ./internal/repository ./internal/runner`, including non-login shell startup and inherited config/function sanitization. |
| `EVID-03` | pass | `TestReviewBasePrecedence`, `TestConfigCommand`, `TestReviewMetadataUsesResolvedCommitForEventSafety` and workflow/app event assertions; affected suite passed. |
| `EVID-04` | pass | Root README, domain rules/states, engineering architecture and FT-016 converge by semantic read-through; `make docs-lint` passes after the documented-global-option evidence update. |
| `EVID-05` | pass | Original delivery passed `go test ./...`, `go vet ./...`, `make docs-lint`, `git diff --check` and the required [PR #19 Verify run](https://github.com/dapi/code-converge/actions/runs/29949418336). The reconciled PR #30 head `cb210eede0c068db3ae58a680aaf84bc6f59cd45` passed the same local suite plus `go test -race ./internal/codex ./internal/repository ./internal/runner ./internal/app ./internal/workflow`, independent Codex review session `019f911a-1214-7843-a205-551329dbd977` reported no actionable correctness findings, and the required [PR #30 Verify run](https://github.com/dapi/code-converge/actions/runs/30050371885) passed. |
