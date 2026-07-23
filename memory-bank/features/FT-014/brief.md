---
title: "FT-014: Diagnostic Session Logs With Retention And Opt-Out"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem space для diagnostic session logs: scope, constraints, validation profile и verify contract без выбора solution или execution sequence."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../../engineering/testing-policy.md
  - ../../../README.md
  - ../../../internal/config/config.go
  - ../../../internal/runner/runner.go
  - https://github.com/dapi/code-converge/issues/14
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - selected_solution
  - implementation_sequence
---

# FT-014: Diagnostic Session Logs With Retention And Opt-Out

## What

### Problem

When a Codex review cannot be classified, workflow diagnostics expose the classification error but do not retain raw Codex terminal streams. Those streams are deliberately excluded from workflow stdout, so an operator cannot reconstruct a failed session afterwards.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Discoverable diagnostic record for a normal workflow session | No retained raw invocation record | One configured-location session record that covers every Codex invocation | Deterministic workflow test inspects the configured diagnostic directory |

### Scope

- `REQ-01` Persist one diagnostic session record by default under the configured default location for every `code-converge` workflow session.
- `REQ-02` For every Codex invocation, record its executable, fully resolved arguments, applicable stdin, stdout, stderr, exit/error outcome, stage, review phase, cycle, resolved model, reasoning effort, timestamps and duration.
- `REQ-03` Configure session-log location and retention through the established CLI, project, user and environment configuration model; show effective values and sources through `code-converge config`.
- `REQ-04` Retain diagnostic records for 24 hours by default and best-effort remove records older than the effective retention period without touching paths outside the configured session-log directory.
- `REQ-05` Provide a per-run CLI opt-out that creates no diagnostic session-log artifacts.
- `REQ-06` Preserve the stable workflow stdout contract: raw Codex stdout/stderr remain absent from the human and `kv` workflow event stream.
- `REQ-07` Document sensitive-data exposure, owner-only permissions where the platform permits, redaction rules and retention behavior.
- `REQ-08` In human format only, after an enabled run creates its session record, emit exactly one initial permanent local-time line `HH:MM:SS Session log: <canonical path>`; emit no such line when opt-out is active and never include record contents in it.

### Non-Scope

- `NS-01` Changing Issue #9 human-progress/liveness behavior or placing raw Codex output in workflow stdout.
- `NS-02` Logging authentication tokens or process environment values.
- `NS-03` Any retention cleanup outside the effective session-log directory.
- `NS-04` A background log-processing service, remote log upload, or a durable logging requirement beyond a workflow session record.

### Constraints / Assumptions

- `ASM-01` The root `README.md` is canonical for public CLI, configuration names/defaults, stdout stream and exit-code contracts; this feature must update it atomically with implementation.
- `ASM-02` `internal/config` already resolves settings with CLI > project > user > environment > built-in precedence and exposes effective values/sources through `code-converge config`.
- `ASM-03` The external-process runner already captures Codex stdin/stdout/stderr and invocation failure at the process boundary; raw terminal streams are not workflow result data and are not forwarded to workflow stdout.
- `CON-01` A cleanup failure is diagnostic only: it must not discard an otherwise useful session record or change an otherwise successful workflow outcome.
- `CON-02` The per-run opt-out must prevent creation of session-log artifacts for that run.
- `CON-03` The feature changes a public CLI/configuration contract and persists potentially sensitive repository and agent data; it requires explicit design reasoning before an implementation plan.
- `DEC-01` Accepted in `decision-log.md#d-01`: `session-log-dir`, `session-log-retention` and `--no-session-log` are the public names; zero retention is invalid; record-write failure is warning-only; and private per-session directories use deterministic redaction and non-following cleanup.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | Public CLI/configuration contract, persistent sensitive diagnostics, filesystem permissions/cleanup, concurrent sessions, redaction and failure semantics require a selected solution. | `design.md`, after `D-01` is accepted |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- |
| `decision-log.md` | selected | Records Feature Flow routing, FPF analysis, review-improve provenance and accepted decisions. | Feature-local record; accepted facts are promoted to canonical owners. |
| `design.md`, `implementation-plan.md` | selected | The accepted public/configuration/filesystem/security design needs canonical solution ownership and an executable high-risk plan. | `design.md` / `implementation-plan.md` |
| Separate C4, contract or sequence artifact | omitted | C2 topology, connector semantics and ordering fit in `design.md`; separate files would duplicate canonical facts. | `design.md` |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `high-risk` | The feature introduces durable handling of potentially sensitive repository/prompts/agent output, owner-only permissions and redaction controls, plus public CLI/configuration and persistent-data retention behavior. | `none`; no downgrade requested. Human approval for risk-bearing execution remains required by the profile. |

## Verify

### Exit Criteria

- `EC-01` A normal workflow creates exactly one discoverable session record in the effective diagnostic location, covering every Codex invocation with all `REQ-02` context and captured streams/outcome.
- `EC-02` Effective location and retention resolve with the established precedence and are displayed by `code-converge config`.
- `EC-03` Default retention removes only records older than 24 hours within the effective session-log directory; cleanup failures are diagnostic and preserve a useful current record.
- `EC-04` The per-run opt-out creates no session-log artifact.
- `EC-05` Workflow stdout remains the documented event stream and never includes raw Codex output; sensitive-data handling, permissions, redaction and retention are documented.
- `EC-06` An enabled human-format run emits the single required path-only handoff after record creation; `kv` and opt-out runs do not emit it.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01`, `REQ-02` | `ASM-03`, `CON-03`, `DEC-01` | `EC-01` | `CHK-01` | `EVID-01` |
| `REQ-03` | `ASM-01`, `ASM-02`, `DEC-01` | `EC-02` | `CHK-02` | `EVID-02` |
| `REQ-04` | `CON-01`, `DEC-01` | `EC-03` | `CHK-03` | `EVID-03` |
| `REQ-05` | `CON-02`, `DEC-01` | `EC-04` | `CHK-04` | `EVID-04` |
| `REQ-06`, `REQ-07` | `ASM-01`, `ASM-03`, `CON-03`, `DEC-01` | `EC-05` | `CHK-05` | `EVID-05` |
| `REQ-08` | `ASM-01`, `DEC-01` | `EC-06`, `SC-06` | `CHK-05` | `EVID-05` |

### Acceptance Scenarios

- `SC-01` A normal workflow with default settings executes multiple Codex stages and leaves one discoverable session record that contains a contextually complete entry for each invocation.
- `SC-02` Conflicting CLI, project, user and environment values resolve by the existing precedence and `code-converge config` exposes the effective diagnostic settings and winning sources.
- `SC-03` Retention cleanup sees old and current records, removes only eligible records inside the effective directory, and reports a cleanup failure without removing the current useful record or failing the workflow solely for cleanup.
- `SC-04` A workflow invoked with the accepted no-session-log flag leaves no session-log directory or record artifact for that run.
- `SC-05` A failed or successful Codex invocation retains raw stdout/stderr privately for diagnosis while neither raw stream appears in workflow stdout.
- `SC-06` An enabled human-format run emits one initial permanent path handoff only after the record exists; an opt-out or `kv` run emits no path handoff.

### Negative Coverage

- `NEG-01` Authentication tokens and process environment values are never included in diagnostic records.
- `NEG-02` Cleanup cannot follow a path outside the configured session-log directory.
- `NEG-03` Concurrent workflows cannot collide or corrupt each other's session records.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `EC-01`, `SC-01`, `SC-05` | Deterministic fake-Codex workflow tests | One session record contains each invocation's required context, arguments, streams and outcome. | `artifacts/ft-014/verify/chk-01/` |
| `CHK-02` | `EC-02`, `SC-02` | Table-driven config/app tests covering every source and precedence conflict | Valid effective values and sources are printed by `config`; invalid values fail by accepted public contract. | `artifacts/ft-014/verify/chk-02/` |
| `CHK-03` | `EC-03`, `SC-03`, `NEG-02` | Deterministic filesystem/time tests at retention boundaries and cleanup failures | Only eligible in-directory records are removed; cleanup error is diagnostic and non-fatal. | `artifacts/ft-014/verify/chk-03/` |
| `CHK-04` | `EC-04`, `SC-04` | Deterministic CLI/workflow tests with opt-out | No session-log artifact is created. | `artifacts/ft-014/verify/chk-04/` |
| `CHK-05` | `EC-05`, `EC-06`, `NEG-01`, `NEG-03` | Stream-isolation, redaction/permission, concurrent-session and human-handoff tests; `make docs-lint`; full required repository checks | Raw streams stay out of workflow stdout; sensitive-data policy and path-only handoff are enforced/documented; concurrent records remain valid; checks pass. | `artifacts/ft-014/verify/chk-05/` |

### Evidence

- `EVID-01` Invocation-capture workflow test log and inspected session-record fixture.
- `EVID-02` Configuration precedence and `config` output test log.
- `EVID-03` Retention boundary and cleanup-failure test log.
- `EVID-04` Opt-out test log.
- `EVID-05` Stream-isolation/security/concurrency tests, documentation lint and full validation log.

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | Invocation-capture test output and fixture | Test runner | `artifacts/ft-014/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Config precedence test output | Test runner | `artifacts/ft-014/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Retention/cleanup test output | Test runner | `artifacts/ft-014/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Opt-out test output | Test runner | `artifacts/ft-014/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Security/stream/concurrency and documentation/full-validation output | Test runner/reviewer | `artifacts/ft-014/verify/chk-05/` | `CHK-05` |

### Execution Evidence Status

| Evidence ID | Status | Concrete carrier |
| --- | --- | --- |
| `EVID-01` | pass | Deterministic `internal/session` and `internal/app` tests inspect one ordered private invocation record with captured streams, outcome and stage context. |
| `EVID-02` | pass | `internal/config` table tests cover source precedence, absolute/tilde path handling, duration validation and `config` setting display. |
| `EVID-03` | pass | `internal/session` tests cover retention boundary, direct-child cleanup, symlink exclusion and non-fatal record-write diagnostics. |
| `EVID-04` | pass | `TestAppNoSessionLogCreatesNoArtifactsOrHandoff` proves the per-run opt-out creates neither directory nor human handoff. |
| `EVID-05` | pass | `go test ./...`, `go test -race ./internal/session ./internal/app ./internal/workflow`, `go vet ./...`, `make docs-lint`, `git diff --check`; required [Verify run](https://github.com/dapi/code-converge/actions/runs/30015452719) passed for PR [#25](https://github.com/dapi/code-converge/pull/25). |
