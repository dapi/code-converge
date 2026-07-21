---
title: "FT-002: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Аудируемый журнал review/FPF reasoning, unresolved decisions и human gates для FT-002; не является canonical owner problem, solution или execution facts."
derived_from:
  - brief.md
  - ../../prd/PRD-001-reviewer-cli.md
  - ../../../README.md
  - ../../engineering/architecture.md
  - ../../ops/release.md
status: active
audience: humans_and_agents
---

# FT-002: Decision Log

## Artifact Contract

- **Role:** evidence/provenance companion for review-improve cycles and FPF analysis.
- **Owns:** questions considered, available carriers, alternatives, confidence, gate status, and promotion route.
- **Must not define:** requirements, selected design, public CLI contract, release policy, or execution sequence. A human-approved feature-local solution is promoted to `design.md` as `SD-*`; reusable architecture is promoted to an ADR; problem/acceptance changes go to `brief.md`.

## Decision Entries

### `DL-01` — Feature route and validation floor

- **Status:** resolved from existing facts.
- **Question:** Which delivery flow and validation floor apply?
- **Facts:** issue #2 explicitly routes one coherent delivery-unit through Feature Flow; it creates a public CLI/config contract, integrates with external Codex, and must produce a distributable release artifact.
- **FPF reasoning:** strict separation keeps lifecycle routing distinct from assurance depth. Feature Flow governs the artifact lifecycle; the new external trust boundary triggers `high-risk`; the release artifact adds all `release-deployment` obligations.
- **Result:** Feature Flow; `high-risk` profile composed with release/deployment obligations. Canonical profile owner is `brief.md`.
- **Confidence:** high; directly entailed by issue #2 and validation-profile rules.

### `DL-02` — Human gate on missing solution/release facts

- **Status:** resolved by explicit user delegation to apply FPF and continue.
- **Questions:** `DEC-01`–`DEC-05` from `brief.md`.
- **Available facts:** the public contract defines observable workflow behavior; `PRD-001` explicitly leaves platform/install and acceptance ownership open; engineering architecture explicitly leaves process policies open; release operations says no artifact matrix, versioning, distribution, approval, or rollback policy exists; no representative Codex report corpus or exact finalization grammar is supplied.
- **FPF reasoning:** Evidence Graph review finds no carrier for selecting concrete values. The Reasoning Cycle can enumerate hypotheses and consequences, but Trust/Assurance remains unsubstantiated because no empirical corpus, product decision, or release owner exists. Selecting defaults would invent facts and could materially change compatibility, safety, supported platforms, and acceptance cost.
- **Result:** use the admissible, fail-closed portfolio recorded in `DL-03`; promote accepted solution facts to `design.md` as `SD-01`–`SD-05`.
- **Confidence:** medium-high because every selected behavior is either entailed by the public contract, verified against the installed Codex CLI/manual, or bounded so unsupported behavior fails closed.

### `DL-03` — FPF selection for former human gates

- **Status:** resolved.
- **Authority:** user instruction “Используй FPF для принятия решений и продолжай”.
- **Method:** SelectorMechanism preserved multiple criteria (contract fit, safety, testability, compatibility, delivery cost) without hidden weighting. Evidence-missing candidates abstained; Agent-Tools-CAL supplied explicit budgets/stop conditions; Evidence Graph anchors are the root README, repository policies, current Codex manual, installed `codex-cli 0.144.5` help, Go/CI configuration, and deterministic tests.
- **`HG-01` result:** Go `1.21.13`; `darwin/linux` on `amd64/arm64`; deterministic `tar.gz` artifacts and `SHA256SUMS`; manual install by copying `reviewer` onto `PATH`; no registry/package-manager publication or signing promise. Artifact deletion/rebuild is the backout unit.
- **`HG-02` result:** finalization uses `codex exec --output-schema` and `--output-last-message`; strict JSON contains one verdict and exactly four typed step outcomes, followed by an explicit consistency matrix in `design.md`.
- **`HG-03` result:** review remains ordinary `codex review --uncommitted`; a narrow anchored clean allowlist and explicit `[P0]`–`[P3]` finding records are accepted. Anything else, including mixed clean/finding claims, abstains as operational failure. Fixtures are the compatibility contract.
- **`HG-04` result:** acceptance is the required CI plus independent `codex review`; deterministic fake runner/executable fixtures are the mandatory corpus, with artifact build/install/smoke checks for distribution.
- **`HG-05` result:** no new timeout option is invented; parent cancellation is propagated to child processes. Reviewer does not override Codex sandbox, approvals, or network policy; those remain inherited Codex/operator configuration. Tests cover cancellation and captured stdio.
- **Consequences:** solution is intentionally conservative. Compatibility expands only by adding evidence-backed fixtures; it never widens through heuristic clean detection.

## Human Gate

Resolved by `DL-03`; the table below is retained as historical provenance.

| Gate | Question | Available facts | Options requiring a human choice | Risk of a wrong choice | Needed from a human |
| --- | --- | --- | --- | --- | --- |
| `HG-01` | What is the first-release baseline? | Go CLI; released binary needs no Go runtime; no official release policy exists. | Select minimum Go version, OS/arch targets, archive/binary format, checksum/signing, install path/channel, versioning, approval and rollback owner; or explicitly defer official release while defining a narrower evidence-only deliverable. | Unsupported promises, unreproducible artifacts, unusable install path, or accidental release-policy creation. | Approved release matrix and owners, including which obligations may be deferred. |
| `HG-02` | What exact finalization response grammar is accepted? | Verdict vocabulary and four step names/statuses are fixed; encoding/order/cardinality are not. | Approve a strict line/JSON/text grammar and consistency matrix, or provide an existing canonical protocol. | False success, inability to parse legitimate responses, or unstable prompt/parser coupling. | Canonical grammar plus representative valid/invalid examples. |
| `HG-03` | Which ordinary Codex review reports form the admissible classification corpus? | Must use normal `codex review`; ambiguous output fails closed; priority mapping is fixed. | Provide versioned clean/finding/malformed samples and supported Codex version envelope, or approve a deliberately narrow recognizer with explicit compatibility limits. | False-clean classification or rejection of normal reports after Codex output drift. | Representative sanitized reports and the supported compatibility envelope. |
| `HG-04` | Who accepts the product and what corpus is sufficient? | `PRD-001` leaves both open; live Codex/remotes/hosted CI are forbidden in deterministic automated tests. | Name an acceptance owner and approve a fake-repository/fixture matrix plus any isolated manual smoke evidence. | Passing tests may not establish the intended product outcome; closure could lack authority. | Named approver and minimum acceptance corpus/manual evidence. |
| `HG-05` | What process timeout/cancellation/sandbox/network policy applies? | External execution is a trust boundary; architecture explicitly leaves these policies unresolved. | Approve no internal timeout with OS cancellation, configurable/default timeout semantics, and sandbox/network inheritance or restriction rules. | Hung runs, unsafe privilege/network assumptions, incompatible CLI surface, or untestable failure semantics. | Chosen policy and whether it changes the public configuration contract. |

## Review Cycles

### Cycle 1

- **Summary:** Package integrity is sufficient for Problem Ready, but Solution Ready and Plan Ready are materially blocked by `HG-01`–`HG-05`.
- **Critical:** exact finalization protocol and safe review classification corpus are absent (`HG-02`, `HG-03`); first-release contract is absent although distribution is acceptance scope (`HG-01`).
- **Important:** acceptance authority/corpus and process execution policy are absent (`HG-04`, `HG-05`).
- **FPF closures:** `DL-01` was closed from current evidence; `DL-02` established that the remaining choices cannot be closed without invented facts.
- **Changes:** bootstrap `README.md`, active canonical `brief.md`, and this provenance log were created before the review-improve loop; no cycle auto-fixes were made after the human gate was detected.
- **Human gate:** yes; review-improve stopped immediately as required.

### Cycle 2

- **Summary:** User delegated the material choices to FPF. All former gates were resolved through evidence-bounded, fail-closed selections and promoted to `design.md`.
- **Critical / important:** none remain in the problem/solution package before implementation.
- **FPF closures:** `HG-01`–`HG-05` closed in `DL-03`.
- **Changes:** brief blockers removed; active design and implementation plan added; feature index and parent index completed.
- **Human gate:** no.

### Cycle 3

- **Summary:** Implementation, canonical operational owners, evidence, and lifecycle state converged after four code review/fix iterations.
- **Critical / important:** none remain. Independent review iteration 4 found no correctness issue.
- **FPF closures:** Evidence Graph links acceptance claims to tests, CI, distribution checks and the independent review session; Trust/Assurance is sufficient for the declared local-artifact scope.
- **Changes:** stale architecture/development/release/config facts reconciled; concrete evidence linked; `brief.md` moved to `done`; execution plan archived.
- **Human gate:** no.
