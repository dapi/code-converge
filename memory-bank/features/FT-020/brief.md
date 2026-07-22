---
title: "FT-020: Root CLI help aliases"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem/verify owner for accepting `-h` and `--help` at the root `code-converge` CLI boundary."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../product/context.md
  - ../../engineering/architecture.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/20
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-020: Root CLI help aliases

## What

### Problem

The root `code-converge` command does not accept conventional `-h` and `--help` aliases. Consequently, an operator cannot request root help without taking the ordinary flag/error path. Issue #20 requires both aliases to write the existing root usage/help text to stdout, exit successfully, and avoid every review or update workflow stage.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Root help aliases | Neither alias is accepted as a successful root command | Both `-h` and `--help` write the same defined root help text to stdout and exit `0` | Deterministic app tests |
| `MET-02` | Help invocation side effects | Root argument parsing proceeds toward normal command/configuration handling | A help invocation starts neither review nor update and writes no error | Fake-runner/updater app tests |

### Scope

- `REQ-01` Accept root `code-converge -h` and `code-converge --help` as aliases.
- `REQ-02` Each alias writes the same root usage/help text to stdout, exits `0`, and writes nothing to stderr.
- `REQ-03` Each alias returns before configuration loading, review setup, agent invocation, update dispatch, progress/event output, or any other operational stage.
- `REQ-04` Add deterministic coverage for each alias and update the root public CLI contract and dependent Memory Bank owners with the implementation.

### Non-Scope

- `NS-01` Adding help flags to the `update` subcommand or changing its `--yes`/`-y` contract.
- `NS-02` Changing root flag meanings, configuration precedence, workflow behavior, update behavior, progress formats, or existing non-help argument-error semantics.
- `NS-03` Inventing additional commands, aliases, configuration options, or an interactive help system.

### Constraints / Assumptions

- `ASM-01` Issue #20 is the source of intent and acceptance: both aliases must be conventional, successful, stdout-only, and stage-free.
- `ASM-02` The root [`README.md`](../../../README.md) owns the public CLI contract; `internal/app` owns root argument parsing and command selection.
- `CON-01` The current root parser discards Go `flag` output. Its only existing root usage text is the stderr error `code-converge: usage: code-converge [flags] [config]` for invalid positional arguments; it has no documented or emitted root help text.
- `DEC-01` Resolved in `DL-03`: root help is the stable line `usage: code-converge [flags] [config]\n`. It is minimal, does not duplicate a volatile flag inventory, and is directly consumable by humans and agents.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | This changes the public CLI argument, stdout, and exit contract. The exact help payload and early-return boundary are selected solution/contract decisions. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | FPF closure and review-improve provenance establish the agent-safe help contract. | Feature-local provenance only |
| `design.md` and `implementation-plan.md` | selected | The resolved public output contract needs explicit solution and grounded execution owners. | Existing sibling documents |
| Separate contract, C4, sequence, or feature use-case artifact | omitted | The root CLI boundary and one synchronous early return can be covered compactly in `design.md`; no new stable project scenario is introduced. | `design.md` / existing project owners |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `standard` | The issue changes a public CLI argument plus stdout and exit behavior. There is no security, persistent-data, migration, concurrency, or cross-system trigger requiring `high-risk`; documentation-only and low-risk do not cover the changed contract. | `none` |

## Verify

### Exit Criteria

- `EC-01` `code-converge -h` and `code-converge --help` each exit `0`, write the selected identical root help text to stdout, and write nothing to stderr.
- `EC-02` Neither root-help invocation loads configuration, invokes the injected process runner or updater, emits workflow progress/events, or reaches a review/update stage.
- `EC-03` Root README, the feature brief/design/plan and validation evidence agree on the selected public help contract; all selected validation passes.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `ASM-01`, `CON-01`, `DEC-01` | `EC-01`, `SC-01`, `SC-02` | `CHK-01` | `EVID-01` |
| `REQ-02` | `ASM-01`, `DEC-01` | `EC-01`, `SC-01`, `SC-02` | `CHK-01` | `EVID-01` |
| `REQ-03` | `ASM-01`, `ASM-02` | `EC-02`, `SC-01`, `SC-02`, `NEG-01` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `REQ-04` | `ASM-02` | `EC-03`, `SC-03` | `CHK-02`, `CHK-03` | `EVID-02`, `EVID-03` |

### Acceptance Scenarios

- `SC-01` Invoking the root command with `-h` writes the selected root help text only to stdout and exits successfully before operational setup.
- `SC-02` Invoking the root command with `--help` has the same stdout, stderr, exit-code, and no-side-effect outcome as `SC-01`.
- `SC-03` The root README and all instantiated FT-020 downstream artifacts describe the delivered contract consistently.

### Negative Coverage

- `NEG-01` A root-help invocation must not invoke the fake process runner or updater, load an invalid configuration, emit `run_started`/`run_completed`, or write a diagnostic to stderr.
- `NEG-02` The implementation must not broaden this feature into subcommand help or alter the existing handling of unrelated invalid positional arguments.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `EC-01`, `SC-01`, `SC-02`, `NEG-01` | Deterministic `internal/app` tests with each alias, buffered stdout/stderr, fake runner and fake updater | Both aliases match the selected stdout contract, return `0`, and have no runner/updater invocation. | `artifacts/ft-020/verify/chk-01/` |
| `CHK-02` | `EC-02`, `EC-03`, `SC-03`, `NEG-01`, `NEG-02` | `go test ./internal/app` plus semantic public-contract review | Early return and unchanged non-help boundary are covered; documents converge. | `artifacts/ft-020/verify/chk-02/` |
| `CHK-03` | `EC-03` | `go test ./... && go vet ./... && make docs-lint && git diff --check` | Full local validation is green. | `artifacts/ft-020/verify/chk-03/` |

### Test matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | `artifacts/ft-020/verify/chk-01/` |
| `CHK-02` | `EVID-02` | `artifacts/ft-020/verify/chk-02/` |
| `CHK-03` | `EVID-03` | `artifacts/ft-020/verify/chk-03/` |

### Evidence

- `EVID-01` Alias-specific deterministic app-test log.
- `EVID-02` Focused app-test and public-contract review record.
- `EVID-03` Full local verification log and required CI reference at delivery closure.

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | Alias-specific app-test log | Test runner | `artifacts/ft-020/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Focused app-test and contract-review record | Test runner/reviewer | `artifacts/ft-020/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Full verification and CI references | Test runner/CI | `artifacts/ft-020/verify/chk-03/` | `CHK-03` |

### Execution Evidence Status

| Evidence ID | Status | Concrete carrier |
| --- | --- | --- |
| `EVID-01` | pass | `TestRootHelpAliasesExitBeforeOperationalSetup` deterministically covers both aliases, exact stdout, empty stderr and no fake runner/updater calls. |
| `EVID-02` | pass | `go test ./internal/app` plus the root README and FT-020 contract review. |
| `EVID-03` | pass | Local `go test ./...`, `go vet ./...`, built-binary alias smoke checks, `make docs-lint` and `git diff --check` pass; PR [#21](https://github.com/dapi/code-converge/pull/21) is mergeable and required [Verify run](https://github.com/dapi/code-converge/actions/runs/29957460759) passed. |
