---
title: "FT-022: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Аудируемый журнал FPF reasoning, routing conflict resolution, evidence boundaries and human gates for FT-022."
derived_from:
  - brief.md
  - https://github.com/dapi/code-converge/issues/22
status: active
audience: humans_and_agents
---

# FT-022: Decision Log

## Artifact Contract

- **Role:** provenance companion for FPF decisions, conflicts, confidence and gate status.
- **Owns:** source facts, alternatives, reasoning, conflict resolution and human-gate records.
- **Must not define:** canonical requirements, public CLI contract, selected solution or execution sequence. Accepted results are promoted to `brief.md`, `design.md` or `implementation-plan.md`.

## FPF Method

The log applies the canonical FPF propose → analyze → test reasoning cycle. Each material question is bounded to one owner context: problem/acceptance in `brief.md`, external-process solution semantics in `design.md`, and execution/approval sequencing in `implementation-plan.md`. A selected hypothesis records rival options and evidence-backed predictions; implementation tests later corroborate or refute those predictions. Confidence never exceeds the weakest source, so unsupported Codex version history or future schemas remain unknown.

## Decision Entries

### `DL-01` — Delivery route conflict

- **Status:** resolved; promoted to `brief.md` and package routing.
- **Question:** Which flow owns issue #22 when its issue text names Bug Fix Flow but the current user explicitly requests Feature Flow?
- **Facts:** Issue #22 describes an observed false failure and says Bug Fix Flow, while also requiring a feature package before implementation because invocation protocol/diagnostics change. Repository routing normally checks Bug Fix before Feature. The current user instruction explicitly says to take issue #22 through `feature-flow`; `AGENTS.md` independently requires a feature package for CLI/agent-contract changes. The delivery is one independently verifiable review-protocol unit.
- **Alternatives:** (A) follow Bug Fix Flow without a feature lifecycle; (B) follow Feature Flow and record the explicit routing override; (C) stop for a routing human gate.
- **FPF reasoning:** **Propose:** B is the only candidate satisfying the latest explicit instruction and the repository's required feature-package ownership. **Analyze:** A contradicts the requested process; C asks the user to repeat a choice already made. B keeps one delivery unit and makes the conflict auditable rather than pretending the issue and instruction agree. **Test prediction:** the package must satisfy every Feature Flow gate and must not claim the issue originally selected Feature Flow.
- **Result:** FT-022 follows Feature Flow. The conflict is explicitly resolved in favor of the current user instruction; no routing human gate remains.
- **Confidence:** high; direct issue, user and repository-governance facts.

### `DL-02` — Validation assurance floor

- **Status:** resolved; promoted to `brief.md` and `AG-01`.
- **Question:** Which validation profile is the minimum adequate assurance?
- **Facts:** The change replaces the command, result carrier and failure semantics of the existing local Codex process integration. `internal/runner`/architecture treat external process execution as a trust boundary. A false clean result can change workflow transition. Validation policy raises material cross-system protocol/failure-semantics changes to `high-risk`; no production deployment, persistent-data, finance or migration trigger exists.
- **Alternatives:** `standard`; `high-risk`; `release-deployment`.
- **FPF reasoning:** **Propose:** high-risk fits the material integration/trust-boundary trigger. **Analyze:** standard under-covers explicit approval, critical failure paths and independent convergence; release-deployment misclassifies a local binary protocol as deployment/config work. **Test prediction:** the plan must contain approval, complete failure/channel coverage, backout, independent review and full evidence.
- **Result:** `high-risk`, no downgrade. `AG-01` blocks code execution until a human explicitly approves the transition from Plan Ready.
- **Confidence:** high; direct validation-policy and architecture facts.

### `DL-03` — Structured final-response protocol

- **Status:** resolved; promoted as `SOL-01`–`SOL-05`, `CTR-01`–`CTR-05`.
- **Question:** Which Codex surface removes the terminal-channel/prose dependency while preserving the selected review target?
- **Facts:** Issue #22 proposes `codex exec --output-schema ... --output-last-message ...`. Local Codex CLI 0.145.0 exposes both flags on `codex exec`, while `codex review` exposes neither. The current official Codex manual states that `codex exec` sends progress to stderr, supports JSON Schema-constrained final responses and can write the last message to a file. `ReviewScope` already supplies the pinned base and private-index environment to any runner invocation.
- **Alternatives:** keep `codex review`; merge terminal streams; use `codex exec --json`; use `codex exec` with schema plus final-message file.
- **FPF reasoning:** **Propose:** schema plus final-message file directly explains and removes both observed failure causes. **Analyze:** the other candidates retain prose/channels or introduce the larger JSONL lifecycle; the selected candidate composes with the existing runner/env boundary. **Test predictions:** args include both flags and `exec`; stdin names base/private index; result stays stable when stdout/stderr vary; missing file fails.
- **Result:** use one synchronous `codex exec` with strict schema, final-message file, stdin review instruction and unchanged `ReviewTarget.Env`.
- **Confidence:** high for Codex CLI 0.145.0 and the issue direction; capability compatibility for other versions remains intentionally bounded by `DL-05`.

### `DL-04` — Sole classification carrier and parser boundary

- **Status:** resolved; promoted as `SD-01`, `SD-02`, `SD-04`, `INV-01`–`INV-03`.
- **Question:** May the adapter retain plain-text fallback or inspect terminal streams when the final-response file is absent or invalid?
- **Facts:** Issue #22 says read only `--output-last-message`, exclude stderr from review data and never treat arbitrary prose as clean. The existing `ParseReview` accepts an allowlisted plain-text clean verdict, while its structured path strictly validates FT-015 JSON. A zero/non-zero runner outcome is known before file parsing.
- **Alternatives:** (A) call `ParseReview` on the file and keep prose fallback; (B) inspect stdout/stderr if the file fails; (C) call only the strict structured parser after zero exit.
- **FPF reasoning:** **Propose:** C is the smallest boundary that can make the issue's safety claims true. **Analyze:** A can accept schema-invalid prose; B recreates the channel dependency; C predicts deterministic results and preserves fail-closed behavior. **Test predictions:** allowlisted prose in the file fails; valid JSON on stdout/stderr cannot rescue a missing/invalid file; valid file after non-zero exit is ignored.
- **Result:** the final-message file is the sole result carrier and uses only strict structured validation after a zero process exit.
- **Confidence:** high; direct issue acceptance plus existing parser behavior.

### `DL-05` — Codex compatibility statement

- **Status:** resolved; promoted as `SD-05`, `FM-05`, `NS-06`.
- **Question:** Should FT-022 publish an exact minimum Codex version or keep legacy fallback for versions without the required flags?
- **Facts:** Local evidence proves the flags exist in Codex CLI 0.145.0, but the issue/current package does not establish the first supporting release. Issue #22 requires non-zero Codex invocation to exit `2` and explicitly rejects terminal parsing as result data.
- **Alternatives:** invent a minimum semantic version; fall back to `codex review`; document required capabilities and let unsupported invocation fail closed.
- **FPF reasoning:** **Propose:** capability-based compatibility is the only evidence-bounded candidate. **Analyze:** an exact version would invent release history; fallback violates `DL-04`; capability failure is already covered by required non-zero semantics. **Test prediction:** an unsupported flag produces the ordinary runner diagnostic and exit `2`, with no fallback/result counters.
- **Result:** document capabilities rather than an unsupported version floor; incompatible versions fail closed.
- **Confidence:** high about the evidence boundary; intentionally no claim about earlier Codex releases.

### `DL-06` — Missing review scope

- **Status:** resolved during review-improve Cycle 1; promoted as `SD-06`, `INV-06`, `FM-07`.
- **Question:** What should `Adapter.Review` do when `ReviewScope == nil` after the single structured protocol is introduced?
- **Facts:** Current `Adapter.Review` has a nil-scope branch that runs `codex review --uncommitted`, primarily exercised by adapter unit tests. Production `App.Run` always constructs and supplies `repository.ReviewScope`. Issue #22 requires the selected base and private `GIT_INDEX_FILE` snapshot to remain available, and `DL-03`/`DL-04` select one `codex exec` final-response protocol.
- **Alternatives:** (A) retain the legacy `codex review --uncommitted` branch; (B) invent an unscoped `codex exec` prompt/target; (C) fail before invocation when scope is missing.
- **FPF reasoning:** **Propose:** C preserves one protocol and makes the missing dependency explicit. **Analyze:** A violates sole-protocol/channel decisions; B cannot satisfy the pinned-base/private-index requirement without inventing a second target contract. C matches production wiring and turns a test convenience into a deterministic configuration failure. **Test predictions:** nil scope produces no runner call; every successful review has `ReviewTarget` metadata and the prepared target environment.
- **Result:** require `ReviewScope`; nil is a contextual adapter configuration error with no Codex invocation.
- **Confidence:** high; direct source/wiring and accepted feature-contract facts.

### `DL-07` — Private index across Codex shell-environment policy

- **Status:** resolved during review-improve Cycle 2; promoted as `SD-07`, `FM-08`, `CON-05`.
- **Question:** Is passing `ReviewTarget.Env` to the Codex process sufficient to keep `GIT_INDEX_FILE` available to reviewing-agent tool commands?
- **Facts:** `internal/runner` passes `ReviewTarget.Env` to the Codex process. The current official Codex manual states that `shell_environment_policy` controls variables forwarded to spawned commands, may use `include_only`/`exclude`, and that `set` values always win. Issue #22 and `REQ-04` require the private-index snapshot to remain available to the reviewing agent. The index path is local non-secret target metadata already present in `ReviewTarget.Env`.
- **Alternatives:** (A) rely on default environment inheritance; (B) mention only the path in the prompt; (C) preserve process env and add an invocation-local `shell_environment_policy.set.GIT_INDEX_FILE` override matching the prompt/path.
- **FPF reasoning:** **Propose:** C is the only candidate robust to documented user filters without changing persistent configuration. **Analyze:** A fails under `include_only`; B relies on model behavior and can still leave ordinary Git tool calls pointed at the real index. C uses the supported override precedence and changes exactly one non-secret variable for this invocation. **Test predictions:** process env, config override and prompt contain the identical TOML-safe path; missing or duplicate target entries fail before Codex starts.
- **Result:** bind the exact private-index path into both the process environment and Codex spawned-tool policy; validate a single non-empty target entry.
- **Confidence:** high; direct runner, issue and current Codex manual facts.

### `DL-08` — Selected base versus merge-base comparison

- **Status:** resolved during review-improve Cycle 3; promoted as `SD-08`, `FM-09` and the refined `REQ-04`.
- **Question:** Should the reviewing agent diff the private index from `ReviewTarget.BaseCommit` or `ReviewTarget.MergeBase`?
- **Facts:** The root README defines the current review scope as a private merge-base-to-worktree snapshot. `ReviewScope.Prepare` computes and returns both the pinned selected-base commit and its merge base with `HEAD`. Current `codex review --base` preserves branch-diff semantics; a direct diff from a diverged base tip can include target-branch-only changes.
- **Alternatives:** (A) `git diff --cached <BaseCommit>`; (B) `git diff --cached <MergeBase>` while retaining `BaseCommit` as provenance; (C) name only a base ref and let the agent infer the comparison.
- **FPF reasoning:** **Propose:** B preserves the existing accepted scope. **Analyze:** A changes the reviewed set on diverged branches; C is not deterministic and discards already computed target data. B predicts the same merge-base-to-private-snapshot delta while keeping selected-base provenance visible. **Test predictions:** fixtures with distinct base/merge-base values put the merge base in the diff instruction and keep both values in result metadata; missing either value fails before Codex starts.
- **Result:** compare the private index from `ReviewTarget.MergeBase`; expose `BaseCommit` as pinned selected-base provenance.
- **Confidence:** high; direct public contract and repository implementation facts.

### `DL-09` — Execution and publication approval

- **Status:** resolved; satisfies `AG-01` and `AG-02`.
- **Question:** May FT-022 leave Plan Ready, modify the repository, publish a branch/PR and complete the required CI/review loop?
- **Facts:** The feature selected the `high-risk` profile and explicitly required `AG-01` before implementation and `AG-02` before remote publication. In the 2026-07-23 delivery turn, the user instructed the agent to implement the task end-to-end, verify it locally, commit and push the changes, create or update the PR, and continue until required CI and critical/high review findings are clear.
- **Alternatives:** (A) remain Plan Ready; (B) execute locally but stop before publication; (C) perform the explicitly requested full delivery flow.
- **FPF reasoning:** **Propose:** C matches the direct authorization and the existing bounded plan. **Analyze:** A and B would contradict respectively the implementation and publication portions of the current instruction. The authorization does not widen feature scope or permit production/live-data actions. **Test prediction:** `brief.md` moves to `in_progress`; implementation stays within `REQ-01`–`REQ-08`; publication targets the current feature branch and `master`; closure still requires green local/CI evidence and no critical/high finding.
- **Result:** `AG-01` and `AG-02` are approved for the scoped FT-022 delivery.
- **Confidence:** high; direct current-turn user instruction.

## Execution Review And Verification

### Iteration 1 — functional and simplify review

- **Critical/high:** none.
- **Important closed:** the legacy plain-text `ParseReview` path became unreachable after the live adapter switched to the strict final-response parser; it was removed with its allowlist fixtures so no alternate clean-classification semantics remain. The review schema also gained an explicit nested strictness/priority assertion.
- **Verification:** focused adapter/repository/workflow/app suites, full `go test ./...`, `go vet ./...`, `go test -race ./internal/codex ./internal/app`, `make docs-lint` and `git diff --check` pass.

### Iteration 2 — independent Codex convergence

- **Critical/high:** none.
- **Findings:** none. Independent review session `019f8dd7-3ff5-7922-b328-059b6a6e3120` returned `findings: []`, `overall_correctness: patch is correct`, and confidence `0.87`.
- **Changes:** none required after the independent review.
- **Hosted evidence:** pending PR creation, mergeability and required CI at this iteration.

### Iteration 3 — publication and hosted convergence

- **Critical/high:** none.
- **Review signals:** PR [#23](https://github.com/dapi/code-converge/pull/23) has no review or comment findings.
- **Mergeability:** GitHub reports `MERGEABLE`.
- **Verification:** implementation commit `eb5dd83` passed required [Verify run](https://github.com/dapi/code-converge/actions/runs/29988067825); local full/race/docs/diff checks and the independent clean review remain valid.
- **Changes:** delivery evidence is closed in `brief.md`, the feature index is marked complete and the implementation plan is archived. A docs-only evidence commit and its required CI remain the final publication check.

## Human Gate

No human gate is open. The feature-document review completed without a gate, and the current end-to-end delivery instruction satisfies planned execution/publication gates `AG-01` and `AG-02`.
