---
title: "FT-022: Structured Codex review result channel"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem/verify owner for schema-valid Codex review results that are classified independently of terminal stdout and stderr."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../product/context.md
  - ../../prd/PRD-001-code-converge-cli.md
  - ../../engineering/architecture.md
  - ../FT-015/brief.md
  - ../FT-015/design.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/22
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-022: Structured Codex review result channel

## What

### Problem

`code-converge` classifies only the stdout returned by its Codex review invocation, while Codex CLI 0.145.0 can emit the review stream and final prose to stderr and leave stdout empty. Even a substantively clean free-form final message may not match the fail-closed plain-text allowlist. Issue #22 records this false operational failure on FT-4830 despite a clean review conclusion and a targeted test run with 22 examples and 0 failures.

The review outcome therefore depends on terminal channel behavior and prose wording instead of a stable final-response contract. Clean reviews, reviews with findings and malformed or failed invocations are not distinguished at the integration boundary before workflow classification.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Clean-review classification | A clean Codex conclusion can become exit `2` when stdout is empty or final prose misses the allowlist | A schema-valid result with `findings: []` always classifies as clean | Adapter and fake-executable tests |
| `MET-02` | Result-channel determinism | Classification depends on terminal stdout content while review data may be on stderr | Classification reads one schema-constrained final-response carrier and never terminal stdout/stderr | Channel-conflict fixtures |
| `MET-03` | Fail-closed diagnostics | Empty or unclassifiable stdout produces a generic parsing failure | Missing, empty, malformed, incomplete, schema-invalid or failed final output produces exit `2` with a failure-specific diagnostic | Adapter/workflow negative matrix |

### Scope

- `REQ-01` Make every Codex review produce one schema-constrained final response whose top-level fields and finding fields match the existing strict structured-review contract established by FT-015.
- `REQ-02` Classify an empty `findings` array as clean and classify findings with their existing priority, code location, confidence, title and body data into the existing result/counter contract.
- `REQ-03` Read review classification and the report supplied to fix-findings only from the final-response carrier; do not classify terminal stdout or stderr and do not merge either stream into the report.
- `REQ-04` Preserve the resolved review base, computed merge base and private branch-and-worktree snapshot as inputs available to the reviewing agent through the scoped Git transport, without changing the existing merge-base-to-worktree review scope or exporting the private index to unrelated commands.
- `REQ-05` Treat a missing, empty, malformed, incomplete or schema-invalid final response, and any non-zero Codex invocation, as an operational failure with exit `2` and a useful diagnostic.
- `REQ-06` Preserve the public workflow event fields, finding severity/count semantics, review/fix budgets, terminal statuses and exit-code meanings.
- `REQ-07` Add deterministic coverage for clean output, findings, malformed/incomplete/schema-invalid output, missing final-output file, non-zero Codex exit, conflicting terminal streams and preserved review-target inputs.
- `REQ-08` Update the root public CLI contract and directly dependent active Memory Bank documents together with implementation; `make docs-lint` must pass.

### Non-Scope

- `NS-01` Diagnostic session logging, retention, redaction or opt-out behavior from issue #14.
- `NS-02` Changing base discovery order, private-index construction, review scope, model/reasoning resolution or workflow cycle budgets.
- `NS-03` Changing the public stdout event schema, human rendering, severity names/mapping, finalization verdicts or exit-code meanings.
- `NS-04` Treating arbitrary prose, legacy terminal output or a merged stdout/stderr stream as a valid clean review result.
- `NS-05` Adding Codex timeout, retry, sandbox, approval, network, authentication or session-persistence policy.
- `NS-06` Defining an unsupported minimum Codex semantic version when the available evidence establishes capabilities rather than their first release.

### Constraints / Assumptions

- `ASM-01` Issue #22 is the source of the observed FT-4830 failure, desired final-response contract, proposed `codex exec` direction and acceptance inventory.
- `ASM-02` Local `codex-cli 0.145.0` help and the current official Codex manual both expose `codex exec --output-schema` and `--output-last-message`; the manual states that `codex exec` progress goes to stderr and the final message can be written to the named file.
- `ASM-03` FT-015 and the current parser establish the exact structured review object and `P0`–`P3` numeric priority mapping; this feature changes the carrier and invocation protocol, not the finding meaning.
- `ASM-04` The root [`README.md`](../../../README.md) owns the public CLI, stdout and exit-code contract; active engineering/PRD documents that describe the old invocation are downstream current-state owners to update atomically with implementation.
- `ASM-05` Codex shell environment policy can filter inherited variables before agent tool commands; its invocation-local `shell_environment_policy.set` values override that filter.
- `CON-01` Unknown, arbitrary or internally invalid output cannot produce a clean result, counters, a fix prompt or finalization.
- `CON-02` The selected base commit and private index snapshot remain the canonical review target prepared by `internal/repository`; the Codex adapter must not reconstruct a different target.
- `CON-03` Terminal stderr remains diagnostic data and terminal stdout remains runner output; neither is review-result data or workflow stdout.
- `CON-04` Historical FT-015 delivery documents remain historical evidence and are not rewritten as current FT-022 execution artifacts.
- `CON-05` Private-index availability must not depend on the operator's unrelated Codex shell-environment whitelist/exclude settings, and the index must not leak to absolute Git paths or commands targeting another repository.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The feature materially changes the external Codex process protocol, its data carrier and failure semantics while preserving public workflow contracts. It requires an explicit schema, trust-boundary analysis, C1 view and rollout/backout decision. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | The issue/flow routing conflict, high-risk assurance floor and result-channel decisions require auditable FPF provenance. | Feature-local provenance only |
| `feature-review-report.md` | selected | The user requires bounded review-improve cycles and a durable final report. | Review findings/status only |
| Separate interaction contract or sequence file | omitted | The one synchronous local-process connector, strict JSON schema and ordering fit within `design.md` without creating a second owner. | `design.md` |
| Feature-local use case | omitted | The feature repairs the existing review stage and does not introduce a new stable project-level scenario. | Existing PRD/domain workflow owners |
| ADR | omitted | The selected protocol is feature-local and follows issue #22 plus the established adapter/runner boundaries; it does not define a reusable cross-feature architecture policy. | `design.md` |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `high-risk` | The change materially replaces the protocol and failure semantics of the existing external Codex process integration. It crosses the process trust boundary and controls whether findings can be interpreted as clean. The profile therefore requires complete boundary/failure coverage, explicit execution approval, backout and independent convergence. | `none`; no downgrade |

Execution was approved explicitly in the 2026-07-23 delivery turn and is recorded as `DL-09` / `AG-01` in `decision-log.md` and `implementation-plan.md`.

## Verify

### Exit Criteria

- `EC-01` A schema-valid final response with `findings: []` produces the existing clean review outcome and all zero-filled severity counters.
- `EC-02` A schema-valid final response with `P0`–`P3` findings preserves every required finding field, maps priorities to the existing counters and supplies the same structured report to fix-findings.
- `EC-03` Review classification is identical regardless of terminal stdout/stderr content when the final-response file is identical; neither terminal stream appears in workflow stdout or the fix-findings report.
- `EC-04` The reviewing agent receives the pinned selected base and computed merge base, reaches the current private snapshot through scoped Git, and compares that snapshot from the merge base without exposing the private index to unrelated commands.
- `EC-05` Missing, empty, malformed, incomplete, unknown-field/type/priority or otherwise schema-invalid final output, plus a non-zero invocation, yields a useful diagnostic and the existing operational-failure path with no unreliable counters.
- `EC-06` Public event records, finding metrics, cycle transitions and exit codes remain byte/semantic compatible; directly dependent current-state documentation describes the delivered protocol consistently.
- `EC-07` The high-risk validation contract, local/full suites, independent convergence and required CI are complete before delivery closure.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `ASM-01`–`ASM-03`, `CON-01` | `EC-01`, `EC-02`, `SC-01`, `SC-02` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `REQ-02` | `ASM-03`, `CON-01` | `EC-01`, `EC-02`, `SC-01`, `SC-02` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `REQ-03` | `CON-01`, `CON-03` | `EC-03`, `SC-03`, `NEG-01`, `NEG-02` | `CHK-01`, `CHK-02`, `CHK-03` | `EVID-01`–`EVID-03` |
| `REQ-04` | `ASM-05`, `CON-02`, `CON-05` | `EC-04`, `SC-04` | `CHK-02`, `CHK-03` | `EVID-02`, `EVID-03` |
| `REQ-05` | `CON-01` | `EC-05`, `SC-05`, `NEG-03`, `NEG-04` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `REQ-06` | `ASM-04`, `CON-03` | `EC-06`, `SC-06` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` |
| `REQ-07` | `ASM-01`, `CON-01`–`CON-03` | `EC-01`–`EC-05`, `SC-01`–`SC-05` | `CHK-01`–`CHK-03` | `EVID-01`–`EVID-03` |
| `REQ-08` | `ASM-04`, `CON-04` | `EC-06`, `EC-07`, `SC-07` | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` |

### Acceptance Scenarios

- `SC-01` Codex exits successfully and writes a schema-valid object with `findings: []` to the final-response file; review emits the existing clean completion and zero counters.
- `SC-02` Codex exits successfully and writes schema-valid findings covering numeric priorities `0`–`3`; the existing counters and fix-findings handoff preserve the structured result.
- `SC-03` The same valid final-response file is paired with arbitrary or conflicting stdout/stderr; classification uses only the file and workflow stdout contains no raw Codex stream.
- `SC-04` The invocation receives the selected base commit and computed merge base in its review instruction, compares from that merge base through the prepared wrapper-first `PATH`, and does not receive `GIT_INDEX_FILE`.
- `SC-05` Each required invalid/missing final-output case and a non-zero Codex exit produces the operational-failure event/exit path without finding counters.
- `SC-06` Clean, findings, exhausted-findings and operational-failure workflow paths retain their existing public event fields, budgets and exit codes.
- `SC-07` Root README, active PRD/engineering/testing owners and FT-022 documents converge on the delivered final-response protocol while FT-015 remains historical evidence.

### Negative Coverage

- `NEG-01` Valid-looking JSON or allowlisted clean prose on stdout cannot compensate for a missing, empty or invalid final-response file.
- `NEG-02` Valid-looking JSON on stderr cannot become review data, cannot enter workflow stdout and cannot compensate for an invalid final-response file.
- `NEG-03` The final-response object is rejected for duplicate, missing, case-variant or unknown fields; wrong types; trailing data; invalid priority; or incomplete nested location data.
- `NEG-04` A final-response file is not trusted after a non-zero Codex exit, even if its contents are otherwise schema-valid.
- `NEG-05` An installed Codex CLI without the required `exec` capabilities fails through the existing non-zero invocation/exit-2 path; the adapter does not fall back to terminal parsing.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `EC-01`–`EC-03`, `EC-05`, `NEG-01`–`NEG-04` | `go test ./internal/codex` with table-driven schema, file-channel and invocation fixtures | Strict clean/findings results pass; every invalid or wrong-channel variant fails closed with a specific error. | `artifacts/ft-022/verify/chk-01/` |
| `CHK-02` | `EC-03`, `EC-04`, `SC-03`, `SC-04` | Fake-executable adapter tests for args, stdin, cwd/env, stdout/stderr and output-file behavior | Invocation uses the pinned base and scoped private snapshot without exporting its index, and classifies only the named final-response file. | `artifacts/ft-022/verify/chk-02/` |
| `CHK-03` | `EC-01`–`EC-06`, `SC-01`–`SC-06` | `go test ./internal/workflow ./internal/app` with updated fake runner/executable plus affected repository tests | Existing workflow records, counters, transitions and exit codes remain unchanged across clean/findings/failure paths. | `artifacts/ft-022/verify/chk-03/` |
| `CHK-04` | `EC-06`, `SC-07` | `make docs-lint` plus semantic read-through of README and directly dependent active owners | Current public/project docs agree on the delivered protocol; historical FT-015 evidence remains unchanged. | `artifacts/ft-022/verify/chk-04/` |
| `CHK-05` | `EC-07` | `go test ./... && go vet ./... && make docs-lint && git diff --check`, required CI and independent code-converge review | Full high-risk validation is green and no critical/high review finding remains. | `artifacts/ft-022/verify/chk-05/` |

### Test matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | `artifacts/ft-022/verify/chk-01/` |
| `CHK-02` | `EVID-02` | `artifacts/ft-022/verify/chk-02/` |
| `CHK-03` | `EVID-03` | `artifacts/ft-022/verify/chk-03/` |
| `CHK-04` | `EVID-04` | `artifacts/ft-022/verify/chk-04/` |
| `CHK-05` | `EVID-05` | `artifacts/ft-022/verify/chk-05/` |

### Evidence

- `EVID-01` Strict parser/final-response channel test log covering clean, findings and the complete rejected matrix.
- `EVID-02` Fake-executable invocation log covering schema/output paths, prompt, selected base, private index and stream exclusion.
- `EVID-03` Workflow/app regression log covering existing records, counters, budgets, transitions and exit codes.
- `EVID-04` Documentation lint output and semantic current-state contract review.
- `EVID-05` Full local verification, explicit high-risk approval, independent review and required CI references.

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | Adapter/parser test log | test runner | `artifacts/ft-022/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Fake-executable boundary log | test runner | `artifacts/ft-022/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Workflow/app regression log | test runner | `artifacts/ft-022/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Docs lint and semantic review record | test runner/reviewer | `artifacts/ft-022/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Approval, full validation, independent review and CI references | approver/test runner/reviewer/CI | `artifacts/ft-022/verify/chk-05/` | `CHK-05` |

### Execution Evidence Status

| Evidence ID | Status | Concrete carrier |
| --- | --- | --- |
| `EVID-01` | pass | `go test ./internal/codex`: strict clean/P0–P3 fixtures, missing/empty/prose/malformed/incomplete/unknown/invalid-priority cases, duplicate/case/trailing parser matrix and non-zero invocation all pass. |
| `EVID-02` | pass | `TestReviewFakeExecutableBoundary`, scoped-review argument tests, repository wrapper isolation tests, invocation/target assertions and stream-conflict fixtures prove schema/final paths, model/effort, stdin, cwd, wrapper-first `PATH`, absence of Codex-visible `GIT_INDEX_FILE`, merge-base comparison, sole-carrier behavior and cleanup. |
| `EVID-03` | pass | `go test ./internal/repository ./internal/workflow ./internal/app` passes; app fixtures preserve clean/no-change/finalization and exit-2 behavior while excluding conflicting terminal streams. |
| `EVID-04` | pass | Root README and dependent architecture/testing/PRD/domain owners were reviewed together; `make docs-lint` and `git diff --check` pass. |
| `EVID-05` | pass | `DL-09` records `AG-01`/`AG-02`; local `go test ./...`, `go vet ./...`, `go test -race ./internal/codex ./internal/app`, docs lint and diff check pass. Independent Codex review session `019f8dd7-3ff5-7922-b328-059b6a6e3120` reports no findings at confidence `0.87`. PR [#23](https://github.com/dapi/code-converge/pull/23) is mergeable and required [Verify run](https://github.com/dapi/code-converge/actions/runs/29988067825) passed. |
