---
title: "FT-009: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Аудируемый журнал FPF reasoning, conflict resolution, review-improve циклов и human gates для FT-009."
derived_from:
  - brief.md
  - design.md
  - https://github.com/dapi/code-converge/issues/9
status: active
audience: humans_and_agents
---

# FT-009: Decision Log

## Artifact Contract

- **Role:** provenance companion for FPF decisions, document review and gate status.
- **Owns:** source facts, alternatives, reasoning, confidence, conflict resolution, open questions and cycle findings.
- **Must not define:** requirements, public contract, selected solution or execution sequence. Accepted facts are promoted to `brief.md`, `design.md` or the root README as appropriate.

## FPF Method

Each material question uses the canonical reasoning cycle: frame the bounded claim, enumerate alternatives, deduce consequences against explicit constraints, compare them with available repository/issue evidence, then either promote the selected fact or abstain through a human gate. Confidence is bounded by the weakest evidence link; absence of implementation evidence is never reported as design verification.

## Decision Entries

### `DL-01` — Route and validation profile

- **Status:** resolved; promoted to `brief.md`.
- **Question:** Which delivery flow and assurance floor apply?
- **Facts:** Issue #9 changes CLI behavior, configuration and the public stdout schema, explicitly requests Feature Flow, and adds liveness with completion/cancellation/write races. Routing prohibits Small Change for CLI/config/stdout contracts. Validation policy raises concurrency semantics to `high-risk`.
- **Alternatives:** Small Change/low-risk; Feature Flow/standard; Feature Flow/high-risk.
- **FPF reasoning:** Separate lifecycle shape from assurance depth. This is one independently verifiable delivery-unit, so Feature Flow fits. The weakest assurance link is concurrent output ordering, making `standard` insufficient even though compatibility is preserved.
- **Result:** Feature Flow package `FT-009`, validation profile `high-risk`, no downgrade.
- **Confidence:** high; direct governance and issue evidence.

### `DL-02` — Semantic format default and compatibility

- **Status:** resolved; promoted as `SOL-01`, `SD-01`, `CTR-05`.
- **Question:** Is human output default or opt-in?
- **Facts:** The root README calls the current `key=value` stream stable; issue #9 requires preserving it for automation/compatibility and forbids implicit semantic selection by TTY. The issue leaves the default open.
- **Alternatives:** human default; kv default; TTY-selected default.
- **FPF reasoning:** `kv` default preserves all known consumers and satisfies deterministic selection. Human default has an unbounded compatibility downside with no evidence that consumers have migrated. TTY selection violates a stated constraint.
- **Result:** `kv` remains the built-in default; `human` is explicitly selectable.
- **Confidence:** high; conservative weakest-link choice from the published compatibility contract.

### `DL-03` — New setting names and precedence

- **Status:** resolved; promoted as `SOL-02`, `CTR-01`.
- **Question:** What exact flag/config/environment names and validation rules fit the existing configuration system?
- **Facts:** Existing settings use kebab-case flags/files, `CODE_CONVERGE_*` environment names and one documented CLI → project → user → environment → default precedence. Issue #9 provisionally names log formats and `--heartbeat=30s`.
- **Alternatives:** reuse the resolver with `log-format`, `heartbeat`, `color`; add flags only; invent a new output-specific precedence layer.
- **FPF reasoning:** Reusing the established binding is the only alternative congruent with the current config boundary and `code-converge config`. A separate precedence layer would create split ownership without evidence.
- **Result:** exact contract in `CTR-01`; invalid values fail before Codex, heartbeat minimum is `1s`, and heartbeat with `kv` is rejected.
- **Confidence:** high for names/precedence; medium-high for the `1s` lower bound, selected as the least restrictive bound that still prevents a busy loop/excessive log rate.

### `DL-04` — Human timestamps and structured heartbeat

- **Status:** resolved; promoted as `SD-02`, `SD-03`, `NS-04`.
- **Question:** Are human timestamps or structured heartbeats part of this delivery?
- **Facts:** Issue #9 says human timestamps are omitted by default and only asks to consider a separate option. It says structured heartbeat is conditional, must be explicitly enabled if supported, while acceptance specifically requires a heartbeat for redirected human output. Existing `kv` compatibility is mandatory.
- **Alternatives:** add both; add human timestamps only; support human heartbeat only; add neither.
- **FPF reasoning:** The acceptance-bearing capability is human heartbeat. Adding either human timestamps or a new structured event expands contract/test surface without supporting an unmet acceptance criterion and weakens structured compatibility.
- **Result:** human timestamps and structured heartbeat are out of scope; `kv` retains its timestamps; heartbeat is human-only.
- **Confidence:** high; explicit issue language and scope minimization.

### `DL-05` — Transient stream and liveness ordering

- **Status:** resolved; promoted as `SOL-04`–`SOL-06`, `CTR-04`.
- **Question:** Which stream owns transient output, and how is it coordinated with permanent records and diagnostics?
- **Facts:** The root README owns workflow progress on stdout and diagnostics on stderr. Issue #9 requires no redirected corruption, clearing before permanent/diagnostic lines, synchronization, and prompt stop on completion/failure/cancellation.
- **Alternatives:** transient stderr; transient stdout with independent locks; transient stdout plus one coordinator and stop/join barrier.
- **FPF reasoning:** Stderr would move progress outside the established connector. Independent locks cannot establish ordering across stdout/stderr. One coordinator preserves ownership and supplies a clear happens-before relation: stop → join → clear → permanent/diagnostic.
- **Result:** transient output stays on stdout; a presentation coordinator serializes both sinks and a stage-scoped worker is joined before later output.
- **Confidence:** high; direct architecture/contract fit.

### `DL-06` — Explicit heartbeat versus transient TTY mode

- **Status:** resolved; promoted as `SOL-04`, `SD-03`.
- **Question:** What happens when heartbeat is explicitly enabled on a TTY?
- **Facts:** Issue #9 describes transient TTY liveness and optional newline heartbeat for CI/redirected human output, but does not specify combined behavior. It rejects excessive log noise.
- **Alternatives:** emit both; ignore heartbeat on TTY; make explicit heartbeat replace transient output.
- **FPF reasoning:** Emitting both violates the noise constraint. Ignoring an explicit caller setting makes behavior depend on TTY and hides caller intent. Replacement yields one deterministic liveness mechanism.
- **Result:** positive heartbeat replaces transient animation regardless of TTY and always emits ANSI-free permanent lines.
- **Confidence:** high; deterministic and least-noisy composition.

### `DL-07` — Shimmer, color fallback and duration precision

- **Status:** resolved; promoted as `SD-04`–`SD-06`.
- **Question:** What exact visual/fallback and duration rules close the issue's implementation ambiguities?
- **Facts:** Issue #9 requires a continuous full-line Codex-style shimmer, one-second timer updates, respect for `NO_COLOR`, no non-TTY ANSI, second/compound durations and no milliseconds. It provides no canonical palette.
- **Alternatives:** configurable theme/frame rate; fixed true-color-only theme; fixed tiered palette with a no-color fallback.
- **FPF reasoning:** A configurable theme expands the public contract without operator-outcome evidence. True-color-only violates graceful fallback. The fixed 10 fps treatment can degrade conservatively by terminal capability; operator feedback showed that a wrapping gradient has a visible seam, so the selected treatment is a returning highlight instead. Duration rules choose the minimum precision demonstrated by issue examples while eliminating milliseconds.
- **Result:** fixed palette/frame/fallback and rounding rules in `SD-04`–`SD-06`; only `auto|never` color policy is public. Operator feedback later replaced the wrapping gradient with a returning soft highlight, preserving the same refresh rate and fallback policy while avoiding a visible cycle seam.
- **Confidence:** medium. The aesthetic values are an engineering selection constrained by the issue, not a user-validated brand fact; they are isolated and reversible without changing workflow semantics.

### `DL-08` — C4/ADR and artifact routing

- **Status:** resolved; promoted to `brief.md` and `design.md`.
- **Question:** Which architecture/support artifacts are necessary?
- **Facts:** Existing architecture already defines CLI/config/workflow/event boundaries. This feature changes their internal collaboration and adds concurrency but no deployable/external boundary or reusable cross-feature policy. The public contract remains rooted in README.
- **Alternatives:** embedded C3; separate C3/contract/sequence docs; ADR plus separate design pack.
- **FPF reasoning:** Embedded C3 covers the changed bindings and keeps canonical contract ownership stable. An ADR would incorrectly promote feature-local mechanics into reusable policy; separate views would duplicate a bounded design.
- **Result:** one `design.md` with embedded C3 and concurrency contract, plus the explicitly required decision log.
- **Confidence:** high.

### `DL-09` — Current upstream wording versus the to-be contract

- **Status:** resolved; promoted to `STEP-05` and cycle record.
- **Question:** Do current PRD/product statements that all workflow stdout is machine-readable conflict with the selected future human mode?
- **Facts:** Root README is the sole public contract owner and currently defines only `kv`. PRD/product/ops documents derive from that current contract. Issue #9 requires the root README and dependent Memory Bank documents to be updated when the feature is delivered.
- **Alternatives:** edit dependent docs now while README/source remain unchanged; treat the conflict as permanent; keep current-state owners unchanged until an atomic contract update in execution.
- **FPF reasoning:** Mixing to-be claims into dependent current-state documents before their canonical owner changes would invert dependency direction. The apparent conflict is temporal, not a competing accepted contract: feature design is planned state, while README/PRD describe delivered state.
- **Result:** `STEP-05` updates README first and then all affected dependents atomically with implementation; the package explicitly records this current/to-be boundary.
- **Confidence:** high; canonical ownership and issue acceptance are explicit.

### `DL-10` — Renderer selection on startup/configuration failure

- **Status:** resolved; promoted as `SD-09`.
- **Question:** Which format renders failures that occur before full configuration resolves?
- **Facts:** Existing app behavior emits legacy kv startup/terminal records for applicable flag/config failures. Semantic format must be deterministic, but a format value can itself be invalid or unavailable before Git-root/config discovery succeeds.
- **Alternatives:** guess human from partial inputs; emit diagnostics only; retain kv until format resolves, then use the selected renderer for later failures.
- **FPF reasoning:** A partial guess violates the single-resolver contract; suppressing existing records breaks compatibility. The third option preserves known behavior and switches only when the setting becomes an assured fact.
- **Result:** pre-resolution failures retain legacy kv records; post-resolution failures use the selected renderer.
- **Confidence:** high.

### `DL-11` — Human-readable default

- **Status:** resolved by the user; promoted to `brief.md`, `design.md` and the root README.
- **Question:** Should `human` remain opt-in or become the built-in log format?
- **Facts:** The delivered `kv` stream remains available through the existing explicit setting. The user requested `human` as the default for normal operation.
- **Result:** `human` is the built-in default; `kv` remains an explicit deterministic compatibility mode. Invalid/unresolvable startup format falls back to the built-in human renderer.
- **Confidence:** high; direct product decision.

### `DL-12` — Human-line context and ordering

- **Status:** resolved by the user; promoted to `brief.md`, `design.md` and the root README.
- **Question:** What minimum context must an operator see on each human progress line?
- **Facts:** The user found the initial human output too sparse and explicitly requested a timestamp on every line, the exact bracket form `[gpt-5.6-sol/high]`, no visual separator between context and message, and visible real attempt budgets. The user then rejected `fixes n/max` as semantically misleading during review, selected compact `[attempt/max]`, and placed it before the model. They also approved liveness as the sole interactive stage-start indicator.
- **Result:** Every human permanent and liveness line begins with local `HH:MM:SS`; retryable stage lines continue with `[attempt/max] [model/reasoning-effort]` and a single space before the message. Review/fix use cycle/max-cycles; CI recovery uses phase/max-ci-recoveries. Interactive human output omits permanent stage-start lines in favour of liveness; non-TTY retains them. The run terminal line retains the timestamp but no stage context.
- **Confidence:** high; direct product decision.

## Review-Improve Cycles

Cycle records are appended after each complete review. A cycle lists only `critical` and `important` findings for remediation; `minor` findings are recorded but not changed unless they block a higher-severity correction.

### Cycle 1 — full package integrity review

- **Review scope:** issue #9, Feature Flow, validation/testing policy, root README, related PRD/product/architecture claims, and all FT-009 documents.
- **Critical:** none.
- **Important:** `IMP-01` missing `REQ-* → SC-*` links; `IMP-02` transient timer cadence conflicted with issue wording; `IMP-03` planned human mode appeared to conflict with current machine-readable-only upstream claims; `IMP-04` heartbeat labels and startup-failure renderer fallback were incomplete.
- **FPF closures:** `DL-09` resolves the temporal owner conflict; `DL-10` resolves pre-format startup failures. The heartbeat composition remained covered by `DL-06`; timer cadence follows the direct issue fact.
- **Changes:** added scenario traceability for every requirement, separated transient whole-second cadence from permanent duration formatting, specified exact heartbeat labels, recorded the current/to-be documentation handoff, and defined kv fallback until format resolution.
- **Minor:** none used to drive changes.
- **Human gate:** no.

### Cycle 2 — post-remediation convergence review

- **Review scope:** all FT-009 artifacts after Cycle 1, Feature Flow tracker obligations, embedded C3 and negative coverage.
- **Critical:** none.
- **Important:** `IMP-05` C3 showed the resolved configuration flowing in the wrong direction; `IMP-06` issue #9 lacked the required package backlink and `NEG-01` did not explicitly name the rejected `heartbeat + kv` combination.
- **FPF closures:** none; the fixes follow direct existing governance and `CTR-01` facts.
- **Changes:** corrected the C3 config request/result directions, expanded `NEG-01`, and added issue tracker links to all existing core artifacts.
- **Minor:** none used to drive changes.
- **Human gate:** no.

### Cycle 3 — final convergence review

- **Review scope:** complete FT-009 package, canonical-ID handoffs into the implementation plan, issue backlink, docs lint and diff hygiene.
- **Critical:** none.
- **Important:** none.
- **FPF closures:** none required.
- **Changes:** no content corrections; recorded the clean terminal review state.
- **Verification:** `make docs-lint`, `git diff --check`, canonical-ID coverage audit and issue-comment readback pass.
- **Minor:** none changed.
- **Human gate:** no; review-improve stopped early by rule.

## Human Gate

No unresolved feature-document question meets the human-gate threshold. `AG-01` was resolved by the user's 2026-07-22 instruction to implement and publish issue #9 end-to-end.

## Execution Approval

- **Approval:** `AG-01` resolved.
- **Evidence:** user instruction dated 2026-07-22 explicitly requests end-to-end implementation, local validation, commit, push, PR creation and review/CI convergence.
- **Consequence:** `brief.md` moved to `delivery_status: in_progress`; execution may begin under the selected high-risk validation profile.

## Implementation Review-Fix Loop

### Iteration 1 — concurrency and failure semantics

- **Critical:** none.
- **High:** stop/cancel could race the first transient write; unexpected stage statuses could be rendered as ordinary failures; CI-fix liveness write failure incorrectly selected exit `3` instead of operational exit `2`.
- **Fix:** added pre-write cancellation/stop gate, strict human status rendering and uniform operational handling for presentation failures; added regression tests.

### Iteration 2 — cancellation assurance

- **Critical:** none.
- **High:** cancellation coverage did not exercise an active workflow Agent stage end-to-end.
- **Fix:** added an active review cancellation test proving operational termination and no writes after the stop/join barrier.

### Iteration 3 — configuration precedence assurance

- **Critical:** none.
- **High:** new settings lacked one test that composed all four explicit source levels in conflict.
- **Fix:** added CLI → project → user → environment cascade coverage for `log-format`, `heartbeat` and `color`.

### Iteration 4 — raw-output isolation assurance

- **Critical:** none.
- **High:** the human app acceptance test did not directly assert absence of raw Codex review text.
- **Fix:** added an explicit negative assertion to the human end-to-end fake-runner test.

### Iteration 5 — final convergence

- **Critical:** none.
- **High:** none.
- **Result:** local implementation, contract, race, vet, documentation and diff gates pass; PR [#13](https://github.com/dapi/code-converge/pull/13) is mergeable and required [Verify run](https://github.com/dapi/code-converge/actions/runs/29901470990) passed. No critical/high findings remain.
