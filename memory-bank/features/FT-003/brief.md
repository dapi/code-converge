---
title: "FT-003: Automated GitHub Releases"
doc_kind: feature
doc_function: canonical
purpose: "Canonical brief для автоматической публикации проверенных code-converge-артефактов в GitHub Releases."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../../ops/release.md
  - ../../../README.md
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
canonical_for:
  - ft_003_problem_space
  - ft_003_validation_profile
  - ft_003_verify_contract
---

# FT-003: Automated GitHub Releases

## What

### Problem

The repository can build deterministic archives but has no official publication path: CI discards the archives and operators cannot obtain a versioned GitHub Release. The requested baseline is the release lifecycle used by `dapi/start-issue`: local semantic-version preparation followed by tag-triggered CI publication.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Automated release coverage | No workflow, tag, or hosted release | A valid pushed `vX.Y.Z` tag publishes all four archives and `SHA256SUMS` after verification | Workflow inspection and first approved release run |
| `MET-02` | Release identity consistency | Caller supplies arbitrary `VERSION` | Tag, repository `VERSION`, archive names, and GitHub Release version agree | Script tests and workflow identity gate |

### Scope

- `REQ-01` Add a tag-triggered GitHub Actions workflow that verifies, builds, checksum-validates, smoke-tests, and publishes the supported archive matrix.
- `REQ-02` Add semantic-version and changelog preparation commands equivalent to the `start-issue` patch/minor/major workflow.
- `REQ-03` Document the official GitHub Release channel, approval boundary, assets, verification, and rollback procedure.
- `REQ-04` Embed the release version in binaries and expose it through `code-converge --version`; provide a checksum-verifying one-line installer for supported targets.

### Non-Scope

- `NS-01` Package-manager publication, signing, self-update behavior, or a server deployment.
- `NS-02` Automatically choosing release significance, pushing a tag without an operator command, or publishing a release during implementation of this feature.

### Constraints / Assumptions

- `ASM-01` The existing deterministic `tools/build-dist` matrix remains the artifact authority.
- `CON-01` Publication may occur only from an exact semantic-version tag whose version matches the repository `VERSION` file.
- `CON-02` A failed repository check, checksum, or Linux AMD64 smoke must prevent publication.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The feature creates a privileged CI publication path, version identity contract, and operational rollout/backout flow. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `design.md` | selected | CI trust boundary and release failure semantics require an explicit solution owner. | `design.md` |
| `implementation-plan.md` | selected | Workflow, scripts, tests, and docs require coordinated verification. | `implementation-plan.md` |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `release-deployment` | The primary surfaces are release artifacts, privileged CI publication, versioning, and rollback. No separate data/security/concurrency high-risk trigger is introduced. | `none` |

## Verify

### Exit Criteria

- `EC-01` A valid version tag can publish exactly the documented assets only after all gates pass.
- `EC-02` Invalid/mismatched versions and failed checks cannot create a GitHub Release.
- `EC-03` Operators have tested preparation commands and an explicit publish/rollback runbook.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `ASM-01`, `CON-01`, `CON-02` | `EC-01`, `EC-02`, `SC-01`, `NEG-01` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `REQ-02` | `CON-01` | `EC-02`, `EC-03`, `SC-02`, `NEG-02` | `CHK-01` | `EVID-01` |
| `REQ-03` | `NS-01`, `NS-02` | `EC-03`, `SC-03` | `CHK-03` | `EVID-03` |
| `REQ-04` | `ASM-01`, `CON-01` | `EC-01`, `SC-04`, `NEG-03` | `CHK-01`, `CHK-02`, `CHK-03` | `EVID-01`, `EVID-02`, `EVID-03` |

### Acceptance Scenarios

- `SC-01` Pushing an approved `vX.Y.Z` tag matching `VERSION` runs full verification, produces the four supported archives plus `SHA256SUMS`, validates them, smoke-tests Linux AMD64, and creates the GitHub Release.
- `SC-02` Patch/minor/major preparation requires a clean worktree and non-empty Unreleased changelog, updates identity/changelog, verifies/builds, and creates a local release commit and annotated tag.
- `SC-03` Documentation tells the operator how to prepare, publish, verify, and replace a bad release.
- `SC-04` A release binary prints `code-converge vX.Y.Z` with `code-converge --version`, and the one-line installer downloads the matching archive, verifies its checksum, and installs it under the documented prefix.

### Negative Scenarios

- `NEG-01` Non-semver or mismatched tags, failed checks, bad checksums, or failed smoke stop before `gh release create`.
- `NEG-02` Dirty worktrees, invalid current versions, duplicate tags, or empty Unreleased sections are rejected by release preparation.
- `NEG-03` Unsupported platforms, malformed release versions, missing checksums, and checksum mismatches stop installation before replacing the local binary.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | scripts and version identity | `go test ./...`, `bash -n scripts/bump-version scripts/prepare-release` | Tests and syntax checks pass | changed tests and command output |
| `CHK-02` | artifacts and repository gates | `make verify`, two `VERSION=0.0.0 make dist` builds, checksum comparison, Linux native smoke where available | Verification is green and artifacts are reproducible | command output and `dist/SHA256SUMS` |
| `CHK-03` | docs and workflow integrity | `make docs-lint`, `git diff --check`, semantic workflow read-through | Documentation and workflow are internally consistent | command output and reviewed diff |

### Test matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | changed automated tests and local output |
| `CHK-02` | `EVID-02` | local verify/dist output; hosted release run after approved tag push |
| `CHK-03` | `EVID-03` | docs-lint/diff output and final diff |

### Evidence

- `EVID-01` Automated bump-version tests and shell syntax check.
- `EVID-02` Repository verification, reproducible checksum manifest, artifact checksum and smoke evidence; the first live GitHub run remains a rollout gate.
- `EVID-03` Documentation lint and reviewed release workflow/docs diff.

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | test/check output | local or CI runner | tests plus command output | `CHK-01` |
| `EVID-02` | artifact manifest and run | build helper / GitHub Actions | `dist/SHA256SUMS` and approved release run URL | `CHK-02` |
| `EVID-03` | lint and review result | local runner / code-converge | command output and feature docs | `CHK-03` |
| `EVID-04` | version/installer evidence | local binary and shell checks | `code-converge --version`, `sh -n`, and checksum fixture output | `CHK-01`, `CHK-02`, `CHK-03` |

### Evidence Results

| Evidence ID | Concrete carriers | Result |
| --- | --- | --- |
| `EVID-01` | `scripts/release_scripts_test.go`, `make verify`, shell syntax and dirty-worktree rejection checks | Pass locally. |
| `EVID-02` | `tools/build-dist`, two independent `0.0.0` builds, matching `SHA256SUMS`, [Release v0.1.0](https://github.com/dapi/code-converge/releases/tag/v0.1.0), [Release workflow](https://github.com/dapi/code-converge/actions/runs/29860631840), [Verify workflow](https://github.com/dapi/code-converge/actions/runs/29860631895) | Pass: hosted release published all five assets; checksums and Linux AMD64 smoke passed. |
| `EVID-03` | `actionlint` v1.7.7, `make docs-lint`, `git diff --check`, final semantic diff review | Pass locally. |
| `EVID-04` | `internal/app/app_test.go`, version ldflags, `scripts/install.sh` shell syntax, published latest installer smoke | Pass: native archive and one-line installer both report `code-converge v0.1.0`. |
