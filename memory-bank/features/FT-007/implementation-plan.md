---
title: "FT-007: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Execution plan for the clean-break rename to code-converge across source, public contracts, documentation, CI, distribution and release references."
derived_from:
  - brief.md
  - design.md
status: archived
audience: humans_and_agents
must_not_define:
  - ft_007_scope
  - ft_007_selected_design
  - ft_007_acceptance_criteria
  - ft_007_validation_profile
---

# FT-007: Implementation Plan

## Цель текущего плана

Rename the local project identity from code-converge to code-converge as a clean break. Preserve workflow behavior, exit codes, stdout events and Codex integration. The GitHub repository itself is renamed separately by dapi; this change updates repository references for the new canonical URL.

## Grounding / Support References

| Document | Role | Facts reused |
| --- | --- | --- |
| brief.md | canonical problem, scope and verify owner | REQ-*, SC-*, CHK-*, EVID-*, clean-break constraint |
| design.md | canonical solution owner | SOL-*, SD-*, CTR-*, INV-*, FM-*, RB-* |
| decision-log.md | provenance | DL-01–DL-04, review cycles and owner decisions |
| ../../../README.md | public CLI contract | command, config, release and installer wording |

## Current State / Reference Points

| Path | Current role | Planned change |
| --- | --- | --- |
| go.mod, Go imports | module identity | github.com/dapi/code-converge |
| cmd/code-converge | executable source | cmd/code-converge, binary code-converge |
| internal/config | config precedence and paths | .code-converge, CODE_CONVERGE_* |
| scripts/install.sh, tools/build-dist | release/install surfaces | new repository, archive, binary and environment names |
| README.md, Memory Bank, CHANGELOG | public/project docs | current identity and links |
| tests and Makefile | executable contract evidence | new names; behavior assertions unchanged |

## Test Strategy

| Surface | Required evidence | Commands |
| --- | --- | --- |
| Go build/imports | clean build and all tests | go test ./..., go vet ./... |
| CLI/config | new command, flags, config paths/env names; old names not read | targeted app/config tests and go run ./cmd/code-converge config |
| Distribution/installer | archive and install names, shell syntax | make dist, sh -n scripts/install.sh, release script tests |
| Documentation | links, frontmatter and identity inventory | make docs-lint, markdown inventory, git diff --check |
| Workflow contract | events, exit codes and Codex arguments | existing workflow/codex test suites |

## Open Questions / Ambiguities

none. DL-03 and DL-04 record the owner decisions. The external repository rename is not performed by this worktree.

## Environment Contract

- Go toolchain and existing Make targets are available.
- No live Codex session, release publication or repository rename is required for local verification.
- dapi owns the external repository rename and later release rollout.

## Preconditions

| ID | Required state | Blocks |
| --- | --- | --- |
| PRE-01 | brief.md and design.md active; clean-break policy recorded | all steps |
| PRE-02 | Current branch remains based on origin/master until PR is opened | publication |

## Design Realization Mapping

| Solution refs | Steps | Checks | Evidence |
| --- | --- | --- | --- |
| SOL-01, SOL-02, SD-01, SD-02, CTR-01, INV-01 | STEP-01, STEP-02 | CHK-01, CHK-03 | EVID-01, EVID-03 |
| SOL-03, SOL-05, SD-03, SD-04, CTR-03 | STEP-02, STEP-03 | CHK-02, CHK-04 | EVID-02, EVID-04 |
| SOL-04, CTR-04, INV-03, RB-02 | STEP-04 | CHK-05 | EVID-05 |

## Порядок работ

| Step | Implements | Touchpoints | Verifies |
| --- | --- | --- | --- |
| STEP-01 | REQ-02, REQ-04, SOL-01, SOL-02 | rename executable/module/imports, CLI/config names and tests | CHK-01, CHK-03 |
| STEP-02 | REQ-03, REQ-05, SOL-02, SOL-03, SOL-05 | README, Memory Bank, prompts, Makefile, scripts, CI, release/archive references | CHK-02, CHK-04 |
| STEP-03 | REQ-06, SD-02, INV-01 | regression tests and identity inventory | CHK-02, CHK-03 |
| STEP-04 | REQ-01, REQ-07, SOL-04, RB-02 | PR evidence and handoff to dapi for external rename/release | CHK-05 |

## Approval Gates

- AG-01 dapi owns the external repository rename and release rollout; this plan does not perform those mutations.

## Checkpoints

| Checkpoint | Condition | Evidence |
| --- | --- | --- |
| CP-01 | new module, command, config and tests build together | EVID-01, EVID-03 |
| CP-02 | no unintended current old identity remains | EVID-02 |
| CP-03 | docs, distribution and installer converge | EVID-04 |
| CP-04 | PR published with green required CI and no conflicts | EVID-05 |

## Stop Conditions / Fallback

- Stop if a remaining code-converge occurrence is a current contract rather than history/generic terminology.
- Stop if tests show workflow/event/exit-code behavior changed.
- Do not rename the GitHub repository or publish releases from this worktree; hand those actions to dapi.

## Готово для приемки

All local implementation and documentation checks pass, the branch is pushed, a PR targets master, required CI is green, and the external rename/release handoff to dapi is explicit.
