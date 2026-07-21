---
title: "FT-007: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Аудируемый журнал FPF reasoning, конфликтов, review-improve циклов и human gates для FT-007."
derived_from:
  - brief.md
  - design.md
  - https://github.com/dapi/reviewer/issues/7
status: active
audience: humans_and_agents
---

# FT-007: Decision Log

## Artifact Contract

- **Role:** provenance companion for FPF decisions, document review and gate status.
- **Owns:** source facts, alternatives, reasoning, confidence, conflict resolution, open questions and cycle findings.
- **Must not define:** requirements, public contract, selected solution or execution sequence. Accepted solution facts are promoted to `design.md`; problem/verify changes go to `brief.md`.

## Decision Entries

### `DL-01` — Route and validation profile

- **Status:** resolved from governance and issue facts.
- **Question:** Which flow and validation floor apply?
- **Facts:** Issue #7 explicitly changes CLI/config/module/repository/release surfaces and calls for Feature Flow. Validation policy raises breaking/compatibility-sensitive public contracts to `high-risk`; release/deployment obligations apply during rollout.
- **FPF reasoning:** Separate lifecycle routing from assurance depth. The change is one delivery-unit, but its public compatibility and cross-system rollout risks rule out `standard` as the sufficient floor.
- **Result:** Feature Flow, package `FT-007`, `high-risk` profile in `brief.md`.
- **Confidence:** high.

### `DL-02` — Canonical identity boundary

- **Status:** resolved and promoted as `SD-01`–`SD-03`.
- **Question:** What does “rename project” cover?
- **Facts:** Issue scope names repository, CLI/binary, Go module/build, CLI/config, docs/prompts/tests/scripts/Makefile/CI/release/installer/distribution, configuration directories/files/env/keys, URLs and release metadata. It separately requires preservation of workflow semantics.
- **FPF reasoning:** Treat the named surfaces as one identity system, while preserving workflow behavior as an invariant. A partial rename would create mixed current identities and fail acceptance.
- **Result:** `code-converge` is canonical across all named current surfaces; residual old strings require classification.
- **Confidence:** high.

### `DL-03` — Clean-break compatibility policy

- **Status:** resolved by `dapi`; promoted to `CON-05`, `SD-04`, and `SOL-05`.
- **Question:** Should old command/config names receive a clean break, aliases, or a deprecation period?
- **Available facts:** Issue #7 requires an explicit compatibility decision and migration notes. Current repository facts show old `reviewer` / `.reviewer` / `REVIEWER_*` surfaces; `dapi` confirmed the project is not in operation and selected clean break.
- **Alternatives:** (A) clean break with migration notes; (B) command/config aliases; (C) time-bounded deprecation with warnings and eventual removal.
- **FPF reasoning:** The choice changes observable public behavior, migration burden and implementation/test scope. Evidence is insufficient to rank the alternatives without inventing a product decision; therefore the autonomy boundary requires a human gate.
- **Result:** clean break. Old `reviewer` command/configuration names receive no aliases, fallback reads, migration tooling, or deprecation period because the project is not in operation.
- **Confidence:** high; explicit owner decision.

### `DL-04` — External rename approval

- **Status:** resolved by `dapi`; promoted to `SOL-04` and `RB-02`.
- **Question:** Who authorizes and owns the live GitHub rename and release rollout?
- **Available facts:** Issue acceptance requires the new URL and redirect check where available; the repository and release systems are external and the current worktree has no approval record or rollout owner.
- **FPF reasoning:** External state mutation and release publication require explicit approval, owner, timing and backout authority. This does not justify inventing those details in the feature package.
- **Result:** `dapi` authorizes and owns the GitHub rename and release/installer rollout, including rollback/corrective release decisions.
- **Confidence:** high; explicit owner decision.

## Review-Improve Cycles

### Cycle 1 — bootstrap package review

- **Review scope:** `README.md`, `brief.md`, `design.md`, `decision-log.md`, issue #7, Feature Flow, routing and validation/testing policy.
- **Critical:** none.
- **Important:**
  - `IMP-01` Old-name compatibility policy is required by the issue but absent from repository facts; it blocks selected solution semantics and the implementation plan. Resolved by `DL-03` human gate, not by assumption.
  - `IMP-02` GitHub rename/release ownership and approval are externally effective and absent; rollout evidence cannot be claimed. Resolved by `DL-04` human gate.
- **Minor:** Not changed because no minor finding is needed to close a critical/important finding.
- **Changes:** bootstrapped FT-007 package; added canonical problem/verify contract, solution owner, FPF provenance, explicit traceability and human-gate records. At that time `implementation-plan.md` was intentionally deferred until the gates were resolved.
- **Gate:** yes — `HG-01` blocks Solution Ready; `HG-02` blocks rollout/Done.

### Cycle 2 — decision closure review

- **Review scope:** issue #7 owner decisions, `brief.md`, `design.md`, `decision-log.md`, Feature Flow and validation policy.
- **Critical:** none.
- **Important:** none after `DL-03` and `DL-04` were resolved.
- **Minor:** historical Memory Bank references will be updated as identity documentation; generic human-review terminology is not a product identity contract.
- **Changes:** promoted the clean-break and `dapi` rollout decisions, activated `design.md`, and authorized creation of `implementation-plan.md`.
- **Gate:** no.

### Cycle 3 — implementation convergence review

- **Review scope:** source, tests, Makefile, installer, distribution builder, GitHub Actions, README, Memory Bank and FT-007 traceability.
- **Critical:** none.
- **Important:** none after correcting the PRD filename, FT-007 index navigation and release smoke paths.
- **Minor:** the Linux archive smoke cannot run on the local macOS host; it remains a required CI surface.
- **Changes:** renamed the executable/module/configuration/release identity to code-converge, preserved workflow contracts, and recorded the external repository rename as dapi-owned.
- **Checks:** make verify, make dist, git diff --check, identity inventory and sh -n scripts/install.sh pass locally; Linux archive execution is CI-only.
- **Gate:** no local implementation gate; external repository rename remains a dapi handoff.

## Human Gate Request

The human gates are resolved. The package may proceed through Plan Ready and Execution; no compatibility layer or migration path is in scope.
