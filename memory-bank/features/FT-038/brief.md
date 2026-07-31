---
title: "FT-038: YAML Configuration"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem, scope and verification contract for issue #38."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../../../README.md
status: active
delivery_status: active
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-038: YAML Configuration

## Scope

- `REQ-01` Project and user configuration resolve only from their respective `config.yaml` files, with CLI > project > user > environment > defaults precedence.
- `REQ-02` The strict typed YAML schema covers every existing file-configurable setting, rejects malformed, unknown, duplicate and invalid values with useful diagnostics, and never reads legacy files.
- `REQ-03` Prompt YAML values are file references resolved from their configuration directory; CLI and environment prompt paths keep their existing behavior.
- `REQ-04` `code-converge config`, README and the operational configuration contract accurately report and document YAML sources.
- `REQ-05` The clean-break configuration contract is released only in the next SemVer major version.

## Non-Scope

- `NS-01` Migrating, warning about, or otherwise supporting legacy per-setting files.
- `NS-02` Changing CLI flags, environment variable names, model profiles or their precedence.

## Design Requirement Decision

`Design required: yes` — this changes a public configuration file format and release compatibility contract.

## Validation Profile Decision

`standard` — the public configuration contract and parser behavior change; no high-risk persistence, security, integration or deployment trigger applies.

## Verify

- `SC-01` A complete project `config.yaml` supplies all settings, including prompt references, and `config` reports `project` sources.
- `SC-02` Conflicting CLI, project YAML, user YAML and environment values resolve in the documented order.
- `SC-03` Malformed YAML, an unknown key, a duplicate key, a nested mapping and an invalid typed value fail with actionable errors.
- `SC-04` Legacy per-setting values and prompt files have no effect.

| Check ID | Covers | Command | Evidence ID |
| --- | --- | --- | --- |
| `CHK-01` | `SC-01`–`SC-04` | `go test ./internal/config` | `EVID-01` |
| `CHK-02` | all requirements | `go test ./...`; `go vet ./...`; `git diff --check` | `EVID-02` |
| `CHK-03` | `REQ-04`, `REQ-05` | `make docs-lint` | `EVID-03` |

- `EVID-01` Focused deterministic configuration tests.
- `EVID-02` Full Go verification and clean diff check.
- `EVID-03` Documentation lint and release-note review.
