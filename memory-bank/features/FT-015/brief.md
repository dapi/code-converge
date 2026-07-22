---
title: "FT-015: Structured Codex review responses and no-change runs"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem/verify owner for accepting validated structured Codex review reports and completing clean no-change worktrees safely."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../product/context.md
  - ../../domain/glossary.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/15
status: active
delivery_status: in_progress
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-015: Structured Codex review responses and no-change runs

## What

### Problem

`code-converge` currently rejects the ordinary JSON response emitted by `codex review` when its `findings` array is empty. The parser only recognizes priority-labelled Markdown findings and a narrow clean-prose allowlist, so a completed clean review becomes an operational failure. A clean review of a worktree with no staged, unstaged or untracked changes would also enter finalization, where an empty commit must not be attempted.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Structured-review classification | Valid JSON clean report from Codex CLI 0.145.0 fails classification | Valid complete structured clean and prioritized-findings fixtures classify deterministically | Adapter fixture tests |
| `MET-02` | Fail-closed behavior | Ambiguous output fails | Malformed, incomplete, duplicate-key or unrecognized structured output still causes operational failure | Negative parser/workflow fixtures |
| `MET-03` | No-change finalization safety | Clean no-change run reaches finalization | A clean worktree exits successfully after review with no finalize invocation or commit attempt | Workflow integration test with fake repository status |

### Scope

- `REQ-01` Classify a strictly validated JSON review report with an empty `findings` array as a clean review.
- `REQ-02` Classify a strictly validated JSON review report containing prioritized findings into the existing severity counters and preserve its report for fix-findings.
- `REQ-03` Preserve existing plain-text clean/finding classification, counters and fail-closed behavior for malformed, incomplete, duplicate-key or unrecognized structured output.
- `REQ-04` After a clean review, detect whether staged, unstaged or untracked changes exist; if none exist, complete the run successfully without entering finalization or attempting a commit.
- `REQ-05` Update the root CLI contract and dependent Memory Bank owners together with the implementation, and retain evidence required by this feature.

### Non-Scope

- `NS-01` Changing the Codex review command, model selection, prompts or asking Codex to emit a caller-supplied schema.
- `NS-02` Accepting arbitrary JSON, partially valid JSON, unknown fields, unknown priorities or free-form prose as a clean verdict.
- `NS-03` Changing finding severity names, stdout counter fields, review/fix budgets, finalization verdict meanings or non-clean workflow behavior.
- `NS-04` Treating ignored files as worktree changes, creating an empty commit, or publishing a change request for a no-change run.

### Constraints / Assumptions

- `ASM-01` Issue #15 supplies a clean JSON sample from `codex-cli 0.145.0`; local CLI evidence and a recorded structured review show the same top-level fields and prioritized finding shape.
- `ASM-02` The root README is the public CLI/log contract; source code owns package-level implementation details.
- `CON-01` The existing safety policy treats ambiguous review output as operational failure; structured support must not weaken that boundary.
- `CON-02` `codex review --uncommitted` covers staged, unstaged and untracked changes; no-change detection must use the same three categories named by the issue.
- `CON-03` This is one vertical workflow delivery-unit; it must not add a reusable architecture policy or change unrelated finalization behavior.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The feature changes review-output and terminal workflow contracts, introduces a strict external-data parser and adds a conditional state transition before finalization. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | FPF closure and review-improve provenance are required for parser and no-change decisions. | feature-local provenance only |
| Separate contract, C4 or sequence artifact | omitted | The C3 boundary and compact strict JSON contract fit in `design.md`; a separate artifact would duplicate ownership. | `design.md` |
| Feature-local use case | omitted | The feature corrects the existing review workflow and does not add a distinct stable project scenario. | existing workflow owners |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `standard` | The public review-result and terminal workflow behavior changes, including an external structured-output boundary. There is no security, persistent-data, concurrency, migration, or cross-system trigger requiring `high-risk`; documentation-only/low-risk are insufficient. | `none` |

## Verify

### Exit Criteria

- `EC-01` Valid structured clean output produces the existing clean review record with all zero-filled counters; valid prioritized structured findings produce the existing counters and proceed through fix-findings.
- `EC-02` Malformed, incomplete, duplicate-key, unknown-field/type/priority, mixed or otherwise unrecognized structured output remains an operational failure and emits no unreliable counters.
- `EC-03` Existing plain-text clean/finding fixtures and severity counters remain valid.
- `EC-04` A clean review of a worktree without staged, unstaged or untracked files ends successfully without a finalize stage, a finalize agent invocation or an empty commit attempt.
- `EC-05` README and dependent canonical Memory Bank documents describe the delivered contract consistently; all selected validation passes.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `ASM-01`, `CON-01` | `EC-01`, `SC-01` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `REQ-02` | `ASM-01`, `CON-01` | `EC-01`, `SC-02` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `REQ-03` | `CON-01` | `EC-02`, `EC-03`, `SC-03` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `REQ-04` | `CON-02` | `EC-04`, `SC-04` | `CHK-02`, `CHK-03` | `EVID-02`, `EVID-03` |
| `REQ-05` | `ASM-02`, `CON-03` | `EC-05`, `SC-05` | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` |

### Acceptance Scenarios

- `SC-01` Codex returns the validated JSON clean-report shape shown in issue #15; review emits `status=clean` and all existing zero-valued finding counters.
- `SC-02` Codex returns a validated JSON report with prioritized findings; the counters map priorities to the existing severity buckets and the original normalized report is supplied to fix-findings.
- `SC-03` A malformed, incomplete, duplicate-key, unknown-field/type/priority or mixed structured report emits failed review completion and terminal operational failure without finding counters.
- `SC-04` A clean review is followed by a repository-status result with no staged, unstaged or untracked changes; the run emits successful completion without `stage_started stage=finalize` or finalize side effects.
- `SC-05` A clean review with changes retains the existing finalization path; README, domain rules/states and feature evidence converge on both paths.

### Negative Coverage

- `NEG-01` JSON with trailing data, duplicate keys, missing required fields, wrong scalar/array/object types, unknown top-level/finding/location fields, or invalid priority is rejected.
- `NEG-02` Plain-text clean/finding fixtures remain classified under their existing rules; arbitrary prose and findings mixed with a clean verdict remain rejected.
- `NEG-03` A repository-status command failure is operational failure, not a no-change success; ignored files alone do not suppress a no-change outcome.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `EC-01`, `EC-02`, `EC-03`, `NEG-01`, `NEG-02` | `go test ./internal/codex` | Strict JSON and existing plain-text parser fixtures pass. | `artifacts/ft-015/verify/chk-01/` |
| `CHK-02` | `EC-01`, `EC-02`, `EC-04`, `NEG-03` | `go test ./internal/workflow ./internal/app` | Review transitions, status failure and no-change/no-finalize behavior pass with fakes. | `artifacts/ft-015/verify/chk-02/` |
| `CHK-03` | `EC-01`, `EC-04`, `SC-01`, `SC-04` | Fake-executable app integration tests | Captured Codex JSON and deterministic repository status produce the documented event sequence. | `artifacts/ft-015/verify/chk-03/` |
| `CHK-04` | `EC-05` | `make docs-lint` and semantic contract read-through | README, domain owners and feature docs agree. | `artifacts/ft-015/verify/chk-04/` |
| `CHK-05` | `EC-05` | `go test ./... && go vet ./... && git diff --check` | Full local verification is green. | `artifacts/ft-015/verify/chk-05/` |

### Test matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | `artifacts/ft-015/verify/chk-01/` |
| `CHK-02` | `EVID-02` | `artifacts/ft-015/verify/chk-02/` |
| `CHK-03` | `EVID-03` | `artifacts/ft-015/verify/chk-03/` |
| `CHK-04` | `EVID-04` | `artifacts/ft-015/verify/chk-04/` |
| `CHK-05` | `EVID-05` | `artifacts/ft-015/verify/chk-05/` |

### Evidence

- `EVID-01` Parser test log covering clean, findings and rejected structured fixtures.
- `EVID-02` Workflow/app test log covering status, no-change and finalization decisions.
- `EVID-03` Fake-executable end-to-end event/output test log.
- `EVID-04` Documentation lint output and reviewed public-contract diff.
- `EVID-05` Full local verification log and required CI reference at delivery closure.

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | Parser test log | test runner | `artifacts/ft-015/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Workflow/app test log | test runner | `artifacts/ft-015/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Fake-executable integration log | test runner | `artifacts/ft-015/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Docs lint and semantic review record | test runner/reviewer | `artifacts/ft-015/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Full verification and CI references | test runner/CI | `artifacts/ft-015/verify/chk-05/` | `CHK-05` |
