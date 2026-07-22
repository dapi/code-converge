---
title: "FT-012: Self-update command"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem/verify owner для интерактивной и unattended команды обновления установленного code-converge из совместимого GitHub Release."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../product/context.md
  - ../../prd/PRD-001-code-converge-cli.md
  - ../../ops/release.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/12
status: active
delivery_status: planned
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-012: Self-update command

## What

### Problem

Installed users can install a verified GitHub Release through `scripts/install.sh`, but the CLI has no built-in, safe path to discover and install the latest compatible release. The requested `code-converge update` command must not replace the running executable until it has identified a newer compatible release, shown its release notes (or URL), received affirmative confirmation unless `--yes` is supplied, downloaded the matching archive and `SHA256SUMS`, and verified the checksum.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Safe no-change paths | No CLI updater exists | Every unsuccessful or declined path preserves the installed executable byte-for-byte | Deterministic fake-endpoint/downloader and replacement-failure tests |
| `MET-02` | Compatible verified update | Installation is external-script only | A newer compatible release is verified and atomically replaces the running executable; the replacement reports its target version | Archive/checksum and post-replacement fake-binary tests |
| `MET-03` | Contract completeness | No update command contract | README and feature contract specify invocation, confirmation, stdout/stderr, exit behavior, permissions/location and recovery | Docs lint plus semantic contract review |

### Scope

- `REQ-01` Add `code-converge update` with `--yes` and `-y`; do not add an `upgrade` alias.
- `REQ-02` Detect the running executable version and supported host targets: macOS/Linux on AMD64/ARM64.
- `REQ-03` Read latest stable metadata from `dapi/code-converge`, reject malformed/non-semantic version metadata, and do not select prereleases or downgrades.
- `REQ-04` When a newer compatible release exists, show its target version and release notes, or its release URL if notes are absent; default interaction requires exact affirmative `y`/`yes` confirmation.
- `REQ-05` With `--yes`/`-y`, run without reading confirmation input so it is usable by unattended scripts.
- `REQ-06` Download the matching release archive and `SHA256SUMS`, verify the archive checksum using the release format already consumed by `scripts/install.sh`, then atomically replace only the running executable.
- `REQ-07` Preserve the existing executable byte-for-byte on unsupported target, malformed metadata/version, network or download failure, missing asset/checksum, checksum mismatch, rejected confirmation, or replacement/permission failure.
- `REQ-08` Update the root README with public invocation, interaction/non-interaction, permissions/installation-location and recovery guidance; update dependent Memory Bank documents only where their current-state interpretation changes.

### Non-Scope

- `NS-01` Package-manager channels, updater daemon, background/automatic checks at normal startup, prerelease selection, downgrade support or an `upgrade` alias.
- `NS-02` Changing current workflow behavior outside the new `update` subcommand.
- `NS-03` Replacing an executable other than the running `code-converge` binary.
- `NS-04` Inventing a new release source, checksum format or host matrix beyond the existing GitHub Release/install-script contract.

### Constraints / Assumptions

- `ASM-01` `scripts/install.sh` and the release workflow establish the supported matrix, latest-release endpoint, archive name and `SHA256SUMS` release format.
- `ASM-02` The root README is the public CLI, stdout/stderr and exit-code contract owner; this package defines the required delta and implementation must update that owner atomically.
- `CON-01` The normal interactive command must never mutate the executable before affirmative confirmation; every unsuccessful path preserves it byte-for-byte.
- `CON-02` The replacement occurs only after download and checksum verification, and must be atomic from the user's observable perspective.
- `CON-03` This command changes a public CLI contract, invokes a GitHub Release integration and changes a release-artifact/rollback path; it requires explicit solution design and a high-risk validation profile.
- `DEC-01` Resolved in `decision-log.md#d-01`: current version and declined/default confirmation return `0`; malformed metadata, unsupported host and all network/download/checksum/replacement failures return existing operational code `2`. Statuses, release notes and confirmation prompt use stdout; diagnostics use stderr.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | New public CLI and confirmation contracts, GitHub Release integration, checksum verification, atomic replacement, permission failure and rollback semantics require selected solution and failure-mode reasoning. | `design.md` after `DEC-01` is resolved |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- |
| `decision-log.md` | selected | User requires FPF closure and review-improve provenance; it holds routing and the blocking human gate. | feature-local provenance; accepted facts are promoted here or to `design.md` |
| `design.md`, `implementation-plan.md` | selected | Public CLI/integration/replacement contract needs selected solution and an executable high-risk plan. | `design.md` / `implementation-plan.md` |
| Separate contract/C4/sequence artifacts | omitted | C1 context, connector contract and failure/order semantics fit in `design.md`; separate files would duplicate canonical facts. | `design.md` |
| Feature-local use case | omitted | The command is one bounded delivery scenario; the current project has no reusable updater use case and no evidence yet that another feature needs one. | `SC-*` in this brief |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `high-risk` | The change combines a public CLI/integration contract with replacement of the installed release artifact, permission/recovery semantics and byte-preservation safety. It also inherits release/deployment obligations; `high-risk` wins under the profile composition rule. | `none`; no downgrade requested |

## Verify

### Exit Criteria

- `EC-01` The root README specifies `update`, `--yes`/`-y`, confirmation input, stdout/stderr and exit contracts: current/declined return `0`; operational failures return `2`.
- `EC-02` Current release, rejected/default confirmation and every specified unsuccessful path leave the old executable byte-for-byte intact.
- `EC-03` `--yes` updates without reading confirmation input; the interactive path updates only after `y`/`yes`.
- `EC-04` A newer compatible release whose archive checksum matches is atomically installed at the running executable path, and the replacement returns the target version for `--version`.
- `EC-05` Target matrix, archive/checksum handling, documentation convergence, release-asset smoke coverage and required repository checks pass.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01`, `REQ-04`, `REQ-05` | `ASM-02`, `DEC-01` | `EC-01`, `EC-03`, `SC-02`, `SC-03` | `CHK-01`, `CHK-04` | `EVID-01`, `EVID-04` |
| `REQ-02`, `REQ-03`, `REQ-06` | `ASM-01`, `CON-02` | `EC-04`, `EC-05`, `SC-04` | `CHK-02`, `CHK-03`, `CHK-05` | `EVID-02`, `EVID-03`, `EVID-05` |
| `REQ-07` | `CON-01`, `CON-02` | `EC-02`, `SC-01`, `SC-05` | `CHK-02`, `CHK-03` | `EVID-02`, `EVID-03` |
| `REQ-08` | `ASM-02` | `EC-01`, `EC-05` | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` |

### Acceptance Scenarios

- `SC-01` A user on a supported host already runs the latest stable version; the command reports that version and makes no download or file modification.
- `SC-02` A newer compatible release has notes; the interactive command displays target version and notes, then a response other than `y`/`yes` leaves the executable unchanged.
- `SC-03` The same newer release is confirmed with `y` or `yes`, or invoked with `--yes`/`-y`; only an affirmative path continues, and `--yes` reads no confirmation input.
- `SC-04` The affirmative path downloads the matching existing-matrix archive and `SHA256SUMS`, verifies its checksum, atomically replaces the running executable and the replacement reports its target version.
- `SC-05` Unsupported host, malformed metadata/version, download failure, missing archive/checksum, checksum mismatch, replacement failure or permission failure reports the public failure contract and preserves the old executable byte-for-byte.

### Negative Coverage

- `NEG-01` Prerelease, equal/older version, invalid semantic version and missing target asset are never selected for installation.
- `NEG-02` A default/negative confirmation and every pre-replacement error create no destination mutation; a failed replacement does not leave a partial executable.
- `NEG-03` `--yes` must not block on or consume stdin.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `EC-01`, `EC-03`, `NEG-03` | Deterministic app/CLI tests with fake stdin and writers | Subcommand parsing, aliases, prompt/no-prompt, output routing and approved exit behavior hold. | `artifacts/ft-012/verify/chk-01/` |
| `CHK-02` | `EC-02`, `SC-01`, `SC-02`, `SC-05`, `NEG-01` | Fake release endpoint/downloader, fake executable and byte comparison | No-change/error paths preserve the original bytes and reject incompatible/malformed releases. | `artifacts/ft-012/verify/chk-02/` |
| `CHK-03` | `EC-04`, `SC-03`, `SC-04`, `SC-05`, `NEG-02` | Deterministic archive, `SHA256SUMS`, replacement and permission-failure tests | Only verified affirmed release atomically replaces; all failure paths preserve the original. | `artifacts/ft-012/verify/chk-03/` |
| `CHK-04` | `EC-01`, `REQ-08` | `make docs-lint` plus semantic README/Memory Bank review | Public contract and dependent interpretations converge. | `artifacts/ft-012/verify/chk-04/` |
| `CHK-05` | `EC-05` | Release-asset smoke coverage plus `go test ./...`, `go vet ./...`, `make docs-lint`, `git diff --check` | Release matrix/checksum smoke and full local validation pass; hosted required CI is recorded at closure. | `artifacts/ft-012/verify/chk-05/` |

### Evidence

- `EVID-01` CLI parsing, prompt/no-prompt and output/exit contract test log.
- `EVID-02` No-change and malformed/incompatible release preservation test log.
- `EVID-03` Verified replacement, checksum and replacement/permission-failure test log.
- `EVID-04` Documentation lint output and reviewed public-contract diff.
- `EVID-05` Release-asset smoke, full local verification and required CI reference.

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | CLI contract test log | test runner | `artifacts/ft-012/verify/chk-01/` | `CHK-01` |
| `EVID-02` | No-change/error preservation log | test runner | `artifacts/ft-012/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Verified replacement and failure-recovery log | test runner | `artifacts/ft-012/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Docs lint and semantic contract review | test runner/reviewer | `artifacts/ft-012/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Smoke/full verification logs and CI reference | test runner/CI | `artifacts/ft-012/verify/chk-05/` | `CHK-05` |
