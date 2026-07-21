---
title: "FT-005: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Аудируемый журнал FPF reasoning, разрешённых конфликтов, review-improve циклов и human gates для FT-005."
derived_from:
  - brief.md
  - ../../../README.md
  - https://github.com/dapi/reviewer/issues/5
status: active
audience: humans_and_agents
---

# FT-005: Decision Log

## Artifact Contract

- **Role:** provenance companion for FPF decisions and document review cycles.
- **Owns:** questions considered, source carriers, alternatives, reasoning, confidence, conflict-resolution and gate status.
- **Must not define:** requirements, public CLI contract, selected design, or execution sequence. Accepted local solution facts are promoted to `design.md`; problem/acceptance changes go to `brief.md`.

## Decision Entries

### `DL-01` — Route, package identity, and validation floor

- **Status:** resolved from governance and issue facts.
- **Question:** Which flow/package/profile apply?
- **Facts:** issue #5 is one delivery-unit and explicitly changes CLI/config contracts and Codex stage configuration; `memory-bank/features/README.md` maps package identifiers to issue IDs.
- **FPF reasoning:** strict separation keeps lifecycle routing distinct from assurance depth. Feature Flow owns delivery artifacts; contract-bearing executable change selects `standard`, with no stronger validation trigger present.
- **Result:** Feature Flow, package `FT-005`, `standard` profile in `brief.md`.
- **Confidence:** high.

### `DL-02` — Missing Finalize and Fix CI effort surfaces

- **Status:** resolved and promoted as `SD-02`.
- **Question:** Does the feature only profile existing effort options, or add stage-specific effort options for Finalize and Fix CI?
- **Available facts:** issue #5 requires model and reasoning effort for all four stages and says every resulting stage setting is visible; its profile table supplies Finalize and Fix CI efforts. Current README/code expose effort only for Review and Fix findings, and current adapter omits effort for Finalize/Fix CI.
- **Alternatives:** (A) keep missing options and make those two efforts profile-only; (B) add symmetric effort configuration/override surfaces; (C) omit those efforts.
- **FPF reasoning:** the Evidence Graph anchors the required eight settings to the issue/profile table and the gap to current README/code. Deduction rejects C because it violates `REQ-02`; A prevents the stated per-stage override property from applying uniformly and makes profile selection less reversible. B is the only candidate satisfying complete resolution, explicit overrides, and config explainability without inventing a new stage or precedence source.
- **Result:** add Finalize and Fix CI reasoning-effort public settings using the established stage naming pattern; they participate in the same four explicit sources and outrank profiles.
- **Confidence:** high; derived from explicit outcome plus current contract gap.

### `DL-03` — Cross-dimensional precedence and source reporting

- **Status:** resolved and promoted as `SD-03`, `CTR-02`, and `CTR-03`.
- **Question:** How does a mode selected at one source interact with a stage override selected at another source, and what source is rendered for inherited values?
- **Available facts:** issue states per-stage CLI/project/user/environment overrides are higher priority than the selected profile and retain existing precedence; existing source order is CLI > project > user > environment > built-in; config output must show mode/source and every resolved stage setting.
- **Alternatives:** (A) compare mode and stage value by source rank; (B) resolve mode independently, then use its profile only for stage fields absent from all explicit sources; inherited values report the profile identity.
- **FPF reasoning:** SelectorMechanism requires criteria without hidden tie-breakers. The issue explicitly makes “explicit stage value vs profile” the first criterion and existing source rank the second criterion only among explicit candidates. A would let a high-source mode erase a lower-source explicit stage value, contradicting acceptance. B preserves both criteria and produces an auditable source.
- **Result:** resolve mode by normal source precedence; resolve each stage field from explicit sources first, otherwise from effective profile. Render inherited source as `<mode> profile` and explicit equal-string values with their actual explicit source. The global built-in comparison remains the `fast` value and is rendered only when the effective string differs.
- **Confidence:** high.

### `DL-04` — Public mode naming and value validation

- **Status:** resolved and promoted as `SD-01` and `CTR-01`.
- **Question:** Which CLI/env/file names expose the mode, and how are values validated?
- **Available facts:** issue consistently calls the concept “mode”; existing contract maps every scalar `<name>` to flag `--<name>`, environment `REVIEWER_<NAME>`, and file `<name>`; explicit scalar values are trimmed and required configuration fails closed.
- **Alternatives:** `mode`, `profile`, or stage-specific selectors.
- **FPF reasoning:** Ontological parsimony and current naming transformations select the one concept named by the issue without aliases or extra selectors. Deduction yields `--mode`, `REVIEWER_MODE`, and `mode`; case-sensitive enum validation matches the documented lowercase values and avoids hidden normalization.
- **Result:** public key `mode` across the established naming transformations; accepted values exactly `fast|best`; empty/unknown explicit values fail.
- **Confidence:** medium-high; names are a direct application of the existing public naming rule, not a new requirement.

### `DL-05` — Apparent default-change contradiction

- **Status:** resolved in problem interpretation.
- **Conflict:** issue Goal/acceptance says an unconfigured implemented run uses `fast`, while Non-goals says not to change currently configured defaults “as part of this planning issue”; root README labels profiles proposed and non-operative until implementation.
- **FPF reasoning:** the reasoning cycle separates time/scope contexts. The non-goal and README describe the pre-implementation planning state; Goal and acceptance describe the delivered behavior. Treating the non-goal as a permanent constraint would make the main acceptance criterion impossible.
- **Result:** documentation work does not prematurely change runtime defaults; implementation makes `fast` the built-in mode and its values the effective unconfigured stage settings, with the public README updated atomically with code.
- **Conflict resolution:** issue Goal/acceptance governs delivered behavior; the Non-goal governs the planning-only state before delivery.
- **Confidence:** high.

## Review-Improve Cycles

Review cycles append findings and changes here. A cycle may close with no critical/important findings; minor findings remain untouched unless they affect a higher-severity issue.

### Cycle 1

- **Review summary:** Package anatomy, owner boundaries, requirements/scenarios/checks/evidence, design coverage, and plan mapping are present; `make docs-lint` and `git diff --check` pass. No critical finding.
- **Important findings:** `I-01` stage `built-in` comparison/source semantics were unspecified for `best` and equal-string explicit overrides; `I-02` exact profile values lacked a stable contract ref in design/plan traceability; `I-03` Feature Flow tracker linkage was not yet recorded.
- **FPF questions closed:** `I-01` used the existing README value-based built-in rule, issue explainability requirement, and explicit-equal source distinction to select global fast baseline plus profile identity; no new external fact was assumed. `I-02` is an ownership repair: issue-owned values are promoted as design contract `CTR-06`.
- **Changes:** added `CTR-06`, `CTR-07`; refined `CTR-03`, alternatives, design verification/traceability, test strategy, realization mapping, and `STEP-02`. Tracker linkage is completed by the issue comment referenced in the next cycle/final report.
- **Human gate:** no.

### Cycle 2

- **Review summary:** Cycle 1 findings are closed and issue [#5](https://github.com/dapi/reviewer/issues/5#issuecomment-5038254349) records routing/package links. Requirements, exact profile contract, config source/built-in semantics, and plan mappings agree. No critical finding.
- **Important findings:** `I-04` `design.md` claimed C4 was not required even though Feature Flow forbids that conclusion when CLI/env/file interaction contracts change.
- **FPF questions closed:** none; the canonical Feature Flow C4 trigger directly determines C1 and leaves no competing material option.
- **Changes:** replaced `C4-00` with embedded `C4-01` C1 System Context, indexed the view in Design Pack, clarified actor/config/Codex connectors, and updated design/plan traceability.
- **Human gate:** no.

### Cycle 3

- **Review summary:** C4/design-pack coverage, requirement-to-evidence chains, and full design realization mapping are present; documentation lint and diff integrity pass. No critical finding.
- **Important findings:** `I-05` `SD-04` and resolution pseudocode retained the obsolete `profile:<mode>` label while `CTR-03`/`DL-03` required `<mode> profile`; `STOP-01` omitted the newly added profile/built-in contracts `CTR-06` and `CTR-07`.
- **FPF questions closed:** none; this was a direct consistency repair against the already accepted single owner.
- **Changes:** aligned `SD-04` and pseudocode with `CTR-03`; expanded `STOP-01` through `CTR-07` and named source/built-in rendering in its trigger.
- **Human gate:** no.

### Cycle 4

- **Review summary:** Public/problem/solution/execution semantics remain aligned and checks pass. No critical finding.
- **Important findings:** `I-06` frontmatter created a circular provenance edge (`design.md` derived from `decision-log.md`, while `decision-log.md` derived from `design.md`); the non-solution decision log was also listed inside Design Pack.
- **FPF questions closed:** none; Feature Flow ownership rules directly classify decision log as support/provenance rather than a solution artifact.
- **Changes:** removed `design.md` from decision-log `derived_from` and removed decision log from Design Pack; feature README remains its navigation owner.
- **Human gate:** no.

### Cycle 5

- **Review summary:** Final cross-document review found no critical or important issues. Problem/solution/execution ownership, issue/default conflict resolution, exact profile and source contracts, C1 coverage, requirement-to-evidence traceability, realization mapping, and provenance are coherent. `make docs-lint` and `git diff --check` pass.
- **Critical / important findings:** none.
- **FPF questions closed:** none.
- **Changes:** no semantic changes; this entry records early stop on a clean final review.
- **Human gate:** no.

## Final Review Status

- **Status:** `done`.
- **Cycles:** 5.
- **Closed critical findings:** none were found.
- **Closed important findings:** `I-01`–`I-06`.
- **Remaining critical / important findings:** none.

### Implementation Convergence

- **Implementation review:** independent `codex review --base master` completed with no findings, `overall_correctness=patch is correct`, and confidence `0.93`.
- **Local evidence:** `make verify`, `make dist`, and `git diff --check` passed.
- **Hosted evidence:** PR [#6](https://github.com/dapi/reviewer/pull/6) is mergeable and its required Verify check passed before lifecycle closure; the final documentation commit is subject to the same required check.
- **Remaining critical / high findings:** none.
