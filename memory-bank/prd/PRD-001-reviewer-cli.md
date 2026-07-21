---
title: "PRD-001: Reviewer CLI"
doc_kind: prd
doc_function: canonical
purpose: "Фиксирует продуктовую проблему, пользователей, goals, scope и success metrics законченной локальной reviewer CLI."
derived_from:
  - ../product/context.md
  - ../product/vision.md
  - ../product/customers.md
  - ../product/metrics.md
  - ../domain/rules.md
  - ../domain/states.md
  - ../../README.md
status: active
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - architecture_decision
  - feature_level_verify_contract
canonical_for:
  - reviewer_cli_product_initiative
---

# PRD-001: Reviewer CLI

## Problem

Developers using coding agents must repeatedly coordinate review, finding fixes, publication, and applicable CI by hand. Each transition depends on interpreting agent prose, so a successful subprocess can be mistaken for a completed outcome while findings, publication failures, or red CI remain unresolved.

The project needs one bounded local workflow that drives this loop to an explicit terminal result and exposes enough progress data for an operator to understand what happened. Project-wide context remains owned by [`../product/context.md`](../product/context.md); the exact public CLI contract remains owned by [`../../README.md`](../../README.md).

## Users And Jobs

| User / Segment | Job To Be Done | Current Pain |
| --- | --- | --- |
| `SEG-01` developers or teams using coding agents in Git repositories | Move an agent-assisted change from review through fixes and publication, with green required CI when it applies | The operator manually coordinates multiple tools and must infer stage outcomes from unstructured agent responses |
| `ACT-01` CLI operator | Configure and run the workflow, observe progress, and act on its terminal result | Configuration sources, remaining findings, CI state, and partial publication failures are otherwise easy to misinterpret |

## Goals

- `G-01` Provide one local CLI workflow that performs review, bounded finding remediation, finalization, and bounded CI recovery in the order required by the domain state machine.
- `G-02` Produce an explicit terminal outcome that distinguishes success, remaining findings, operational failure, and CI-recovery failure.
- `G-03` Treat ambiguous or internally inconsistent agent output as failure rather than inferred success.
- `G-04` Make stage progress, review severity counts, durations, finalization steps, and the terminal result observable through the public one-line stdout contract.
- `G-05` Let operators inspect and override effective configuration without starting a workflow run.

## Non-Goals

- `NG-01` Replace Codex, Git, repository hosting, CI, or a task tracker.
- `NG-02` Support agents other than Codex.
- `NG-03` Provide hosted execution, a graphical interface, persistent metrics, dashboards, or cross-run analytics.
- `NG-04` Standardize one Git hosting provider or one CI implementation.
- `NG-05` Define pricing, enterprise rollout, or a broader go-to-market model.

## Product Scope

### In Scope

- Invoke the configured local Codex review command and safely classify its ordinary report as clean, findings, or failure.
- Normalize finding priorities into the public severity buckets and report complete counters for every classified review.
- Run bounded review/fix cycles, including the mandatory verification review after the final permitted fix.
- Finalize only after a clean review and interpret a constrained finalization result, including commit, push, change-request, and CI step outcomes.
- When publication succeeded but applicable required CI is red, run bounded CI recovery and restart review in a fresh review phase.
- Resolve settings from the documented CLI, project, user, environment, and built-in sources; expose them through `reviewer config`.
- Emit the documented stdout records, diagnostics on stderr, and the specified exit codes.
- Be buildable and distributable as a local Go CLI without requiring a Go runtime for a released binary.

### Out Of Scope

- Internal behavior or credential management of Codex, Git remotes, hosting providers, and CI systems.
- Automatic discovery or ownership of repository-specific policies beyond what the configured agent and target repository expose.
- Persistence or aggregation of run history outside the current process output.
- Product expansion beyond the documented CLI without separate evidence and an explicit product decision.

## UX / Business Rules

- `BR-01` Downstream delivery must preserve the workflow invariants and terminal outcomes owned by [`../domain/rules.md`](../domain/rules.md) and the transitions owned by [`../domain/states.md`](../domain/states.md).
- `BR-02` A clean review is necessary but not sufficient for run success; finalization must establish the documented successful terminal state.
- `BR-03` Unclassified review output, an unrecognized finalization verdict, or inconsistent finalization details must never be interpreted as success.
- `BR-04` Fix-findings and CI-recovery loops are independently bounded; exhausting either budget produces its specified non-zero terminal outcome.
- `BR-05` Operational stdout remains machine-readable and one-record-per-line; raw agent output and human-readable diagnostics do not contaminate that stream.
- `BR-06` An operator can inspect every effective setting and its source before execution.
- `BR-07` Observability is a cross-cutting acceptance requirement: every internal implementation checkpoint that adds or changes a workflow transition must include the corresponding stdout records, stderr behavior, counters, and durations before that checkpoint is considered complete.

## Success Metrics

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Terminal outcome contract coverage | No implementation evidence supplied | Every acceptance path ends with the specified exit code and `run_completed` record; no ambiguous result reaches exit `0` | Automated state-machine and adapter acceptance tests against the public contract |
| `MET-02` | Classified review observability | No implementation evidence supplied | Every clean/findings review reports total plus all severity buckets, with the total equal to their sum | Contract tests over clean, prioritized, unknown-priority, malformed, and failed review fixtures |
| `MET-03` | Stage and run timing coverage | No implementation evidence supplied | Every completed stage reports `duration_ms`; every terminal run reports `total_duration_ms` | Event-stream tests for every terminal path |
| `MET-04` | Configuration explainability | No implementation evidence supplied | Every effective option is shown with its winning source, and non-default values also show the built-in default | Precedence-matrix tests and `reviewer config` golden output |
| `MET-05` | Complete workflow delivery | Not measured; implementation proof is unavailable | The documented happy path, findings path, exhausted-findings path, operational-failure paths, CI-recovery path, and exhausted-CI path all have reproducible acceptance evidence | Delivery evidence plus a final end-to-end utility acceptance run |

These are contract-conformance targets for the complete utility. Adoption, time savings, satisfaction, and reliability in real repositories remain unmeasured product outcomes and must not be inferred from passing tests.

## Risks And Open Questions

- `RISK-01` Ordinary Codex review prose may change or contain ambiguous language; unsafe classification could produce a false clean result.
- `RISK-02` Finalization delegates material Git, hosting, and CI actions to an agent; a process exit alone does not prove that required external outcomes occurred.
- `RISK-03` The default models are not available to every Codex account, which can block first-run success unless configuration and diagnostics are clear.
- `RISK-04` Repeated review and CI-recovery loops can consume substantial time and model budget even when correctly bounded.
- `RISK-05` The intended user pain and adoption hypothesis are specified but not validated by interviews or usage analytics.
- `RISK-06` Resolved by FT-002 (artifact matrix/platform baseline) and FT-003 (official GitHub Release process); signing and package-manager distribution remain outside the supported policy.
- `OQ-01` Which representative repositories and failure fixtures form the minimum credible acceptance corpus?
- `OQ-02` Who owns final product acceptance?
- `OQ-03` Resolved by FT-002 and FT-003: macOS/Linux on AMD64/ARM64, GitHub Release archives, and manual installation to an operator-owned `PATH` directory.

## Downstream Delivery

The complete product is small enough to be delivered as one coherent delivery-unit through a single Feature Flow package. An Epic and separate feature packages for review, finalization, CI recovery, configuration, observability, or packaging would add coordination without creating independently useful product outcomes.

Observability remains a cross-cutting acceptance requirement: every internal checkpoint that adds or changes a workflow transition must include its corresponding stdout records, stderr behavior, counters, and durations. The feature package must finish with contract-convergence verification across the complete run.

| Delivery unit | Included outcome | Status |
| --- | --- | --- |
| Reviewer CLI complete delivery | Review/fix convergence, finalization, CI recovery, configuration inspection, operational records, terminal outcomes, reproducible binary, and distribution evidence | planned |
