---
title: "FT-016: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Аудируемый журнал FPF reasoning, review-improve циклов и human gates для FT-016."
derived_from:
  - brief.md
  - https://github.com/dapi/code-converge/issues/16
status: active
audience: humans_and_agents
---

# FT-016: Decision Log

## Artifact Contract

- **Role:** provenance companion for FPF decisions, feature-document review and gate status.
- **Owns:** source facts, alternatives, reasoning, confidence, document conflicts and human gates.
- **Must not define:** requirements, public CLI/configuration contract, selected solution or execution sequence. Accepted facts move to `brief.md`, `design.md` or the root README.

## FPF Method

The reasoning bounds this feature to review-input selection, separates issue facts from public-contract choices, compares alternatives only against stated constraints, and treats a missing product choice as unknown. The gate prevents a configuration/event contract from being invented in a downstream design or plan.

## Decision Entries

### `DL-01` — Route and assurance floor

- **Status:** resolved; promoted to `brief.md`.
- **Question:** Which delivery flow and validation profile apply?
- **Facts:** Issue #16 changes executable review behavior, CLI/configuration and stdout observability; it is one independently verifiable delivery-unit. Routing excludes Small Change when CLI, configuration or event contracts change. The issue records no financial, security/auth, persistent-data, migration or concurrency trigger.
- **Alternatives:** Small Change/low-risk; Feature Flow/standard; Feature Flow/high-risk.
- **FPF reasoning:** Lifecycle route and validation depth are different bounded decisions. The externally observable Git/provider and review-input behavior needs Feature Flow plus standard regression/contract evidence. No available fact justifies high-risk.
- **Result:** `FT-016` follows Feature Flow with `standard` validation and no downgrade.
- **Confidence:** high; direct issue, routing and validation-policy facts.

### `DL-02` — Blocked public base/scope contract

- **Status:** open; escalated as `HG-01` and referenced by `DEC-01`.
- **Question:** What are the public names, exact candidate precedence, default scope/compatibility transition and terminal behavior for an already-merged branch?
- **Facts:** The issue requires an explicit CLI/configuration override, at least four discovery inputs, deterministic precedence/source reporting and fail-safe ambiguity behavior. It explicitly lists public names/precedence, direct default versus compatibility transition and already-merged behavior as design questions. The existing README only establishes the generic source precedence, not a review-base option, scope value, source vocabulary or compatibility policy.
- **Alternatives:** (A) select a new default `branch-and-worktree` and choose new public names/source values; (B) preserve `--uncommitted` as default and introduce the new scope through an opt-in/transition; (C) choose another explicitly documented compatibility policy. Each alternative still needs one exact candidate order and merged-branch terminal rule.
- **FPF reasoning:** The issue defines required capabilities, but not which external contract an existing user receives. Generic configuration-source precedence cannot derive the name, default value, compatibility promise or semantic candidate ranking. Selecting any alternative would create a requirement and alter CLI/event/documentation tests without evidence.
- **Result:** stop before `design.md` and `implementation-plan.md`; request a human choice. The brief records the common, evidence-backed problem/verify contract only.
- **Confidence:** high that the choice is unresolved; direct issue wording and current README absence.

### `DL-03` — FPF resolution of public base/scope contract

- **Status:** resolved; promoted to `brief.md`; `HG-01` closed.
- **Question:** Which smallest public contract satisfies the desired default, deterministic discovery and safe failure requirements without adding an unsupported compatibility surface?
- **Facts:** Issue #16 says the default must review the complete proposed change, requires an explicit CLI/configuration override and lists the discovery inputs in priority order: explicit override, unique open PR, branch-specific merge intent, default branch. It requires source reporting, local fallback without `gh`, no implicit fetch, no worktree/real-index mutation and explicit already-merged behavior. The existing README has one settings convention: flag, environment variable and project/user file share a hyphenated setting name; no current review-base or review-scope contract exists. Current review behavior is `--uncommitted` only.
- **Alternatives:** (A) default `branch-and-worktree`, with `--review-base`, `CODE_CONVERGE_REVIEW_BASE` and `review-base`; no scope selector; precedence explicit → unique open PR → `branch.<current>.gh-merge-base` → one unambiguous remote default ref. (B) retain worktree-only default and add opt-in branch scope. (C) expose both scope and base selectors with a compatibility matrix.
- **FPF reasoning:** In the abductive step, A is the smallest hypothesis that directly realizes the issue's default outcome. Deductively, B contradicts the requested default until a future transition and C introduces values, interactions and migration obligations absent from the issue. A preserves the established configuration naming and source model; its fallback order follows the issue's enumerated order. For the already-merged case, the merge-base-to-HEAD committed delta is empty, but worktree changes can still be proposed changes; failing would omit required staged/unstaged/untracked coverage. Therefore the selected scope continues with worktree content, retaining the existing clean/no-change workflow when none exists. No candidate permits implicit fetch or a guessed ambiguous ref.
- **Result:** ship `branch-and-worktree` as the only default scope; add `--review-base`, `CODE_CONVERGE_REVIEW_BASE` and `.code-converge/review-base`; select the base in order `explicit` → `open_pr` → `branch_merge_base` → `remote_default`; provider unavailability is non-fatal only when a later local source resolves; candidate ambiguity/missing/stale refs fail before Codex; an already-merged branch has an empty committed delta but still reviews worktree changes. `review_base_source` is one of those four stable values. No separate public scope setting or implicit fetch is introduced.
- **Confidence:** high for scope, source order and safety constraints because they follow the issue and existing configuration model; medium for the exact `review-base` spelling because it is a new but minimal convention-aligned public name authorized by the present FPF decision.

## Review-Improve Cycles

### Cycle 1 — bootstrap package and blocker review

- **Review scope:** issue #16, Task Routing, Feature Flow, validation/testing policy, root README configuration/event/review contract, domain/architecture owners, current app/config/Codex/repository/workflow boundaries, and all instantiated FT-016 documents.
- **Critical:** none.
- **Important:** `IMP-01` The feature lacks an evidence-backed public base/scope contract: the issue intentionally leaves names, precise precedence, compatibility/default and already-merged behavior open. Creating design or plan artifacts would make an unsupported selection.
- **FPF closures:** `DL-01` resolves the route/profile. `DL-02` establishes that `IMP-01` cannot be closed from available facts and must be escalated.
- **Changes:** created bootstrap `README.md`, canonical `brief.md` and this decision log; deferred downstream artifacts; updated the feature index.
- **Minor:** none changed.
- **Human gate:** yes, `HG-01`; the cycle stops here as required.

### Cycle 2 — FPF decision convergence

- **Review scope:** `HG-01`, issue #16 desired/default behavior, existing configuration naming/precedence, root README review semantics and all FT-016 artifacts.
- **Critical:** none.
- **Important:** `IMP-01` is resolved by `DL-03`; no other critical or important document inconsistency remains.
- **FPF closures:** `DL-03` selects the direct default, minimal public setting, candidate order, ambiguity rule and merged-branch behavior from stated facts and existing conventions.
- **Changes:** promoted the accepted decision to `brief.md` and unblocked `design.md` / `implementation-plan.md` creation.
- **Minor:** none changed.
- **Human gate:** no.

### Cycle 3 — implementation convergence review

- **Review scope:** complete FT-016 package, root CLI contract, configuration/app/Codex/repository/runner/workflow change set, focused tests and full local verification.
- **Critical:** none.
- **Important:** `IMP-02` PR-base discovery initially treated a unique `baseRefName` as a local branch only; ordinary clones may expose it only as a remote-tracking ref. Resolved by a deterministic unique remote-tracking normalization and a regression test; multiple matching remotes remain fail-closed.
- **FPF closures:** none required; `IMP-02` is a direct contract realization gap against `CTR-02` and `FM-02`.
- **Changes:** added the normalization, config output for discovery default, temporary-index review preparation, metadata events, full documentation and deterministic test coverage.
- **Verification:** `go test ./...`, `go vet ./...`, `make docs-lint`, `git diff --check` pass locally.
- **Publication evidence:** PR [#19](https://github.com/dapi/code-converge/pull/19) targets `master`; required [Verify run](https://github.com/dapi/code-converge/actions/runs/29947500863) passed.
- **Human gate:** no.

### Cycle 4 — external review remediation

- **Review scope:** PR #19 reviewer findings against `internal/repository/review.go` and `internal/workflow/workflow.go`.
- **Critical:** none.
- **Important:** reviewer `P1`: provider PR base could choose a stale local branch; reviewer `P2`: merge-base-initialized temporary index lost sparse-checkout committed paths; reviewer `P2`: a valid ref containing `=` broke event encoding.
- **Changes:** provider branch names now prefer a unique remote-tracking ref and fail on multiple; the temporary index begins as a private copy of the real index before `git add -A`; review event reports the resolved base SHA rather than a raw ref. Added deterministic provider/event tests and an actual sparse-checkout regression test.
- **Verification:** focused repository/workflow/app tests, then full local verification and CI are required before closure.
- **Human gate:** no.

### Cycle 5 — base identity review remediation

- **Review scope:** PR #19 reviewer findings about slash-containing PR target names, stale provider refs and mutable symbolic refs across review cycles.
- **Critical:** none.
- **Important:** reviewer `P1`: slash-containing PR base name was excluded from remote tracking lookup; reviewer `P1`: provider name alone could accept a stale local ref; reviewer `P2`: later review could use a moved symbolic ref while metadata remained pinned to the old commit.
- **Changes:** request and validate `baseRefOid` with `baseRefName`; all provider branch names search remote tracking refs first; mismatch fails with an actionable fetch diagnostic; Codex receives the resolved immutable base SHA. Added slash, stale-ref and pinned-invocation regression coverage.
- **Verification:** full local suites, documentation lint and required CI are required before closure.
- **Human gate:** no.

### 2026-07-23 Cycle 1 — closure and evidence-convergence review

- **Review scope:** issue #16, merged PR #19 and its required CI, every FT-016 artifact, the feature-package index, root README, and the directly dependent domain/architecture owners.
- **Critical:** none.
- **Important:** `IMP-03` `brief.md` still declared `delivery_status: planned` and `implementation-plan.md` remained active despite recorded passing checks and a merged PR; this violated the Feature Flow terminal-state contract. `IMP-04` `memory-bank/features/README.md` still described FT-016 as blocked at `HG-01`, although `DL-03`, the feature index and the design/plan all showed that the gate was closed. `IMP-05` required design-verification analyses said `planned` while their canonical `CHK-*`/`EVID-*` carriers were recorded as pass; `EVID-05` did not identify the final PR head's required Verify run.
- **FPF closures:** none. These are direct state and traceability corrections; no unresolved product, architecture or contract choice remained.
- **Changes:** marked the brief `done`; archived the plan; corrected the package index and feature index; changed design-verification results to their canonical evidence carriers; and linked `EVID-05` to the successful Verify run for PR #19's final head commit.
- **Human gate:** no.

### 2026-07-23 Cycle 2 — post-correction convergence review

- **Review scope:** corrected FT-016 package, issue #16, PR #19 status/required CI, root README, directly dependent domain/architecture owners, and `REQ` → solution → plan → `CHK`/`EVID` traceability.
- **Critical:** none.
- **Important:** none. The package now has one terminal lifecycle state, its archived plan and evidence state agree with the merged delivery, and the public/base-resolution contract agrees across all canonical owners.
- **FPF closures:** none; no blocking open question or material ambiguity was found.
- **Changes:** none; stopped review-improve early because no critical or important finding remained.
- **Human gate:** no.

## Human Gate

### `HG-01` — Public review-base/scope contract

- **Question:** Which public option/configuration names and default review scope should FT-016 ship, what exact candidate precedence should resolve the intended base, and what should happen when the branch is already merged into that base?
- **Available facts:** Issue #16 requires branch commits plus staged/unstaged/untracked coverage, explicit override, optional unique-open-PR discovery, branch merge intent including `branch.<current>.gh-merge-base`, default-branch fallback, source reporting and fail-safe ambiguity. Existing configuration precedence is CLI > project > user > environment > built-in default. Current behavior is only `codex review --uncommitted`; no existing base/scope public setting exists.
- **Options:**
  - **A — Direct default:** introduce named base/scope settings and make `branch-and-worktree` the default; preserve the existing four-source precedence for the explicit base override, then use PR → branch merge intent → default branch as discovery order unless you specify another order.
  - **B — Compatibility transition:** retain `--uncommitted` as the default and require opt-in to the new scope until a later approved default change; define whether the base override implicitly selects the new scope.
  - **C — Custom contract:** provide the exact names, default, candidate order and already-merged rule to implement.
- **Risk of a wrong choice:** a guessed default or precedence changes public CLI/configuration/event behavior, can unexpectedly expand review cost/scope or retain the reported defect, and makes documentation and regression evidence internally inconsistent.
- **Needed from a human:** choose A, B or C; for A/B, confirm the proposed candidate order and state whether an already-merged branch should fail before review or perform a documented empty/no-change review. Also provide preferred public names if the generic terms `review scope` and `review base` are not desired.

**Resolution:** closed by the user's instruction to use FPF for the decision, recorded as `DL-03`. The selected result is option A with the stated merged-branch behavior.
