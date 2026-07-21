---
title: "FT-002: Complete Reviewer CLI"
doc_kind: feature
doc_function: canonical
purpose: "Canonical brief полной delivery-единицы reviewer CLI: problem space, scope, blockers, validation profile и verify contract без выбора solution или execution sequence."
derived_from:
  - ../../flows/feature.md
  - ../../prd/PRD-001-reviewer-cli.md
  - ../../../README.md
  - ../../domain/rules.md
  - ../../domain/states.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../ops/release.md
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
  - release_policy
canonical_for:
  - ft_002_problem_space
  - ft_002_validation_profile
  - ft_002_verify_contract
  - ft_002_blocker_state
---

# FT-002: Complete Reviewer CLI

## What

### Problem

Issue [#2](https://github.com/dapi/reviewer/issues/2) delivers the product defined by [`PRD-001`](../../prd/PRD-001-reviewer-cli.md) and the public [`README`](../../../README.md). The repository currently has specifications but no Go module or implementation. The delivery must turn the documented workflow into one distributable local CLI without inferring success from ambiguous agent output or contaminating the machine-readable stdout stream.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Public workflow contract coverage | No implementation evidence | Every documented terminal path produces its exact exit code and `run_completed` record | Deterministic state-machine/adapter acceptance suite |
| `MET-02` | Classified review consistency | No implementation evidence | Every classified review has all five severity counters and `findings_total` equals their sum | Plain-text fixture and event golden tests |
| `MET-03` | Configuration explainability | No implementation evidence | Every public setting reports effective value and winning source; overridden defaults remain inspectable | Full precedence matrix and `reviewer config` golden tests |
| `MET-04` | Distribution readiness | No release process or artifact matrix | Reproducible binary plus approved platform, installation, identity, and smoke evidence | Release checks defined after `DEC-01` is resolved |

### Scope

- `REQ-01` Provide the `reviewer` workflow and `reviewer config` command with the options, defaults, source precedence, prompt resolution, prerequisites, and error semantics owned by the root `README.md`.
- `REQ-02` Invoke Codex through a controlled process boundary and safely classify ordinary review output into clean, findings, or operational failure, including priority normalization and internally consistent counters.
- `REQ-03` Enforce bounded fix-findings phases, including the mandatory verification review after the final permitted fix.
- `REQ-04` Finalize only after a clean review; interpret only consistent `SUCCESS`, `CI_FAILED`, or `FAILED` results and expose all four finalization step outcomes.
- `REQ-05` Enforce bounded CI recovery; after each successful CI fix, restart a complete review phase with a fresh fix budget.
- `REQ-06` Emit the complete newline-terminated stdout event contract, keep raw Codex output out of workflow stdout, send human diagnostics to stderr, and preserve documented phase/cycle/duration semantics.
- `REQ-07` Produce deterministic automated coverage using fake runners/fake executables without live Codex, remote mutation, change-request creation, or hosted-CI waits.
- `REQ-08` Produce a reproducible distributable binary and evidence for the approved supported platforms, installation paths, artifact identity, and smoke procedure.

### Non-Scope

- `NS-01` Agents other than Codex; hosted execution, GUI, dashboards, persistent cross-run analytics, or replacement of Git, hosting, CI, and task tracking.
- `NS-02` Hosting-provider-specific product behavior, credential ownership, or automatic discovery of repository policy beyond the configured agent and target repository.
- `NS-03` Production/live-data release actions or an ongoing product roadmap beyond this utility.
- `NS-04` Unspecified public options, output fields, platform promises, release channels, or agent protocols invented during implementation.

### Constraints / Assumptions

- `ASM-01` The root `README.md` is the sole owner of the public CLI contract; domain and downstream feature artifacts interpret it but do not redefine it.
- `ASM-02` Codex, Git, repository hosting, and CI are external systems; deterministic verification substitutes fakes for live side effects.
- `CON-01` Ambiguous, malformed, incomplete, internally inconsistent, or non-zero agent results cannot establish a successful review or finalization.
- `CON-02` The workflow is sequential and both remediation budgets are non-negative and independently bounded.
- `CON-03` Release and smoke evidence cannot claim support beyond an explicitly approved first-release baseline.
- Accepted feature-local decisions that close the former `DEC-01`–`DEC-05` blockers are owned by `design.md` as `SD-01`–`SD-05`; their FPF provenance is recorded in `decision-log.md`.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The feature creates a CLI/config/file contract, a cross-system Codex process boundary, state-machine orchestration, failure semantics, and a release artifact. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | The requested review-improve process needs auditable FPF reasoning and a human-gate record without making it a second canonical owner. | Feature-support provenance; accepted solution decisions must be promoted to `design.md` or ADR. |
| `design.md` | selected | Required contract/integration/release decisions are now evidence-backed and accepted under the user's FPF delegation. | `design.md` |
| `implementation-plan.md` | selected | The feature is proceeding to execution against the active design. | `implementation-plan.md` |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `high-risk` composed with all `release-deployment` obligations | New cross-system Codex integration and trust boundary activate `high-risk`; build/release artifact activates `release-deployment`, so the composition rule selects `high-risk` and retains release/package/smoke/rollback obligations. | `none` (no downgrade); profile acceptance owner remains `DEC-04` |

## Verify

### Exit Criteria

- `EC-01` All documented workflow states, budgets, terminal paths, verdicts, exit codes, phase/cycle counters, and finalization steps conform to the root README.
- `EC-02` All review/finalization parsers fail closed on malformed, ambiguous, incomplete, inconsistent, and non-zero results.
- `EC-03` All public configuration values and source conflicts resolve by the documented precedence and are inspectable without starting a run.
- `EC-04` Operational stdout/stderr, counters, timestamps, durations, and raw-output isolation conform on every path.
- `EC-05` Deterministic affected unit/integration/contract/e2e suites, repository checks, and independent convergence review pass without live external side effects.
- `EC-06` Approved release artifacts are reproducible and have platform, installation, identity, smoke, stop, and backout evidence.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `ASM-01` | `EC-03`, `SC-01`, `NEG-01` | `CHK-01`, `CHK-04` | `EVID-01`, `EVID-04` |
| `REQ-02` | `ASM-02`, `CON-01` | `EC-02`, `SC-02`, `NEG-02` | `CHK-02`, `CHK-04` | `EVID-02`, `EVID-04` |
| `REQ-03` | `CON-02` | `EC-01`, `SC-03`, `NEG-03` | `CHK-01`, `CHK-04` | `EVID-01`, `EVID-04` |
| `REQ-04` | `CON-01` | `EC-01`, `EC-02`, `SC-04`, `NEG-04` | `CHK-01`, `CHK-02`, `CHK-04` | `EVID-01`, `EVID-02`, `EVID-04` |
| `REQ-05` | `CON-02` | `EC-01`, `SC-05`, `NEG-05` | `CHK-01`, `CHK-04` | `EVID-01`, `EVID-04` |
| `REQ-06` | `CON-01` | `EC-04`, `SC-06`, `NEG-06` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` |
| `REQ-07` | `ASM-02` | `EC-05`, `SC-07` | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` |
| `REQ-08` | `CON-03` | `EC-06`, `SC-08`, `NEG-07` | `CHK-05`, `CHK-06` | `EVID-05`, `EVID-06` |

### Acceptance Scenarios

- `SC-01` `reviewer config` resolves every setting across default/environment/user/project/CLI conflicts and reports the winning source and overridden built-in default.
- `SC-02` A normal Codex report is classified as clean or findings; all priorities normalize and all counters are present and arithmetically consistent.
- `SC-03` Findings consume only fix attempts; the last allowed fix is followed by a verification review, with remaining findings terminating at exit `1`.
- `SC-04` A clean review reaches finalization; consistent `SUCCESS`, `CI_FAILED`, and `FAILED` responses route to the documented next state and expose four step outcomes.
- `SC-05` Successful CI repair increments `review_phase`, resets `cycle`, and restarts complete review; failure/exhaustion terminates at exit `3`.
- `SC-06` Every workflow path emits exactly the required one-line records and durations while raw agent output is captured away from stdout.
- `SC-07` The complete workflow corpus runs deterministically with fake runners/executables and no external mutation.
- `SC-08` Each approved release target installs and passes the approved smoke procedure without a Go runtime.

### Negative Scenarios

- `NEG-01` Invalid numbers, missing explicit prompt paths, unreadable values, or a non-Git target fail as documented and never start a review.
- `NEG-02` Ambiguous/malformed/incomplete review reports, unknown report shapes, and non-zero Codex review exits produce operational failure without counters or false-clean classification.
- `NEG-03` `max-cycles=0` performs no fix; findings terminate at exit `1` after the initial review.
- `NEG-04` Unknown or inconsistent finalization verdict/details cannot reach exit `0`; invocation/parsing failure omits a verdict and uses failed completion semantics.
- `NEG-05` CI-fix command failure or exhausted recovery budget exits `3` without silently reclassifying the outcome.
- `NEG-06` Whitespace, `=`, newlines, raw agent text, or secrets never enter encoded stdout values.
- `NEG-07` Unsupported or unverifiable release targets are not advertised as supported.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | State transitions, budgets, exits, configuration | Run deterministic table-driven unit/integration suites | All positive, boundary, and terminal cases pass | `artifacts/ft-002/verify/chk-01/` |
| `CHK-02` | Codex process and parser boundaries | Run fake-executable tests against the approved report/finalization corpus | Arguments/cwd/stdio/exit and fail-closed parsing conform | `artifacts/ft-002/verify/chk-02/` |
| `CHK-03` | Stdout/stderr contract | Run golden event-stream tests for every terminal path | Exact records, fields, ordering, counters, durations, and raw-output isolation pass | `artifacts/ft-002/verify/chk-03/` |
| `CHK-04` | Complete executable change | Run `go test ./...`, `go vet ./...`, required CI, and independent convergence review | All required automated suites and review gates pass | `artifacts/ft-002/verify/chk-04/` |
| `CHK-05` | Repository/doc integrity | Run `make docs-lint` and `git diff --check` | Both commands pass | `artifacts/ft-002/verify/chk-05/` |
| `CHK-06` | Release artifacts | Run approved reproducible-build, identity/checksum, install, and per-target smoke procedures | Every approved target passes; unsupported targets are absent | `artifacts/ft-002/verify/chk-06/` |

### Test matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | `artifacts/ft-002/verify/chk-01/` |
| `CHK-02` | `EVID-02` | `artifacts/ft-002/verify/chk-02/` |
| `CHK-03` | `EVID-03` | `artifacts/ft-002/verify/chk-03/` |
| `CHK-04` | `EVID-04` | `artifacts/ft-002/verify/chk-04/` |
| `CHK-05` | `EVID-05` | `artifacts/ft-002/verify/chk-05/` |
| `CHK-06` | `EVID-06` | `artifacts/ft-002/verify/chk-06/` |

### Evidence

- `EVID-01` State/config suite report covering `SC-01`, `SC-03`–`SC-05` and their negative cases.
- `EVID-02` Versioned fake Codex input/output corpus and process/parser boundary report.
- `EVID-03` Golden stdout/stderr outputs for all terminal paths.
- `EVID-04` Go test/vet, required CI, and independent convergence-review results.
- `EVID-05` Documentation lint and diff-integrity results.
- `EVID-06` Reproducible artifact manifest, supported-target matrix, checksums/identity, installation and smoke results, plus approval/backout references.

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | State/config test report | deterministic test runner | `artifacts/ft-002/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Fixture corpus and parser/process report | deterministic fake-executable suite | `artifacts/ft-002/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Event golden report | deterministic event suite | `artifacts/ft-002/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Test/vet/CI/review bundle | local/CI runners and independent reviewer | `artifacts/ft-002/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Docs/diff reports | local or CI runner | `artifacts/ft-002/verify/chk-05/` | `CHK-05` |
| `EVID-06` | Release evidence bundle | approved release verifier | `artifacts/ft-002/verify/chk-06/` | `CHK-06` |

### Evidence Results

| Evidence ID | Concrete carriers | Result |
| --- | --- | --- |
| `EVID-01` | `internal/config/config_test.go`, `internal/workflow/workflow_test.go` | Pass locally and in required CI. |
| `EVID-02` | `internal/codex/adapter_test.go`, `internal/runner/runner_test.go` | Parser/process corpus, failure diagnostics and cancellation pass. |
| `EVID-03` | `internal/event/event_test.go`, workflow terminal-path tests | Schema, counters, ordering and failed-writer behavior pass. |
| `EVID-04` | [PR #4](https://github.com/dapi/reviewer/pull/4), [Verify CI](https://github.com/dapi/reviewer/actions/runs/29839977861/job/88666073511), Codex review session `019f8519-b65f-77b1-81c1-141503394e38` | CI green; fourth independent review found no correctness issue. |
| `EVID-05` | `make docs-lint`, `git diff --check`, same CI job | Pass. |
| `EVID-06` | `tools/build-dist`, `SHA256SUMS` generated twice, native macOS ARM64 smoke, CI Linux AMD64 smoke | Identical checksums; all four archives verify; native and CI smoke pass. |
