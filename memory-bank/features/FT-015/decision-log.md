---
title: "FT-015: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Аудируемый журнал FPF reasoning, conflict resolution, review-improve циклов и human gates для FT-015."
derived_from:
  - brief.md
  - design.md
  - https://github.com/dapi/code-converge/issues/15
status: active
audience: humans_and_agents
---

# FT-015: Decision Log

## Artifact Contract

- **Role:** provenance companion for FPF decisions, document review and gate status.
- **Owns:** source facts, alternatives, reasoning, confidence, conflict resolution, open questions and cycle findings.
- **Must not define:** requirements, public contract, selected solution or execution sequence. Accepted facts are promoted to `brief.md`, `design.md` or the root README as appropriate.

## FPF Method

For each material question, this log bounds the claim to the feature, separates evidence from an implementation choice, compares alternatives against explicit constraints and promotes only the selected fact to its canonical owner. The confidence of a result cannot exceed its weakest source; unknown future schemas remain unknown rather than becoming assumed compatibility.

## Decision Entries

### `DL-01` — Route and assurance floor

- **Status:** resolved; promoted to `brief.md`.
- **Question:** Which delivery flow and validation profile apply?
- **Facts:** Issue #15 changes review parsing, a terminal workflow branch and CLI documentation. Routing excludes Small Change when a CLI behavior/contract changes; the issue is one independently verifiable delivery-unit. No listed high-risk trigger (security, persistent data, concurrency, migration or cross-system integration) is present.
- **Alternatives:** Small Change/low-risk; Feature Flow/standard; Feature Flow/high-risk.
- **FPF reasoning:** Lifecycle shape and assurance depth are distinct. The changed external parser and state transition require Feature Flow and standard regression/contract evidence; raising to high-risk lacks a documented trigger.
- **Result:** Feature Flow package `FT-015`, validation profile `standard`, no downgrade.
- **Confidence:** high; direct issue and governance facts.

### `DL-02` — Strict structured report boundary

- **Status:** resolved; promoted as `CTR-01`–`CTR-03`, `SD-01`–`SD-02`.
- **Question:** What makes a structured response safe to classify?
- **Facts:** Issue #15 supplies the complete clean top-level object from `codex-cli 0.145.0`. Installed local `codex --version` reports 0.145.0. A recorded local Codex review contains the same top-level fields and finding entries with `title`, `body`, `confidence_score`, numeric `priority`, `code_location.absolute_file_path`, and `code_location.line_range.start/end`. Existing finalization parsing rejects duplicate keys, unknown fields and trailing data; current review parsing fails closed.
- **Alternatives:** accept any JSON with `findings`; validate only clean reports; validate the observed complete object recursively.
- **FPF reasoning:** The first two alternatives allow incomplete/unrelated output to select a clean or counter-bearing transition. Exact validation preserves the existing trust boundary while accepting evidenced output. A future incompatible shape has no current evidence and stays rejected.
- **Result:** exact observed schema, numeric priority `0`–`3`, strict nested validation, duplicate/trailing-data rejection; structured findings keep the normalized report for fix-findings.
- **Confidence:** high for 0.145.0 shape; intentionally bounded for future incompatible versions.

### `DL-03` — Clean no-change transition

- **Status:** resolved; promoted as `CTR-04`, `SD-03`, `INV-03`.
- **Question:** Where and how does the no-change run stop?
- **Facts:** Issue #15 explicitly names staged, unstaged and untracked changes, requires successful no-op without empty commit, and says a clean structured review with changes continues to existing finalize. Workflow currently owns transitions and calls finalization after clean review; architecture assigns working-directory process execution to the runner and forbids the Codex boundary from deciding exit policy.
- **Alternatives:** always invoke finalization; let finalization discover no change; query status in the workflow transition through a fakeable collaborator.
- **FPF reasoning:** Only a pre-finalize status query can prove no finalization invocation. Keeping the branch in workflow preserves responsibility boundaries; a collaborator makes the query deterministic in tests without embedding a Git policy in parser code.
- **Result:** clean review → repository-status query → no-change success or unchanged finalize path; query error is operational failure.
- **Confidence:** high; direct issue acceptance and existing architecture ownership.

### `DL-04` — Current versus to-be upstream documents

- **Status:** resolved; promoted to `STEP-05`.
- **Question:** Should root README/domain documents be edited while only the feature plan exists?
- **Facts:** README is the current public contract; domain rules/states derive from it. Issue #15 requires them updated with the delivery. Source still implements the old transition.
- **Alternatives:** update upstream wording now; leave it permanently unchanged; update atomically during implementation after README.
- **FPF reasoning:** Editing dependent current-state docs now would assert an undelivered contract and invert their dependency on README. The discrepancy is temporal, not a conflict between accepted facts.
- **Result:** package contains to-be solution; execution updates README first and downstream owners in the same delivery change.
- **Confidence:** high; direct canonical-ownership evidence.

### `DL-05` — Evidence boundary of strict validation

- **Status:** resolved; promoted as `CTR-01`–`CTR-02` and `NEG-01`.
- **Question:** May the feature impose content/range rules on structured fields beyond their observed shape and JSON types?
- **Facts:** Issue #15 and locally observed Codex 0.145.0 samples establish field names, nesting and observed values, but do not specify that prose is non-empty, confidence is constrained to a range, or a line range has positive/ordered endpoints. They do require strict validation and fail-closed behavior.
- **Alternatives:** invent semantic ranges; validate only documented/observed structure and types; leave all fields unchecked.
- **FPF reasoning:** Adding unknown semantic constraints would reject a supported report without evidence; leaving types/keys unchecked would weaken the explicit safety requirement. The bounded middle option verifies all evidenced structural facts and rejects unknown/ill-typed data without claiming an unprovided Codex protocol.
- **Result:** `CTR-01`–`CTR-02` require exact fields/nesting and JSON types, with `priority` constrained to evidenced values `0`–`3`; no unsupported content/range rule is added.
- **Confidence:** high; direct evidence boundary and issue constraint.

## Review-Improve Cycles

Cycle entries are appended after each complete review. Only critical and important findings drive corrections; minor findings are recorded but not changed unless they block a higher-severity correction.

### Cycle 1 — package integrity review

- **Review scope:** issue #15, Feature Flow, validation/testing policy, current README/domain/architecture owners, source parser/workflow, and all FT-015 artifacts.
- **Critical:** none.
- **Important:** `IMP-01` `CTR-01`/`CTR-02` introduced unproven non-empty/range/line-order semantic constraints; `IMP-02` issue #15 had no required links to the new core artifacts.
- **FPF closures:** `DL-05` bounds strict validation to available evidence and resolves `IMP-01` without weakening key/type/duplicate/trailing-data validation.
- **Changes:** removed unsupported semantic-range claims from the contract and negative inventory; posted feature-package links to issue #15.
- **Minor:** none changed.
- **Human gate:** no.

### Cycle 2 — post-remediation convergence review

- **Review scope:** all FT-015 artifacts after Cycle 1, canonical-ID handoffs, issue backlink readback and public/domain current-to-be boundary.
- **Critical:** none.
- **Important:** `IMP-03` `CTR-03` still claimed rejection of any invalid value/range after `DL-05` removed unsupported semantic range rules from `CTR-01`/`CTR-02`.
- **FPF closures:** none; this is a direct wording conflict between `CTR-03` and the accepted `DL-05` evidence boundary.
- **Changes:** narrowed `CTR-03` to malformed structure, duplicate/trailing data, missing/unknown fields, invalid required-field types and invalid priority; no conflict remains.
- **Minor:** none changed.
- **Human gate:** no.

### Cycle 3 — final convergence review

- **Review scope:** complete FT-015 package after Cycle 2, Feature Flow gate inventory, canonical-ID realization, issue backlink readback, documentation lint and diff hygiene.
- **Critical:** none.
- **Important:** none.
- **FPF closures:** none required.
- **Changes:** no document correction; recorded the clean terminal review state.
- **Verification:** `make docs-lint`, `git diff --check`, Feature Flow gate/traceability audit and issue-comment readback pass.
- **Minor:** none changed.
- **Human gate:** no; review-improve stopped early because no critical or important finding remained.

## Human Gate

No human gate is open. The exact supported structured contract is bounded to evidence available for Codex CLI 0.145.0; a new incompatible format triggers `STOP-01` rather than speculative acceptance.

## Execution Approval

- **Approval:** execution authorized.
- **Evidence:** the user's current instruction explicitly requests end-to-end implementation, validation, commit, push, PR creation and review/CI convergence for the current feature context.
- **Consequence:** execution proceeded under the selected `standard` profile; after evidence completion `brief.md` moved to `delivery_status: done`.

## Implementation Review-Fix Loop

### Iteration 1 — implementation and convergence

- **Critical:** none.
- **High:** none.
- **Review signals:** table-driven structured parser fixtures; fake repository-status workflow/app tests; full local Go, vet, documentation and diff checks; independent GitHub Verify run.
- **Result:** strict structured clean/finding classification, preserved plain-text behavior, no-change finalization skip and public-contract updates converge. No critical/high finding remains.

### Iteration 2 — exact structured-key validation

- **Review finding:** `P2` — `encoding/json` matches struct fields case-insensitively, so case-variant keys could bypass the documented exact-key contract.
- **Fix:** added recursive exact-key validation before struct decoding for the response, finding, code-location and line-range objects; added case-variant and case-variant duplicate top-level fixtures.
- **Verification:** `go test ./internal/codex`, `go test ./...`, `go vet ./...`, `make docs-lint`, `git diff --check`.
- **Result:** case-variant keys are unclassifiable and cannot produce a clean review; no critical/high finding remains.
