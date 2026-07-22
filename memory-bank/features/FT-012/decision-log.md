---
title: "FT-012: Decision Log"
doc_kind: feature
doc_function: record
purpose: "Provenance журнала решений, FPF-анализа, review-improve циклов и human gates для FT-012."
derived_from:
  - brief.md
  - ../../flows/routing.md
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../../../scripts/install.sh
  - https://github.com/dapi/code-converge/issues/12
status: active
audience: humans_and_agents
must_not_define:
  - canonical_problem_space
  - selected_solution
  - implementation_sequence
---

# FT-012: Decision Log

## Routing Record — 2026-07-22

- **Selected flow:** Feature Flow.
- **Facts:** Issue #12 requests one independently verifiable `code-converge update` delivery unit with a new public CLI command, GitHub Release integration, checksum verification and executable replacement. The root README owns public CLI contracts; `scripts/install.sh` and release workflow already own the supported release matrix and artifact/checksum conventions.
- **Rejected routes:** Small Change is ineligible because it changes a CLI and release/rollback contract and requires explicit design. It is not a bug, incident or behavior-preserving refactoring. No evidence says this must be decomposed into multiple delivery features or requires an epic roadmap.
- **Result:** Bootstrap `FT-012` with `brief.md`; `Design required: yes`. The validation profile is `high-risk` because the public integration and executable-replacement safety trigger it; release/deployment obligations are included by profile composition.

## D-01 — Public no-change exit semantics require a human decision

- **Status:** accepted.
- **Prompt (FPF B.5.2):** Issue #12 requires a public exit-code contract and says a non-affirmative response leaves the installed binary unchanged, but does not say whether cancellation is success, a distinct non-success result, or an operational failure. It also lists multiple no-change failure modes without assigning exit behavior.
- **Facts:** Existing project public exits are `0` success, `1` findings remaining, `2` operational failure and `3` CI failure. Existing `--version` returns `0`. The issue requires `--yes` to suit unattended scripts and requires every unsuccessful path to preserve the original binary.
- **Candidates:**
  1. Reuse `0` for current-version and declined confirmation; reuse existing `2` for all update operational failures.
  2. Reuse `0` only for current-version; use a new distinct code for declined confirmation and `2` for operational failures.
  3. Treat declined confirmation as existing operational failure `2`.
- **FPF filters:** Candidates must preserve the stated unattended-script suitability, remain consistent with the existing public exit taxonomy, and give callers an unambiguous success/cancellation/failure signal. The available facts establish the existing codes but do not rank the cancellation semantics or authorize a new code.
- **Selection (FPF B.5.2):** Candidate 1 is the prime hypothesis: current version and declined/default confirmation return `0`; malformed metadata, unsupported target and all network/download/checksum/replacement failures return existing operational code `2`. Status, release notes and the prompt write to stdout; error diagnostics write to stderr.
- **Rationale:** It has the best scope fit and consistency with the established `0/1/2/3` taxonomy, introduces no new public code, keeps a user's intentional no-change path non-error, and gives `--yes` scripts a conventional non-zero failure signal. It is testable through the required deterministic stdout/stderr and exit cases.
- **Rejected candidates:** A new cancellation code expands the public contract with no upstream need; treating cancellation as `2` conflates an intentional safe response with an operational failure.
- **Execution approval:** The user's instruction to use FPF to decide the gate, together with the prior end-to-end delivery authorization, approves this feature-local public-contract decision and the bounded implementation. Production/live-release actions remain out of scope.

## Review-Improve Record

### Cycle 1 — 2026-07-22

- **Review scope:** `FT-012` bootstrap package, routing evidence, issue #12, current root README, `scripts/install.sh`, release workflow and relevant Memory Bank owners.
- **Critical findings:** none.
- **Important finding:** I-01 — the issue explicitly demands public exit-code/stdout/stderr contracts, but the supplied requirements do not decide cancellation/no-change semantics or output routing.
- **Minor findings:** none recorded; no minor changes made.
- **Action:** Created the bootstrap package and recorded the gate.

### Cycle 2 — 2026-07-22

- **Review scope:** `D-01` alternatives against issue #12 and the existing public exit taxonomy.
- **Critical/important findings:** none after accepting candidate 1.
- **Action:** Promoted the resolved exit/output contract into `brief.md`; created downstream solution and plan artifacts.

### Delivery Review — 2026-07-22

- **Implementation review:** deterministic updater tests, race tests, full repository verification and release-artifact smoke are passing. A Codex review against `master` produced no critical/high finding.
- **CI:** required `verify` check passed for PR #18.
- **Result:** no open critical/high implementation or document finding remains.
