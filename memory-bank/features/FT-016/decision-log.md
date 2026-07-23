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

### Cycle 6 — shell-policy and provider-identity remediation

- **Routing:** Bug Fix flow. The root README already promises that the private review index is forced without narrowing the user's normal command environment, and FT-016 requires fail-safe provider discovery; the report identifies implementation that violates those accepted behaviors. The remediation remains in the existing FT-016 package. The validation profile is raised to `high-risk` because the shell-environment security boundary and provider trust decision are affected; the user's 2026-07-22 `fix findings` directive authorizes this repair.
- **Review scope:** review findings against `internal/codex/adapter.go` and `internal/repository/review.go`, plus the FT-016 design/architecture contract.
- **Important:** reviewer `P1`: overriding `shell_environment_policy.include_only` with `[*]` exposed variables deliberately excluded by the user. Reviewer `P2`: a provider identity discarded the host, so same-named repositories on different hosts were indistinguishable. Reviewer `P2`: only the first push URL was examined despite Git pushing to all configured `pushurl` values.
- **Changes:** retain only the private-index `shell_environment_policy.set` override; derive host-aware identities from every push URL; reject conflicting identities; scope `gh pr list` to the selected host/repository. Added deterministic allowlist, host-selection and conflicting-push-URL regressions.
- **Verification:** focused Codex/repository suites, affected app tests, all non-update packages and `go vet ./...` pass. `go test ./...` is blocked only because sandbox policy forbids the loopback listener used by `internal/update`; `make docs-lint` is blocked because sandbox DNS cannot download its pinned linter. `git diff --check`, required CI and independent final review remain closure evidence.
- **Human gate:** no; the user explicitly authorized the security remediation.

### Cycle 7 — nested-index and fork-target remediation

- **Routing:** Bug Fix flow within FT-016. The reported behavior violates the accepted private-index isolation and verified-provider-base contracts; the active `high-risk` profile remains applicable.
- **Review scope:** review findings against the forced Codex environment and open-PR discovery.
- **Important:** reviewer `P1`: the forced private index leaks into `git -C <other-repository>` commands and corrupts the review snapshot. Reviewer `P1`: querying only the fork push repository misses a PR owned by an upstream base repository.
- **Changes:** route Git through a per-review wrapper that clears `GIT_INDEX_FILE` outside the reviewed worktree, and query the verified head branch across same-host configured remotes before validating the returned PR head identity. Environment override merging now replaces inherited duplicate keys so the wrapper `PATH` takes effect.
- **Verification:** focused runner/Codex/repository/app suites and `go vet ./...` pass; `git diff --check` passes. `go test ./...` remains blocked only because sandbox policy prevents `internal/update` from binding `::1`; `make docs-lint` remains blocked because sandbox DNS cannot fetch its pinned linter. Required CI and independent final review remain closure evidence.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 8 — review-finding closure for fork, environment and Git-global-option paths

- **Routing:** Bug Fix flow within FT-016. The findings contradict accepted fork-base selection, optional-provider fallback and private-index isolation behavior; the active `high-risk` validation profile remains applicable.
- **Review scope:** reported regressions in `internal/repository/review.go` and `internal/codex/adapter.go`.
- **Important:** reviewer `P1`: after a fork PR is found through `upstream`, same-named `fork/main` and `upstream/main` refs remain ambiguous despite the provider SHA. Reviewer `P1`: a Codex `shell_environment_policy.set.PATH` is overwritten with the Code-Converge process PATH. Reviewer `P1`: `git -c … -C <other>` and explicit `--git-dir`/`--work-tree` commands retain the review index. Reviewer `P2`: valid local or unsupported-provider push URLs abort local fallback.
- **Changes:** carry the queried PR owner through base resolution and restrict matching remote refs to that owner; classify an all-non-provider push remote as unavailable provider discovery while still rejecting conflicting valid identities; retain only the private-index shell-policy override; parse Git global options before deciding whether a command remains in the reviewed worktree. Added end-to-end fork-base, non-provider fallback, PATH-policy and nested-repository regressions.
- **Verification:** all non-update packages, `go vet ./...` and `git diff --check` pass with a workspace-local Go cache. `go test ./...` remains blocked only because the sandbox forbids the loopback listener used by `internal/update`; `make docs-lint` remains blocked because sandbox DNS cannot download its pinned linter. Required CI and independent final review remain high-risk gates.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 9 — wrapper-policy and repository-creation remediation

- **Routing:** Bug Fix flow within FT-016. The findings violate the accepted private-index isolation invariant; the existing `high-risk` profile remains applicable.
- **Review scope:** the Codex shell-policy boundary and temporary Git wrapper in `internal/codex/adapter.go` and `internal/repository/review.go`.
- **Important:** reviewer `P1`: a configured or reconstructed Codex `PATH` can bypass the process-only wrapper while `GIT_INDEX_FILE` remains forced, allowing a nested Git command to alter the private snapshot. Reviewer `P2`: `clone` and worktree creation probe the current reviewed repository before their destination exists, so they retain the private index and can create a broken destination or overwrite the snapshot.
- **Changes:** force the wrapper-prefixed `PATH` through the same per-invocation Codex shell policy as the private index; the wrapper now clears the index before `clone` and `worktree add`, before any current-root probe. Added adapter policy regression coverage and real-Git clone/worktree snapshot-isolation coverage.
- **Verification:** focused Codex/repository/runner/app suites and `go vet ./...` pass with a writable temporary Go cache; `git diff --check` passes. `go test ./...` is blocked only because sandbox policy forbids the loopback listener used by `internal/update`; `make docs-lint` is blocked because sandbox DNS cannot download its pinned linter. Required CI and independent final review remain high-risk gates.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 10 — non-bypassable review-index environment remediation

- **Routing:** Bug Fix flow within FT-016. The reported absolute-path, noexec-temporary-mount and separated-namespace behavior contradicts the accepted private-index isolation invariant; the active `high-risk` profile remains applicable.
- **Review scope:** `internal/repository/review.go`, `cmd/code-converge/main.go` and `internal/codex/adapter.go`.
- **Important:** reviewer `P1`: globally exporting `GIT_INDEX_FILE` lets `/usr/bin/git` or `xcrun git` use the review snapshot outside the reviewed repository. Reviewer `P2`: a shell wrapper placed on a noexec temporary mount cannot run. Reviewer `P2`: `--namespace <name>` was not consumed before later global options or repository-creation detection.
- **Changes:** keep `GIT_INDEX_FILE` only in the snapshot operation and inside a scoped helper after it resolves the target repository. Codex receives a wrapper-prefixed `PATH` plus inert helper configuration, never `GIT_INDEX_FILE`; absolute Git paths therefore retain their normal index. The temporary `git` entry is a symlink to the running executable rather than a temporary script, and the helper parses both namespace forms. Added real-Git coverage for reviewed, nested and absolute Git commands plus namespace parser/creation cases.
- **Verification:** `go test ./internal/repository ./internal/codex`, `go vet ./...` and `git diff --check` pass with a writable temporary Go cache. `go test ./...` and `make docs-lint` remain required closure evidence; known sandbox limitations are recorded in `brief.md`.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 11 — descendant Git and provider-case remediation

- **Routing:** Bug Fix flow within FT-016. The findings violate `CTR-02` provider identity handling and the private-index isolation invariant; the active `high-risk` profile remains applicable.
- **Review scope:** descendant Git commands from the scoped wrapper and case-variant GitHub remote identities in `internal/repository/review.go`.
- **Important:** reviewer `P1`: a Git shell alias, hook or helper inherits `GIT_INDEX_FILE` and can bypass the wrapper after Git prepends its normal exec directory to `PATH`. Reviewer `P2`: host, owner and repository casing creates duplicate or conflicting identities for the same GitHub repository.
- **Changes:** make the wrapper directory the temporary `GIT_EXEC_PATH`, forwarding normal Git helpers while returning descendant `git` invocations to the repository-checking wrapper; clear that execution path alongside the private index for non-review targets. Canonicalize GitHub identity components before map-key deduplication. Add real-Git shell-alias isolation and case-variant push-URL regressions.
- **Verification:** focused repository suite, all non-update packages and `go vet ./...` pass with a writable temporary Go cache; `make docs-lint` and `git diff --check` pass. `go test ./...` remains blocked only because the sandbox forbids the loopback listener used by `internal/update`; required CI and independent final review remain high-risk gates.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 12 — restrictive-policy Git-exec and PATH remediation

- **Routing:** Bug Fix flow within FT-016. The findings violate the accepted private-index isolation invariant and the existing promise to preserve the user's normal Codex shell environment; the active `high-risk` profile remains applicable.
- **Review scope:** the forced Codex helper environment in `internal/codex/adapter.go`.
- **Important:** reviewer `P1`: a restrictive `shell_environment_policy` can omit inherited `GIT_EXEC_PATH`, allowing descendant Git commands to bypass the wrapper while they retain the private review index. Reviewer `P2`: forcing `shell_environment_policy.set.PATH` replaces a user-configured Codex policy PATH with Code-Converge's launch PATH.
- **Changes:** force and validate the temporary `GIT_EXEC_PATH` alongside scoped helper variables; leave `PATH` as the wrapper-prefixed launch environment rather than a Codex policy override, so an explicit Codex policy PATH remains intact. Added deterministic adapter coverage for forced `GIT_EXEC_PATH`, absence of a PATH policy override and a missing-exec-path failure.
- **Verification:** focused Codex/repository/runner/app suites and `go vet ./...` pass with a writable temporary Go cache; `git diff --check` passes. `go test ./...` remains blocked only because the sandbox forbids the loopback listener used by `internal/update`; `make docs-lint` remains blocked because sandbox DNS cannot download its pinned linter. Required CI and independent final review remain high-risk gates.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 13 — effective-PATH and restrictive-allowlist remediation

- **Routing:** Bug Fix flow within FT-016. The findings contradict the accepted complete-snapshot contract: a Codex policy `set.PATH` can bypass a launch-only wrapper, and an `include_only` policy can discard its environment-carried helper state. The active `high-risk` profile remains applicable.
- **Review scope:** scoped-index transport in `internal/codex/adapter.go` and `internal/repository/review.go`, plus the FT-016 design and root CLI contract.
- **Important:** reviewer `P1`: wrapper reachability must be forced through Codex's effective `PATH`, not just the Code-Converge launch environment. Reviewer `P1`: helper variables cannot survive a restrictive `include_only` policy without widening that user allowlist.
- **Changes:** force only the wrapper-first `PATH` through `shell_environment_policy.set`. Move the scoped root, private index, real Git executable and wrapper directory into a private sidecar beside the noexec-safe symlink wrapper; the wrapper injects `GIT_EXEC_PATH` only into the Git child process to retain alias/hook routing. No helper variable, `GIT_EXEC_PATH` or private index is exported through Codex policy. Added a real-Git `PATH`-only regression, equivalent to `include_only = ["PATH"]`, covering the reviewed repository, nested repositories, shell aliases, clones and worktree creation.
- **Verification:** focused Codex/repository/app/runner suites and `go vet ./...` pass with a writable temporary Go cache; `git diff --check` passes. `go test ./...` remains blocked only because the sandbox forbids the loopback listener used by `internal/update`; `make docs-lint` remains blocked because sandbox DNS cannot fetch its pinned linter. Required CI and independent final review remain high-risk gates.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 14 — complete PR-uniqueness and descendant-index remediation

- **Routing:** Bug Fix flow within FT-016. The reported failures contradict `INV-01` (one safely established base) and the private-index isolation invariant; the active `high-risk` profile remains applicable.
- **Review scope:** multi-target `gh` selection and the scoped Git child process in `internal/repository/review.go`.
- **Important:** reviewer `P1`: a matching PR from one provider target was accepted even though another target query failed, leaving uniqueness unproven. Reviewer `P1`: a shell alias, hook or helper can invoke an absolute Git path after inheriting the review index and overwrite the stable snapshot.
- **Changes:** reject any matching PR result if another provider target query failed; retain optional provider fallback only when no match exists. Every reviewed-root wrapper invocation now starts from a disposable command-local copy of the stable review index, so a descendant that bypasses `PATH` cannot modify the next review's snapshot. Added deterministic multi-target failure and real-Git absolute-alias regressions.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./internal/repository`, `go vet ./...`, `make docs-lint` and `git diff --check` pass. `go test ./...` is blocked only by the sandbox denying `internal/update` its IPv6 loopback listener; all other packages pass. Required CI remains to be recorded.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 15 — login-shell wrapper-bypass remediation

- **Routing:** Bug Fix flow within FT-016. The supported login-shell behavior contradicts `CTR-04`: profile initialization can reorder `PATH` after the Codex policy is applied and bypass the private-index wrapper. The active `high-risk` profile remains applicable.
- **Review scope:** `internal/codex/adapter.go` and the scoped Git transport contract.
- **Important:** reviewer `P1`: macOS login-shell initialization such as `path_helper` can prepend `/usr/bin` ahead of the forced wrapper path, causing Git to use the normal index and omit untracked snapshot content.
- **Changes:** force `allow_login_shell=false` for the review-only Codex invocation alongside its wrapper-first `PATH`. This keeps the wrapper active without re-exporting `GIT_INDEX_FILE`, preserving the already-established isolation of absolute and other-repository Git commands. Added deterministic argument coverage for the login-shell override while retaining the real-Git PATH-only snapshot regression.
- **Verification:** `codex review --strict-config -c 'allow_login_shell=false' --help`, `go test ./internal/codex ./internal/repository` and `git diff --check` pass with a writable temporary Go cache. `make docs-lint` is blocked because sandbox DNS cannot download its pinned linter; full `go test ./...` remains blocked only by the sandbox denying `internal/update` its IPv6 loopback listener. Required CI and independent final review remain high-risk gates.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 16 — Git-global-option private-index remediation

- **Routing:** Bug Fix flow within FT-016. Valid global Git flags that the scoped helper did not recognize could run against the real index, contradicting the accepted complete-snapshot and private-index isolation contracts. The active `high-risk` profile remains applicable.
- **Review scope:** global-option parsing and target resolution in `internal/repository/review.go`.
- **Important:** reviewer `P2`: `git -P diff --cached` and `git --no-lazy-fetch status` bypassed the private review index because unrecognized global flags were treated as commands targeting another repository.
- **Changes:** recognize Git's supported short pager and target-neutral global flags, including `-P`, `--no-lazy-fetch` and `--no-advice`; reject unsupported or malformed global options from the wrapper rather than falling back to the real index. Added real-Git PATH-only snapshot coverage and parser coverage for those flags.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./internal/repository` and `go vet ./...` pass. `go test ./...` is blocked only because sandbox policy denies `internal/update` an IPv6 loopback listener; `make docs-lint` is blocked because sandbox DNS cannot resolve `proxy.golang.org` to download its pinned linter. `git diff --check`, required CI and independent final review remain closure evidence.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 17 — absolute shell-alias repository-creation remediation

- **Routing:** Bug Fix flow within FT-016. An absolute Git executable in a shell alias could evade repository-creation detection and inherit a disposable review index, violating the private-index isolation invariant. The active `high-risk` profile remains applicable.
- **Review scope:** repository-creation classification in `internal/repository/review.go`.
- **Important:** reviewer `P2`: `alias.copy=!/usr/bin/git clone ...` was not recognized because detection accepted only the literal `git` executable name, producing a clone with an inherited disposable index.
- **Changes:** recognize absolute Git executable paths by basename, parse simple Git global options before classifying `clone` and `worktree add`, and fail closed for shell aliases containing shell syntax or otherwise not safely classifiable. Added a real-Git absolute-shell-alias clone regression and classifier coverage.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./internal/repository`, `GOCACHE=/private/tmp/code-converge-gocache go test ./...`, `GOCACHE=/private/tmp/code-converge-gocache go vet ./...`, `make docs-lint` and `git diff --check` pass. Independent review and required CI remain closure evidence.
- **Human gate:** no; this restores the established repository-creation safety contract.

### Cycle 18 — shell-alias target-isolation remediation

- **Routing:** Bug Fix flow within FT-016. A read-only absolute Git shell alias with `-C`, `--git-dir` or `--work-tree` could receive the review index before redirecting to another repository, violating the established isolation invariant. The active `high-risk` profile remains applicable.
- **Review scope:** shell-alias Git-target classification in `internal/repository/review.go`.
- **Important:** reviewer `P2`: `!/usr/bin/git -C ../other status` was treated as a safe read-only alias even though its descendant bypassed the wrapper and used the reviewed repository's disposable index in `../other`.
- **Changes:** shell aliases with repository-target-selection global options now fail closed and receive no review index. Added a real-Git regression proving an absolute alias targeting a nested repository returns its normal status and leaves the review index unchanged.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./internal/repository`, `GOCACHE=/private/tmp/code-converge-gocache go test ./...`, `GOCACHE=/private/tmp/code-converge-gocache go vet ./...`, `make docs-lint` and `git diff --check` pass. Independent review and required CI remain closure evidence.
- **Human gate:** no; this restores the accepted repository-isolation behavior.

### Cycle 19 — provider endpoint-identity remediation

- **Routing:** Bug Fix flow within FT-016. The findings violate `CTR-02`: a partially supported push destination set is not a uniquely verified provider repository, and a non-default endpoint port is part of its identity. The active `high-risk` profile remains applicable.
- **Review scope:** provider identity derivation and provider-query target selection in `internal/repository/review.go`.
- **Important:** reviewer `P2`: mixed provider and local/unsupported push URLs silently selected the provider URL; reviewer `P2`: URL parsing discarded a non-default SSH port and queried the default provider endpoint.
- **Changes:** provider discovery now falls back to local sources when its push URL set is mixed, and preserves non-default SSH/HTTP(S) ports in the canonical identity and `gh --repo` target. Added deterministic mixed-destination fallback and non-default-port query regressions.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./internal/repository`, `GOCACHE=/private/tmp/code-converge-gocache go test ./...`, `GOCACHE=/private/tmp/code-converge-gocache go vet ./...`, `make docs-lint` and `git diff --check` pass.
- **Human gate:** no; this restores the established verified-provider and local-fallback behavior.

### Cycle 20 — escaped-alias and paginated-provider-candidate remediation

- **Routing:** Bug Fix flow within FT-016. The findings contradict `INV-01` because uniqueness was decided from `gh`'s default partial result set, and violate the private-index isolation invariant because a valid backslash-escaped regular alias could hide `clone`. The active `high-risk` profile remains applicable.
- **Review scope:** regular Git alias parsing and open-PR candidate collection in `internal/repository/review.go`.
- **Important:** reviewer `P1`: Git executes `alias.clone-repository = cl\\one` as `clone`, while the wrapper treated `cl\\one` literally and could inject the disposable review index into a newly created repository. Reviewer `P2`: `gh pr list` returns only 30 results by default, allowing a later matching PR to be omitted from uniqueness evaluation.
- **Changes:** parse backslash escapes in regular aliases, including escaped whitespace and double-quoted escapes, before recursively classifying the resulting Git command. Query open PR candidates with a one-million `gh --limit`, which makes the CLI paginate beyond its default result page before uniqueness is evaluated. Added direct parser coverage, a real-Git escaped-clone isolation regression, and fake-provider assertions for the explicit limit.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./...` and `GOCACHE=/private/tmp/code-converge-gocache go vet ./...` pass; `git diff --check` passes after the documentation update. `make docs-lint` remains blocked because sandbox DNS cannot resolve `proxy.golang.org` to download its pinned linter. Required CI and independent final review remain closure evidence.
- **Human gate:** no; this restores established fail-closed alias classification and provider candidate pagination without changing the public contract.

### Cycle 21 — direct-helper and path-list isolation remediation

- **Routing:** Bug Fix flow within FT-016. The findings violate `CTR-04`: Git helpers exposed through the Codex-facing wrapper directory bypass private-index selection, and a temporary wrapper path containing a path-list separator can make the wrapper unreachable. The active `high-risk` profile remains applicable.
- **Review scope:** scoped Git transport in `internal/repository/review.go`.
- **Important:** reviewer `P1`: direct `git-*` commands resolved from the wrapper directory use the real index. Reviewer `P2`: a caller-controlled temporary directory containing the platform path-list separator is split when prepended to `PATH`.
- **Changes:** keep only the `git` symlink and its sidecar in the Codex-facing `PATH` directory; link Git helpers into a separate directory used solely as the wrapper child's `GIT_EXEC_PATH`. Reject temporary wrapper and helper paths containing the platform path-list separator before snapshot preparation. Added real-Git coverage for direct-helper separation and a deterministic separator rejection test.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./...`, `GOCACHE=/private/tmp/code-converge-gocache go vet ./...` and `git diff --check` pass. `make docs-lint` passed before the final evidence update, but its required retry is blocked because sandbox DNS cannot resolve `proxy.golang.org` for the pinned linter module. Required CI and independent final review remain closure evidence.
- **Human gate:** no; this restores the established scoped-review transport contract.

### Cycle 22 — configured-remote fallback and SCP-user remediation

- **Routing:** Bug Fix flow within FT-016. The findings violate `REQ-05` and the provider-identity contract: an expected missing configured remote must not block local base selection, and an SCP-style remote's username is not part of its provider host. The active `high-risk` profile remains applicable.
- **Review scope:** configured push-remote error handling and SCP-style provider identity parsing in `internal/repository/review.go`.
- **Important:** reviewer `P2`: a removed or URL-less `branch.<name>.remote` prevented the documented `gh-merge-base`/local fallback; `alice@github.example.com:owner/repo.git` was queried with `alice@` as part of the host.
- **Changes:** classify only Git's expected missing-remote/no-URL diagnostics as optional provider unavailability, preserving cancellation and unrelated command errors; remove any SCP username before canonicalizing the provider host. Added deterministic local-fallback, command-error propagation and arbitrary-username identity regressions.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./...`, `GOCACHE=/private/tmp/code-converge-gocache go vet ./...` and `git diff --check` pass. `make docs-lint` is blocked because the sandbox cannot resolve `proxy.golang.org` to fetch its pinned linter; required CI and independent final review remain closure evidence.
- **Human gate:** no; this restores the accepted optional-provider fallback and canonical provider-identity behavior.

### Cycle 23 — complete-hidden-snapshot and endpoint-normalization remediation

- **Routing:** Bug Fix flow within FT-016. The findings contradict `REQ-01`, `REQ-05` and `CTR-03`: valid tracked worktree edits must enter the private snapshot, and optional provider discovery must be stable across locale and valid endpoint syntax. The active `high-risk` profile remains applicable.
- **Review scope:** private-index staging and provider endpoint derivation in `internal/repository/review.go`.
- **Important:** reviewer `P1`: copied `skip-worktree` and `assume-unchanged` flags let sparse or hidden tracked edits evade ordinary `git add -A`. Reviewer `P2`: missing-remote recognition depended on localized stderr. Reviewer `P2`: default-port IPv6 provider hosts lost their required brackets.
- **Changes:** clear hidden flags only in the disposable index, retaining absent sparse paths while using sparse-aware staging; run the diagnostic-bearing push-URL lookup with `LC_ALL=C`; preserve IPv6 brackets for default and non-default ports in provider identities. Added real-Git sparse and assume-unchanged snapshot regressions plus deterministic locale and IPv6 normalization coverage.
- **Verification:** `GOCACHE=/private/tmp/code-converge-gocache go test ./internal/repository`, `GOCACHE=/private/tmp/code-converge-gocache go test ./...`, `GOCACHE=/private/tmp/code-converge-gocache go vet ./...`, `make docs-lint` and `git diff --check` pass. Required CI and independent final review remain closure evidence.
- **Human gate:** no; this restores the documented complete-scope and optional-provider behavior without changing the public contract.

### Cycle 24 — root-scoped snapshot and pinned-base remediation

- **Routing:** Bug Fix flow within FT-016. The findings contradict `REQ-01`, `REQ-02` and `REQ-05`: a complete local snapshot must work from a repository subdirectory, branch merge intent remains a local fallback when its name contains `/`, and emitted merge-base metadata must describe the pinned review base. The active `high-risk` profile remains applicable.
- **Review scope:** local base resolution and private-index snapshot preparation in `internal/repository/review.go`.
- **Important:** reviewer `P1`: a relative index path from a subdirectory was resolved against the repository root, and valid slash-containing `gh-merge-base` branch names were not checked against remote-tracking refs. Reviewer `P2`: merge-base was computed from a mutable symbolic ref after its commit had already been pinned.
- **Changes:** run root-dependent index and snapshot commands with `git -C <review-root>`; allow every unresolved branch candidate, including slash-containing names, through the unambiguous remote-tracking lookup; calculate `merge-base` from the pinned base commit. Added deterministic branch-merge-base and pinned-commit tests, a real-Git subdirectory snapshot regression, and aligned app fake-runner coverage for root-scoped commands.
- **Verification:** `GOCACHE=/private/tmp/code-converge-test.XXXXXX go test ./...`, `GOCACHE=/private/tmp/code-converge-vet.XXXXXX go vet ./...`, `make docs-lint`, `gofmt -d` for the modified Go files and `git diff --check` pass. Required CI and independent final review remain high-risk gates.
- **Human gate:** no; the user explicitly authorized the remediation.

### Cycle 25 — external-subcommand isolation remediation

- **Routing:** Bug Fix flow within FT-016. The review finding violates `CTR-04`: an external `git-*` command is neither a built-in nor a safely classifiable alias, so it must not receive the review index. The active `high-risk` profile remains applicable because the private-index security boundary is affected.
- **Review scope:** scoped Git subcommand classification in `internal/repository/review.go`.
- **Important:** reviewer `P2`: an unknown external command such as `git-foo` inherited the disposable review index; an absolute Git descendant could consequently operate on another repository or create one using the wrong index.
- **Changes:** classify an unresolved non-built-in, non-alias subcommand as unclassifiable and run it with the normal index. Added a real-wrapper regression that installs an external helper which invokes an absolute Git executable against a nested repository, verifies its normal status, and confirms that the review index remains unchanged.
- **Verification:** `GOCACHE=/private/tmp/code-converge-review-test.XXXXXX go test ./internal/repository`, `GOCACHE=/private/tmp/code-converge-full-test.XXXXXX go test ./...`, `GOCACHE=/private/tmp/code-converge-full-vet.XXXXXX go vet ./...`, `make docs-lint`, `gofmt -d` for modified Go files and `git diff --check` pass. Required CI and independent final review remain high-risk gates.
- **Human gate:** no; this restores the documented fail-closed private-index isolation behavior without changing the public contract.

### Cycle 26 — documented-global-option remediation

- **Routing:** Bug Fix flow within FT-016. The review finding contradicts `CTR-04`: a valid Git global option must not fail before the wrapped Git command runs. The active `high-risk` profile remains applicable because parser errors determine whether the review snapshot is available.
- **Review scope:** scoped Git global-option parsing in `internal/repository/review.go`.
- **Important:** reviewer `P2`: the allowlist rejected documented `--attr-source=<tree-ish>` and `--list-cmds=<group>` forms, turning valid Codex review-time Git commands into an exit-125 operational failure.
- **Changes:** accept both documented `--attr-source` forms and the `--list-cmds=<group>` probe while retaining unknown/malformed-option rejection. Added parser coverage and real PATH-only wrapper assertions for each accepted form.
- **Verification:** `GOCACHE=/private/tmp/code-converge-global-options.XXXXXX go test ./internal/repository`, `GOCACHE=/private/tmp/code-converge-options-test.XXXXXX go test ./...`, `GOCACHE=/private/tmp/code-converge-options-vet.XXXXXX go vet ./...`, `make docs-lint`, `gofmt -d` for modified Go files and `git diff --check` pass. Required CI and independent final review remain high-risk gates.
- **Human gate:** no; this restores supported Git invocation behavior without widening the parser to unknown options or changing the public contract.

### Cycle 27 — master reconciliation and independent-review remediation

- **Routing:** Bug Fix flow within FT-016. Reconciling PR #30 with the structured-result work already merged to `master` preserved both accepted contracts; the independent integration reviews then found valid provider, wrapper-parser and older-Git inputs that contradicted `CTR-02`–`CTR-04`. The active `high-risk` profile remains applicable.
- **Review scope:** PR #30 merged with current `master`, including provider identity parsing, scoped Git global-option parsing, private-index snapshot compatibility and FT-022's schema-constrained result channel.
- **Important:** first independent review reported two `P2` findings: userless SCP-style remotes were parsed as URI schemes before SCP syntax, and the documented `--exec-path=<path>` Git form was rejected. The second review confirmed those fixes and reported one `P1`: unconditional `git add --sparse -A` made every review fail on older supported Linux Git installations, even outside sparse checkout. The next review confirmed the compatibility fix and reported one `P1`: caller-exported `GIT_INDEX_FILE`, `GIT_EXEC_PATH` or repository selectors survived the PATH-only runner override and could redirect Codex or absolute Git commands. A fourth review confirmed that sanitization and reported two `P1` findings: non-login shell startup (`.zshenv`, `BASH_ENV`, or exported functions) could still replace the wrapper path, and an explicit or aliased split-index request could write shared-index state outside the disposable command index. A fifth review confirmed those remediations and reported one `P2`: Go-only escapes from `strconv.Quote` can make a valid Unix environment value invalid TOML in a Codex `-c` override. A sixth review confirmed the TOML fix and reported one `P2`: JSON replaces invalid UTF-8 bytes in Linux paths, which could change repository identity and fall back to the real index. A seventh review confirmed the path validation and reported one `P2`: `submodule add` creates a nested repository but inherited the disposable superproject index. The next re-review confirmed that fix and reported one `P2`: `submodule update --init` has the same nested-clone behavior. The following full review confirmed the nested-repository remediation and reported one `P2`: an uppercase `.GIT` URL suffix survived canonicalization and could reject an otherwise valid GitHub PR.
- **Changes:** try SCP syntax before URI parsing while excluding `://` transports; accept `--exec-path=<path>`; fall back to plain `git add -A` after a failed sparse-aware add only when `core.sparseCheckout` is confirmed disabled; and extend the runner/review-target contract with explicit environment removal so top-level review Git and Codex invocations sanitize inherited Git transports. Force a neutral shell/startup environment through Codex policy, remove inherited Git config injection and exported shell functions, reject reviewed-root commands or aliases that explicitly enable split-index, encode shell-policy values as TOML basic strings while rejecting non-UTF-8 values actionably, validate all JSON-sidecar paths as UTF-8 before wrapper creation, and withhold the review index from all potentially repository-creating submodule operations just like `clone` and `worktree add`. A child-only no-index marker preserves that classification across helper and alias descendants, while inherited copies are removed before review. Normalize the optional `.git` suffix after case folding the repository name. An active sparse checkout fails actionably when the installed Git lacks the required option. Added deterministic and real-shell regressions for all findings and retained the current FT-022 final-message-only classification path during conflict resolution.
- **Verification:** `go test ./...`, `go vet ./...`, `make docs-lint`, `git diff --check` and `go test -race ./internal/codex ./internal/repository ./internal/runner ./internal/app ./internal/workflow` pass after the complete submodule-initialization remediation. The focused real-Git scenario covers `submodule add`, deinitialization and `submodule update --init --recursive`, confirms the nested repository uses its own clean index, confirms the stable review snapshot is unchanged, and confirms an inherited no-index marker is removed before Codex. The provider regressions cover case-insensitive repository names and `.git` suffixes. Independent final re-review and required PR #30 CI remain closure evidence.
- **Human gate:** no; these changes restore already accepted provider, compatibility and fail-closed snapshot contracts.

### 2026-07-23 Original-delivery closure and evidence-convergence review

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
