---
title: "FT-005: Fast and Best Model Profiles"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem space и verify contract для fast/best mode, выбирающего stage-specific model и reasoning effort."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../../README.md
  - https://github.com/dapi/reviewer/issues/5
status: active
delivery_status: in_progress
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
canonical_for:
  - ft_005_problem_space
  - ft_005_validation_profile
  - ft_005_verify_contract
  - ft_005_blocker_state
---

# FT-005: Fast and Best Model Profiles

## What

### Problem

Reviewer exposes independent model settings and only partial reasoning-effort settings. Operators must currently assemble a stage policy setting by setting, while the root README already documents two intended stage profiles as non-operative guidance. Issue [#5](https://github.com/dapi/reviewer/issues/5) requires one mode selector that makes those profiles executable without weakening explicit stage overrides or configuration explainability.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Default policy completeness | No operative mode; independent built-ins | An unconfigured run resolves all four stages from `fast` | Default-resolution and Codex invocation tests |
| `MET-02` | Override preservation | Existing source precedence for stage settings | Every explicit stage setting wins over the profile; conflicts among explicit values retain existing CLI > project > user > environment precedence | Full source/profile precedence matrix |
| `MET-03` | Explainability | `reviewer config` lists current settings and sources | Output identifies effective mode/source and every resolved stage model/effort/source | Golden config tests |

### Scope

- `REQ-01` Add one mode configuration value with valid values `fast` and `best`; absent configuration resolves to `fast`, and invalid or empty explicit values fail as configuration errors.
- `REQ-02` Resolve Review, Fix findings, Finalize, and Fix CI model plus reasoning effort from the selected profile using the exact profile values documented by issue #5 and the root README.
- `REQ-03` Treat any explicit per-stage model or reasoning-effort value from CLI, project, user, or environment as higher priority than the selected profile; among explicit values preserve the existing source precedence.
- `REQ-04` Make `reviewer config` show effective mode, mode source, and every resolved stage model and reasoning effort with an unambiguous effective source.
- `REQ-05` Pass every resolved model and reasoning effort through the existing Codex adapter boundary for its stage and update the public CLI/configuration documentation.
- `REQ-06` Provide deterministic coverage for defaults, both profiles, every configuration source and precedence conflict, invalid values, config rendering, and all four Codex invocations.

### Non-Scope

- `NS-01` Dynamic escalation to `gpt-5.6-sol`, automatic profile switching during a run, or model availability probing.
- `NS-02` Changing prompts, workflow stages, budgets, stdout event schema, finalization verdicts, exit-code meanings, or configuration source ordering.
- `NS-03` Adding modes beyond `fast` and `best`, per-repository adaptive heuristics, or live Codex calls in tests.

### Constraints / Assumptions

- `ASM-01` The root README owns the public option names, values, profile table, source ordering, and rendered configuration contract; this package must update that owner during implementation rather than becoming a second public contract.
- `ASM-02` An explicitly configured stage value is distinguishable from a value inherited from a profile, even when their strings are equal.
- `CON-01` Profile lookup is a fallback layer, not a fifth competing configuration source: explicit stage values from all four sources outrank it.
- `CON-02` Resolution occurs before the workflow starts; a run does not change mode or effective stage settings in response to findings or CI results.
- `CON-03` Deterministic tests use fake runners and must not require access to the named models.

No unresolved blocking `DEC-*` remains. Feature-local decisions and their FPF provenance are recorded in `design.md` and `decision-log.md`.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The feature changes CLI/env/file configuration contracts, adds cross-setting precedence semantics, and changes all four Codex invocation configurations. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | The requested iterative review and FPF decisions need auditable provenance without redefining canonical facts. | Feature-support provenance; accepted solution facts are promoted to `design.md`. |
| `design.md` | selected | Contract and resolution-order changes require an explicit solution owner. | `design.md` |
| `implementation-plan.md` | selected | Configuration, adapter, tests, and documentation require coordinated execution. | `implementation-plan.md` after upstream gates are ready. |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `standard` | Executable CLI/config contract changes and four stage invocation paths require full affected regression, acceptance, contract, and integration coverage. No security, persistent-data, concurrency, cross-system protocol, or release/deployment trigger raises the profile. | `none` |

## Verify

### Exit Criteria

- `EC-01` Default, `fast`, and `best` resolution produce the documented complete eight-value stage matrix.
- `EC-02` Explicit stage overrides win over profiles while explicit-source conflicts retain the existing source order.
- `EC-03` Invalid/empty modes and invalid/empty required stage settings fail before workflow execution with existing configuration-error semantics.
- `EC-04` `reviewer config` exposes mode/source and every effective stage value/source without implying that a profile-derived value came from an explicit source.
- `EC-05` All four Codex invocations receive their resolved model and reasoning effort; repository tests, vet, docs lint, diff check, and independent convergence review pass.

### Traceability Matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `CON-02` | `EC-01`, `EC-03`, `SC-01`, `NEG-01` | `CHK-01`, `CHK-04` | `EVID-01`, `EVID-04` |
| `REQ-02` | `CON-02`, `CON-03` | `EC-01`, `SC-01`, `SC-02` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `REQ-03` | `ASM-02`, `CON-01` | `EC-02`, `SC-03`, `NEG-02` | `CHK-02` | `EVID-02` |
| `REQ-04` | `ASM-01`, `ASM-02` | `EC-04`, `SC-04` | `CHK-01`, `CHK-04` | `EVID-01`, `EVID-04` |
| `REQ-05` | `ASM-01` | `EC-05`, `SC-05` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` |
| `REQ-06` | `CON-03` | `EC-05`, `SC-01`–`SC-05`, `NEG-01`, `NEG-02` | `CHK-01`–`CHK-04` | `EVID-01`–`EVID-04` |

### Acceptance Scenarios

- `SC-01` With no mode or stage overrides, `reviewer config` reports `fast` as built-in and all eight stage values from the documented fast profile.
- `SC-02` Each explicit valid mode selects exactly its documented four model/effort pairs.
- `SC-03` For each stage field, environment, user, project, and CLI explicit values override the selected profile; conflicts among those values resolve CLI > project > user > environment.
- `SC-04` Config output reports the winning mode source and distinguishes profile-derived stage values from explicit stage-source winners, including equal-string overrides.
- `SC-05` Review, Fix findings, Finalize, and Fix CI invocations each contain the resolved `model` and `model_reasoning_effort` arguments.

### Negative Scenarios

- `NEG-01` Unknown, whitespace-only, or explicitly empty mode values fail as configuration errors and do not invoke Codex.
- `NEG-02` Empty required explicit model or reasoning-effort values fail instead of silently falling back to the selected profile; the selected profile never overrides an explicit valid stage value.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | mode/profile resolution and config rendering | Run table-driven `internal/config` tests and config command golden tests | Profiles, validation, source metadata, and output conform | changed Go tests and local output |
| `CHK-02` | two-dimensional precedence | Run pairwise/full-matrix tests across mode source, stage source, and all eight fields | Every explicit field wins; explicit-source ordering is unchanged | changed config test table |
| `CHK-03` | Codex invocation boundary | Run fake-runner adapter tests for all four stages under profile and override cases | Both `model` and effort reach every invocation | changed adapter tests |
| `CHK-04` | executable and documentation convergence | Run `go test ./...`, `go vet ./...`, `make docs-lint`, `git diff --check`, required CI, simplify review, and independent final review | All applicable gates pass | local/CI/review output |

### Test Matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | config/app tests and output |
| `CHK-02` | `EVID-02` | precedence test table and output |
| `CHK-03` | `EVID-03` | adapter invocation tests and output |
| `CHK-04` | `EVID-04` | repository gates, CI, simplify and convergence reviews |

### Evidence

- `EVID-01` Deterministic profile-resolution, validation, and `reviewer config` report.
- `EVID-02` Explicit cross-source/profile precedence matrix report for all eight stage fields.
- `EVID-03` Fake-runner invocation report for Review, Fix findings, Finalize, and Fix CI.
- `EVID-04` Repository verification, required CI, simplify-review, independent convergence-review, and documentation-contract results.

### Evidence Contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | config/profile test report | deterministic Go tests | changed tests plus command output | `CHK-01` |
| `EVID-02` | precedence matrix report | deterministic Go tests | changed config tests plus command output | `CHK-02` |
| `EVID-03` | invocation argument report | fake-runner adapter tests | changed adapter tests plus command output | `CHK-03` |
| `EVID-04` | verification/review bundle | local/CI runners and independent reviewer | command output, CI run, review result | `CHK-04` |
