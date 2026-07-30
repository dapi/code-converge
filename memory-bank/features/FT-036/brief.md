---
title: "FT-036: Discoverable CLI Help"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem, scope, validation profile и verify contract для discoverable CLI help из issue #36."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../../../README.md
status: active
delivery_status: in_progress
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-036: Discoverable CLI Help

## What

### Problem

Root help currently exposes only a usage line, and `config --help` and `update --help` are treated as invalid invocations. Operators cannot discover commands, flags, configuration entry points, or command-specific syntax without reading the README.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Interactive CLI discoverability | Root has one usage line; subcommand help exits 2 | Root and both supported subcommands provide stable successful help | Focused app tests and public README contract |

### Scope

- `REQ-01` `-h` and `--help` render concise root help to stdout and exit 0, including usage, `config` and `update` synopses, global options, and a README/configuration pointer.
- `REQ-02` `config --help` and `update --help` render command-specific stdout help and exit 0; update documents `--yes` and `-y`.
- `REQ-03` Every help invocation returns before configuration loading, diagnostic session logging, workflow start, Codex invocation, and self-update.
- `REQ-04` Focused tests and the public CLI contract describe the help text and exit semantics without changing machine-readable workflow output.

### Non-Scope

- `NS-01` Changing workflow, configuration-command, update-command, or machine-readable event semantics outside help invocations.
- `NS-02` Adding commands, changing flag values/defaults, loading configuration to build help, or performing a self-update check.

### Constraints / Assumptions

- `ASM-01` The existing `flag` definitions are the authoritative inventory of global options.
- `CON-01` Public CLI output must remain concise and stable; README remains the complete configuration reference.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The public CLI/help and stdout contract changes; command dispatch must preserve non-help side-effect boundaries. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| Separate use-case or runtime-surface artifact | omitted | One CLI entrypoint and three help paths are fully traceable in the brief and design. | `none` |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `standard` | Executable public CLI contract and exit semantics change. No security, data, integration, release, or rollout trigger applies. | `none` |

## Verify

### Exit Criteria

- `EC-01` All three help surfaces satisfy their documented output and exit contract without side effects.
- `EC-02` Normal workflow and machine-readable output remain unaffected.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `ASM-01`, `CON-01` | `EC-01`, `SC-01` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `REQ-02` | `CON-01` | `EC-01`, `SC-02` | `CHK-01`, `CHK-03` | `EVID-01`, `EVID-03` |
| `REQ-03` | `CON-01` | `EC-01`, `SC-03` | `CHK-02` | `EVID-02` |
| `REQ-04` | `CON-01` | `EC-02`, `SC-04` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` |

### Acceptance Scenarios

- `SC-01` An operator runs `code-converge --help` or `-h` and receives root usage, command synopses, grouped global options, and a README/configuration pointer with exit 0.
- `SC-02` An operator runs `code-converge config --help` or `code-converge update --help` and receives the valid syntax and purpose; update lists both confirmation aliases, with exit 0.
- `SC-03` A help invocation with fake runner and updater performs neither a runner invocation nor self-update.
- `SC-04` A normal invalid/workflow invocation retains its pre-existing event and exit behavior.

### Negative Coverage

- `NEG-01` Invalid `update` arguments still exit operationally and do not become a successful help path.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `SC-01`, `SC-02`, `NEG-01` | Focused `internal/app` tests | Exact required help fragments and exit codes pass | `go test ./internal/app` |
| `CHK-02` | `SC-03` | Fake-runner/updater tests | No configuration/workflow/update side effect on help | `go test ./internal/app` |
| `CHK-03` | `SC-01`, `SC-02`, `SC-04` | `go test ./...`, `go vet ./...`, `git diff --check` | All suites pass and diff is valid | command output |
| `CHK-04` | `REQ-04` | `make docs-lint` | Public docs links and governed frontmatter pass | command output |

### Test matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | `go test ./internal/app` |
| `CHK-02` | `EVID-02` | `go test ./internal/app` |
| `CHK-03` | `EVID-03` | local command output and CI run |
| `CHK-04` | `EVID-04` | `make docs-lint` |

### Evidence

- `EVID-01` Focused app-test output covering root and subcommand help.
- `EVID-02` Test assertions showing help bypasses runner/updater.
- `EVID-03` Full local suite, vet, diff-check, and required CI result.
- `EVID-04` Documentation-lint output and README contract review.
