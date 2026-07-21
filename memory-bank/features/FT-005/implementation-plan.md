---
title: "FT-005: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Grounded execution plan for fast/best profile resolution, configuration visibility, Codex binding, tests, and public docs."
derived_from:
  - brief.md
  - design.md
status: archived
audience: humans_and_agents
must_not_define:
  - ft_005_scope
  - ft_005_selected_design
  - ft_005_acceptance_criteria
  - ft_005_blocker_state
  - ft_005_validation_profile
---

# FT-005: Implementation Plan

## Цель текущего плана

Implement the active `brief.md` and `design.md` as one compatibility-aware configuration change, with deterministic acceptance evidence and no live Codex or release action.

## Grounding / Support References

| Document | Role in this plan | Facts reused | Conflict action |
| --- | --- | --- | --- |
| `brief.md` | canonical problem/profile/verify owner | `REQ-*`, scenarios, checks, evidence | update `brief.md` first |
| `design.md` | canonical solution owner | profiles, resolution tiers, contracts, failures, rollout | update `design.md` first |
| `decision-log.md` | provenance only | FPF evidence and conflict history | promote semantic changes to canonical owner first |
| `../../../README.md` | public CLI contract | current sources/options and proposed profile values | update atomically; README remains public owner |

## Current State / Reference Points

| Path / module | Current role | Why relevant | Reuse / mirror |
| --- | --- | --- | --- |
| `internal/config/config.go` | scalar spec list, five-level resolution, validation, source metadata, rendering | central change surface for mode/profile tiers | preserve existing explicit-source resolver and `Setting` metadata pattern |
| `internal/config/config_test.go` | precedence, validation, prompt, formatting tests | nearest deterministic coverage | extend environment cleanup, table-driven sources, and format assertions |
| `internal/app/app.go` | flag binding and config command dispatch | owns new CLI flags | use existing `optionalFlag`/`bind` pattern |
| `internal/app/app_test.go` | CLI/config behavior tests | validates errors and user-visible output | add mode/effort command cases without starting workflow |
| `internal/codex/adapter.go` | translates resolved fields to Codex `-c` arguments | Finalize/Fix CI currently omit effort | reuse `modelArgs` for all four stages |
| `internal/codex/adapter_test.go` | fake-runner argument capture | canonical process-boundary evidence | assert exact model/effort pair per invocation |
| `README.md` | sole public CLI/config contract | proposed profiles become operative | update mode/options/config example and remove “not implemented” wording |
| `memory-bank/domain/model.md`, `domain/rules.md`, `ops/config.md` | derived interpretation of public configuration | may need Tell+Cite updates if effective-profile/source concept is material | keep values only in root README; update interpretations only if needed |

## Test Strategy

| Test surface | Canonical refs | Existing coverage | Planned automated coverage | Required local suites / commands | Required CI suites / jobs | Manual-only gap / justification | Approval ref |
| --- | --- | --- | --- | --- | --- | --- | --- |
| mode/profile resolver | `REQ-01`, `REQ-02`, `SC-01`, `SC-02`, `NEG-01`, `SOL-01`, `SOL-02` | scalar defaults and validation | both profiles, default, invalid/empty mode, all eight fields | `go test ./internal/config ./internal/app` | required Verify job | none | none |
| precedence/source metadata | `REQ-03`, `REQ-04`, `SC-03`, `SC-04`, `NEG-02`, `CTR-02`, `CTR-03`, `CTR-07` | one scalar precedence chain | every source, mode/stage cross-dimension, equal-string explicit values, rendered sources and value-based built-in suffix | `go test ./internal/config ./internal/app` | required Verify job | none | none |
| Codex invocation | `REQ-05`, `SC-05`, `CTR-04`, `FM-04` | all stages, effort only first two | exact model/effort for all stages under profiles and override | `go test ./internal/codex` | required Verify job | none; fake runner is mandated deterministic substitute | none |
| repository/public docs | `REQ-05`, `REQ-06`, `CTR-05`, `FM-05`, `RB-01` | repository checks | operative README and any necessary Tell+Cite docs; full suite and reviews | `go test ./...`; `go vet ./...`; `make docs-lint`; `git diff --check` | all required jobs | release smoke belongs to normal release flow, not this config feature | none |

## Open Questions / Ambiguities

`none`. Material questions were resolved from current carriers through FPF and promoted to `design.md`; provenance is in `decision-log.md`.

## Environment Contract

| Area | Contract | Used by | Failure symptom |
| --- | --- | --- | --- |
| setup | Go toolchain and repository Make targets already used by required checks | all steps | existing suites cannot build/run |
| test | fake runner only; tests must isolate `REVIEWER_*`, project, user, and CLI sources | `STEP-01`–`STEP-04` | host config leaks into expected results |
| access/network/secrets | no network, model access, GitHub write, or Codex authentication required | all steps | a test attempts live Codex/remote access |

## Preconditions

| Precondition ID | Canonical ref | Required state | Used by steps | Blocks start |
| --- | --- | --- | --- | --- |
| `PRE-01` | `brief.md`, `design.md` | both active; no unresolved `DEC-*`; `standard` contract defined in brief | all | yes |
| `PRE-02` | `CTR-05` | root README remains public owner and is updated in the same implementation change | `STEP-04` | yes |

## Design Realization Mapping

| Canonical solution refs | Owner | Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- | --- | --- |
| `C4-01`, `SOL-01`, `SD-01`, `CTR-01`, `FM-01` | `design.md` | operator/env/file → app/config input boundary and mode validation | `STEP-01` | `CHK-01` | `EVID-01` |
| `SOL-02`, `SD-03`, `SD-04`, `CTR-02`, `CTR-03`, `CTR-06`, `CTR-07`, `INV-02`, `FM-02`, `FM-03` | `design.md` | config profile table, per-field resolution metadata, formatter | `STEP-02` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `SOL-03`, `SD-02`, `CTR-04`, `FM-04` | `design.md` | config/app fields and Codex adapter arguments | `STEP-03` | `CHK-03` | `EVID-03` |
| `SOL-04`, `SOL-05`, `SD-04`, `CTR-03`, `CTR-05`, `CTR-07`, `INV-01`, `FM-05`, `RB-01`, `RB-02` | `design.md` | config output, root README, derived Tell+Cite docs, final verification | `STEP-04`, `STEP-05` | `CHK-01`, `CHK-04` | `EVID-01`, `EVID-04` |

## Workstreams

| Workstream | Implements | Result | Owner | Dependencies |
| --- | --- | --- | --- | --- |
| `WS-1` | `REQ-01`–`REQ-04`, `SOL-01`, `SOL-02`, `SOL-04` | complete deterministic resolver and config rendering | agent | `PRE-01` |
| `WS-2` | `REQ-05`, `SOL-03` | all four stage invocations use resolved pairs | agent | `WS-1` |
| `WS-3` | `REQ-05`, `REQ-06`, `SOL-05`, `RB-01` | public/docs/test/review convergence | agent | `WS-1`, `WS-2`, `PRE-02` |

## Approval Gates

`none`. The plan changes local code/docs and runs deterministic checks; it does not publish a release or mutate external systems.

## Порядок работ

| Step ID | Actor | Implements | Goal | Touchpoints | Artifact | Verifies | Evidence IDs | Check command / procedure | Blocked by | Needs approval | Escalate if |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `STEP-01` | agent | `REQ-01`, `SOL-01`, `SD-01`, `CTR-01`, `FM-01` | add mode inputs, validation, built-in selection, and focused tests | app/config code and tests | validated effective mode with source | `CHK-01` | `EVID-01` | targeted config/app tests | `PRE-01` | none | accepted values/names conflict with new canonical facts |
| `STEP-02` | agent | `REQ-02`–`REQ-04`, `SOL-02`, `SOL-04`, `SD-03`, `SD-04`, `CTR-02`, `CTR-03`, `CTR-06`, `CTR-07`, `FM-02`, `FM-03` | add exact profile fallback, explicit tier, metadata, built-in comparison, and exhaustive tests | config resolver/formatter/tests | eight effective stage settings and auditable sources | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` | targeted matrix/golden tests | `STEP-01` | none | explicit-vs-profile precedence cannot be represented without contract change |
| `STEP-03` | agent | `REQ-05`, `SOL-03`, `SD-02`, `CTR-04`, `FM-04` | bind missing effort options and invoke all stages with model/effort | app/config/codex code and tests | four complete invocation pairs | `CHK-03` | `EVID-03` | adapter fake-runner tests | `STEP-02` | none | Codex CLI contract differs from current `modelArgs` assumption |
| `STEP-04` | agent | `REQ-04`–`REQ-06`, `SOL-05`, `CTR-05`, `INV-01`, `FM-05`, `RB-01`, `RB-02` | update public/derived docs and run functional verification | README, applicable Memory Bank docs, whole repo | coherent executable/public contract | `CHK-04` | `EVID-04` | full required commands | `STEP-03`, `PRE-02` | none | docs require behavior outside brief scope |
| `STEP-05` | independent reviewer | `REQ-06`, `RB-01` | run simplify review, independent convergence review, and close evidence | complete diff/package | acceptance-ready change or findings | `CHK-04` | `EVID-04` | separate semantic review passes | `STEP-04` | none | critical/important finding cannot be resolved from canonical owners |

## Parallelizable Work

- `PAR-01` README wording can be drafted alongside adapter tests after `STEP-02`, but it must not be finalized before implementation semantics stabilize.
- `PAR-02` Resolver and adapter production edits are sequential because adapter fields depend on the resolved configuration shape.

## Checkpoints

| Checkpoint ID | Refs | Condition | Evidence IDs |
| --- | --- | --- | --- |
| `CP-01` | `STEP-01`, `STEP-02`, `CHK-01`, `CHK-02` | both profiles and all explicit-source conflicts resolve with correct metadata | `EVID-01`, `EVID-02` |
| `CP-02` | `STEP-03`, `CHK-03`, `CTR-04` | all four fake invocations contain exact resolved pairs | `EVID-03` |
| `CP-03` | `STEP-04`, `STEP-05`, `CHK-04` | full suites/docs/CI and separate simplify/convergence reviews pass | `EVID-04` |

## Execution Risks

| Risk ID | Risk | Impact | Mitigation | Trigger |
| --- | --- | --- | --- | --- |
| `ER-01` | treating profile as ordinary low-priority source loses explicit-equal metadata or creates wrong cross-source ordering | operator intent/config output incorrect | separate explicit candidate resolution from profile fallback and test cross-product | any matrix row reports profile over explicit source |
| `ER-02` | existing tests construct partial `config.Config` values directly | unrelated workflow/adapter tests fail after efforts become required | keep resolver validation at load boundary; update direct fixtures only where invocation semantics need values | failures from zero-value direct fixtures |
| `ER-03` | README retains old independent built-ins or proposed-only language | public contract contradicts runtime | semantic read-through against `CTR-05` and config golden output | old defaults/profile disclaimer remains after implementation |

## Stop Conditions / Fallback

| Stop ID | Related refs | Trigger | Immediate action | Safe fallback state |
| --- | --- | --- | --- | --- |
| `STOP-01` | `PRE-01`, `CTR-01`–`CTR-07` | new evidence makes mode naming, precedence, profile values, source/built-in rendering, or required effort surfaces materially ambiguous | stop implementation; update decision log and canonical owner; request human decision if evidence cannot resolve it | active package, no contradictory runtime change |
| `STOP-02` | `CON-03`, `INV-01` | verification would require live Codex/remote mutation or changes workflow/event semantics | stop and re-route the added scope | deterministic local change only |

## Plan-local Evidence

No separate plan-local evidence IDs are needed; all execution outputs map to the canonical `EVID-01`–`EVID-04` contract in `brief.md`.

## Готово для приемки

- `CP-01`–`CP-03` have concrete evidence.
- All canonical checks pass locally and in required CI.
- Simplify review and independent convergence review are distinct and clean.
- Root README, configuration output, code, and feature package agree.
- Final acceptance and lifecycle transition use `brief.md#verify`.
