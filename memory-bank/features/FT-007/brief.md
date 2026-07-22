---
title: "FT-007: Rename Project to code-converge"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem space и verify contract для переименования проекта, CLI, конфигурационных контрактов и release identity."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../../README.md
  - https://github.com/dapi/reviewer/issues/7
status: active
delivery_status: in_progress
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
canonical_for:
  - ft_007_problem_space
  - ft_007_validation_profile
  - ft_007_verify_contract
  - ft_007_blocker_state
---

# FT-007: Rename Project to `code-converge`

## What

### Problem

Issue #7 requires changing the identity of the local CLI and its GitHub repository from `reviewer` to `code-converge`. The change crosses source, public CLI/configuration, Go module/build references, documentation, CI/release automation and distribution artifacts. The current repository and public contract are still named `reviewer`.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | New identity convergence | Source, docs, CI, installers and artifacts use `reviewer` identity | All intended public and delivery surfaces use `code-converge`; remaining `reviewer` occurrences are explicitly classified migration references | Repository identity inventory and semantic review |
| `MET-02` | Executable contract | `reviewer` command/binary and `.reviewer` / `REVIEWER_*` contract | `code-converge` command/binary and documented new configuration contract work from a clean checkout | Build, CLI/config contract and smoke tests |
| `MET-03` | Workflow preservation | Existing workflow, exit codes, stdout event schema and Codex integration | Behavior remains unchanged except intentionally renamed identifiers | Existing contract/regression suites and stdout golden tests |
| `MET-04` | Migration readiness | No migration policy for old names | Human-approved compatibility policy and migration notes exist before merge | Decision/design review and migration documentation check |

### Scope

- `REQ-01` Rename the GitHub repository identity from `dapi/reviewer` to `dapi/code-converge`; the external rename is owned by `dapi`.
- `REQ-02` Rename the public CLI command, binary, Go module path and package/build references to `code-converge`.
- `REQ-03` Update CLI help, examples, README, Memory Bank, prompts, tests, scripts, Makefile, CI/release workflows, installer and distribution archive names.
- `REQ-04` Rename applicable configuration directories, files, environment variables and configuration keys; old `reviewer` / `REVIEWER_*` names are not read.
- `REQ-05` Update versioning and release documentation/metadata so future artifacts are published under `code-converge`.
- `REQ-06` Preserve workflow semantics, exit codes, stdout event schema and Codex integration except for explicitly documented compatibility changes.
- `REQ-07` Add migration notes and make the compatibility policy available before implementation is merged.

### Non-Scope

- `NS-01` Changing review/fix/finalize workflow semantics, budgets, finding classification, exit-code meanings or stdout event fields.
- `NS-02` Introducing a second product, new workflow stages, new Codex integration or unrelated repository refactoring.
- `NS-03` Treating historical issue/commit text as an unintended current identity; historical references may remain when clearly marked as history or migration context.
- `NS-04` Executing the GitHub repository rename, publishing releases or changing live external state before the required human approval.

### Constraints / Assumptions

- `CON-01` The root `README.md` remains the canonical public CLI contract; feature documents may not invent a parallel option/configuration contract.
- `CON-02` The feature is one delivery-unit, but its rollout crosses local code, repository hosting and release systems; implementation must preserve the existing behavior contract.
- `CON-03` Existing users may have scripts, aliases, `.reviewer` directories, `~/.reviewer/` files and `REVIEWER_*` variables; the approved clean break deliberately does not read or migrate them.
- `CON-04` GitHub may redirect the old repository URL after rename, but issue #7 makes redirect behavior conditional rather than guaranteed by this repository.
- `CON-05` Owner `dapi` approved a clean break: old command/configuration names are not supported, aliased, read, or migrated.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The feature changes CLI, Go module, env/config contracts, repository identity, release artifacts and migration behavior. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | FPF reasoning, contradiction resolution, review cycles and human gates need provenance without becoming a canonical contract | feature-support provenance / `decision-log.md` |
| `design.md` | selected | Identity migration, compatibility and repository/release boundaries require a solution owner | `design.md` |
| `implementation-plan.md` | selected | Coordinated source, config, CI, installer and release execution needs one derived sequence | `implementation-plan.md` |
| `runtime-surfaces.md` | omitted | Current issue scope is broad but the affected surfaces are already enumerated by REQ-02/03 and need no separate owner at bootstrap | none |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `high-risk` | Breaking public CLI/configuration identity change, cross-system repository identity change and release/installer artifact impact. Owner `dapi` approved the clean-break policy; release/deployment obligations apply during rollout. | `dapi` approval recorded in `decision-log.md` |

## Verify

### Exit Criteria

- `EC-01` A clean checkout builds and runs `code-converge`; `code-converge config` and all documented flags/configuration sources use the approved new identity.
- `EC-02` The identity inventory finds no unintended current `reviewer` identity in source, docs, CI, installers, release assets or repository links; FT-007 migration references are classified.
- `EC-03` Existing workflow behavior and contract tests remain green, apart from intentionally renamed identifiers.
- `EC-04` Release archives, installer behavior, smoke tests and GitHub Actions use the new repository and artifact names.
- `EC-05` The clean-break policy is documented and tested: old names are rejected/not read, and the renamed repository is reachable.

### Traceability Matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `CON-04` | `EC-05` | `CHK-05` | `EVID-05` |
| `REQ-02` | `CON-01` | `EC-01`, `EC-02` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `REQ-03` | `CON-01` | `EC-02`, `EC-04` | `CHK-02`, `CHK-04` | `EVID-02`, `EVID-04` |
| `REQ-04` | `CON-03`, `CON-05` | `EC-01`, `EC-05` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `REQ-05` | `CON-02` | `EC-04` | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` |
| `REQ-06` | `CON-02` | `EC-03` | `CHK-03` | `EVID-03` |
| `REQ-07` | `CON-05` | `EC-05` | `CHK-05` | `EVID-05` |

### Acceptance Scenarios

- `SC-01` From a clean checkout, an operator invokes `code-converge` and `code-converge config`; the command, help, flags and effective configuration use the approved new identity.
- `SC-02` A repository-wide identity inventory classifies every remaining `reviewer` occurrence as intentional migration/history or reports a current-surface defect.
- `SC-03` Existing workflow fixtures run with unchanged stage order, event records, exit codes and Codex argument semantics, excluding renamed tokens.
- `SC-04` A release build produces the approved `code-converge` archive/binary names and CI/installer references resolve against the renamed repository.
- `SC-05` A user following the clean-break notes receives no old-name compatibility; old command/configuration inputs are not recognized or read.

### Negative Scenarios

- `NEG-01` Old command/configuration names are not silently accepted or read.
- `NEG-02` A stale repository, module, archive, installer or `REVIEWER_*` reference fails the identity inventory unless explicitly classified.
- `NEG-03` Rename work does not alter workflow events, exit codes or Codex integration as an accidental side effect.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | clean build and public CLI/config contract | Build and run documented command/config smoke tests from a clean checkout | New identity works and public surfaces agree | `artifacts/ft-007/verify/chk-01/` |
| `CHK-02` | identity convergence | Run a reviewed inventory over source, docs, CI, scripts, installers, releases and URLs | Only approved migration/history references remain | `artifacts/ft-007/verify/chk-02/` |
| `CHK-03` | behavior preservation and migration | Run existing Go/contract/stdout tests plus approved old-name policy matrix | Existing behavior remains green and old names follow policy | `artifacts/ft-007/verify/chk-03/` |
| `CHK-04` | build/release surfaces | Run distribution, installer, workflow and archive-name checks | Release outputs and automation use new identity | `artifacts/ft-007/verify/chk-04/` |
| `CHK-05` | external rename and approvals | Human-approved repository rename/reachability/redirect and release smoke procedure | External state and migration evidence match approved decision | `artifacts/ft-007/verify/chk-05/` |

### Test Matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | `artifacts/ft-007/verify/chk-01/` |
| `CHK-02` | `EVID-02` | `artifacts/ft-007/verify/chk-02/` |
| `CHK-03` | `EVID-03` | `artifacts/ft-007/verify/chk-03/` |
| `CHK-04` | `EVID-04` | `artifacts/ft-007/verify/chk-04/` |
| `CHK-05` | `EVID-05` | `artifacts/ft-007/verify/chk-05/` |

### Evidence

- `EVID-01` Clean-checkout build and CLI/config smoke output.
- `EVID-02` Reviewed identity inventory and classified residual-reference report.
- `EVID-03` Existing regression/contract/stdout tests and approved migration-policy matrix.
- `EVID-04` Distribution, installer, CI and release artifact verification.
- `EVID-05` Human approval record, renamed repository reachability/redirect result and release smoke evidence.

### Evidence Contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | build and smoke report | local/CI runner | `artifacts/ft-007/verify/chk-01/` | `CHK-01` |
| `EVID-02` | identity inventory | implementer + independent code-converge | `artifacts/ft-007/verify/chk-02/` | `CHK-02` |
| `EVID-03` | regression and migration matrix | deterministic test runner | `artifacts/ft-007/verify/chk-03/` | `CHK-03` |
| `EVID-04` | distribution/release report | local/CI/release runner | `artifacts/ft-007/verify/chk-04/` | `CHK-04` |
| `EVID-05` | approval and external smoke record | human approver + release owner | `artifacts/ft-007/verify/chk-05/` | `CHK-05` |

### Current Delivery Evidence

- `EVID-01` pass — go test ./..., go vet ./..., go run ./cmd/code-converge config; new module, command and config output work locally.
- `EVID-02` pass — repository identity inventory shows no unintended old identity outside explicit FT-007 rename facts and the current pre-rename issue URL.
- `EVID-03` pass — existing workflow, event, Codex, app and config tests pass unchanged apart from renamed identifiers.
- `EVID-04` pass — make dist, sh -n scripts/install.sh, GitHub Actions smoke paths and make docs-lint pass; Linux binary execution is reserved for CI.
- `EVID-05` pending — external repository rename and reachability smoke are owned by dapi and are not performed from this worktree.
