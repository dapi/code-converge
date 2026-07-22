---
title: "FT-009: Human-readable progress logging and liveness"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem/verify owner для явного human progress format, совместимого structured output и bounded liveness indicators."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../product/context.md
  - ../../prd/PRD-001-code-converge-cli.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/9
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-009: Human-readable progress logging and liveness

## What

### Problem

The current workflow stdout contract is stable and machine-readable, but operators must interpret repetitive `key=value` records, millisecond durations and zero-valued counters. Long-running Codex-backed stages also provide no bounded signal that the process is alive. Improving operator readability must not silently change the semantic format based on TTY state, break existing structured consumers, contaminate redirected output with terminal controls, or expose raw Codex output.

### Outcome

| Metric ID | Metric | Baseline | Target | Measurement method |
| --- | --- | --- | --- | --- |
| `MET-01` | Human rendering coverage | No workflow human format | Every existing workflow event/result and terminal path has a deterministic human rendering or an explicit omission rule | Table-driven renderer golden tests against the README catalog |
| `MET-02` | Structured compatibility | Current stable `key=value` stream | Explicit structured mode remains byte-contract compatible for existing events when heartbeat is disabled | Existing and new `kv` golden tests |
| `MET-03` | Liveness safety | No liveness output | TTY transient updates or explicit newline heartbeats stop on completion, failure and cancellation with no late/interleaved writes | Deterministic clock, terminal and race-oriented tests |
| `MET-04` | Output isolation | Raw Codex stdout captured | Raw Codex output remains absent from workflow stdout in all formats | Adapter/workflow integration fixtures |

### Scope

- `REQ-01` Provide `human` and structured `kv` workflow log formats, with `human` as the built-in default and no TTY-selected semantic format.
- `REQ-02` Specify and implement human rendering for every current workflow event/result, including review phase/cycle context, non-zero severity summaries, readable durations, finalization steps and every terminal exit path.
- `REQ-03` Preserve the current stable `key=value` event stream for automation and compatibility when structured mode is selected and heartbeat is disabled.
- `REQ-04` Provide an in-place elapsed-time liveness line with full-line color shimmer for long-running stages when human mode writes to an interactive terminal.
- `REQ-05` Keep non-TTY output newline-safe and ANSI-free by default, and allow an explicit bounded heartbeat for callers that need liveness in redirected output or CI.
- `REQ-06` Stop liveness promptly and serialize progress writes across completion, failure, cancellation and output-write failure paths.
- `REQ-07` Preserve stderr diagnostics and raw Codex-output isolation.
- `REQ-08` Add the public CLI/config/environment contract and update the root README plus dependent Memory Bank documents.

### Non-Scope

- `NS-01` Streaming or forwarding raw Codex output.
- `NS-02` Percentage-complete estimates, spinner glyphs, stage prediction or telemetry backends.
- `NS-03` Changes to workflow state transitions, retry/recovery budgets, review classification, finalization verdicts or exit-code meanings.
- `NS-04` Adding structured heartbeat events or changing the schema of existing `kv` records beyond explicit selection of that format.
- `NS-05` Persistent metrics storage, dashboards, tracing, hosted log aggregation or a new GUI/TUI.

### Constraints / Assumptions

- `ASM-01` The root README's current event catalog, field semantics and exit codes are the baseline public contract; source code owns implementation details.
- `ASM-02` The existing configuration resolver supplies the project-wide source precedence and naming pattern for new settings.
- `CON-01` Semantic format selection must be deterministic; the built-in `human` default and every explicit override are independent of TTY state, which may affect only liveness/color mechanics within human mode.
- `CON-02` Non-TTY workflow stdout must contain no ANSI/control sequences unless a future separately specified contract changes that rule.
- `CON-03` Detailed diagnostics remain on stderr, and raw Codex output must never be forwarded to workflow progress output.
- `CON-04` Liveness is an elapsed waiting indicator, never a completion-percentage claim.
- `CON-05` No new reusable cross-feature architecture policy is required; the change stays within the existing CLI/config/workflow/event component boundaries.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | The feature changes public CLI/config/stdout contracts, adds concurrent liveness behavior and requires compatibility, failure and cancellation decisions. | `design.md` |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- | --- |
| `decision-log.md` | selected | The user requires FPF closure and review-improve provenance for material contract decisions. | feature-local provenance; accepted facts are promoted to `brief.md` or `design.md` |
| Separate contract/C4/sequence artifacts | omitted | The public contract belongs in the root README and the bounded C3/concurrency view fits in `design.md`; separate files would duplicate ownership. | `design.md` and `../../../README.md` |
| Feature-local use case | omitted | The feature improves observability of the existing workflow but does not introduce a new stable workflow scenario. | existing PRD/workflow owners |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `high-risk` | Public stdout/config compatibility is consumer-sensitive, and the liveness worker introduces concurrency, ordering, cancellation and shared-writer failure semantics. The profile includes all affected unit/contract/e2e surfaces and explicit approval before risk-bearing execution. | `none`; no downgrade requested |

## Verify

### Exit Criteria

- `EC-01` `human` and `kv` are explicitly selectable through the documented CLI/config/environment contract, with the documented default and precedence.
- `EC-02` Every current event/result and terminal path matches its canonical human rendering; durations contain no milliseconds and zero severity counters are omitted.
- `EC-03` Existing `kv` records remain machine-safe and compatible when heartbeat is disabled.
- `EC-04` Human TTY liveness updates in place with elapsed time and full-line shimmer, while no-color behavior retains the timer without color.
- `EC-05` Human non-TTY output is ANSI-free and silent between permanent records unless heartbeat is explicitly enabled; enabled heartbeats are bounded and newline-safe.
- `EC-06` Completion, failure, cancellation and write errors stop the liveness worker before subsequent permanent output and do not produce races or late writes.
- `EC-07` Diagnostics remain on stderr and raw Codex output is absent from workflow stdout.
- `EC-08` Root README and dependent project documents converge on the new public contract and all required checks pass.

### Traceability matrix

| Requirement ID | Problem refs | Acceptance refs | Checks | Evidence IDs |
| --- | --- | --- | --- | --- |
| `REQ-01` | `ASM-01`, `CON-01` | `EC-01`, `SC-01`, `SC-02` | `CHK-01`, `CHK-05` | `EVID-01`, `EVID-05` |
| `REQ-02` | `ASM-01` | `EC-02`, `SC-01`, `SC-05` | `CHK-02` | `EVID-02` |
| `REQ-03` | `ASM-01`, `CON-01` | `EC-03`, `SC-02` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `REQ-04` | `CON-04` | `EC-04`, `SC-03` | `CHK-03` | `EVID-03` |
| `REQ-05` | `CON-02`, `CON-04` | `EC-05`, `SC-04` | `CHK-03` | `EVID-03` |
| `REQ-06` | `CON-02` | `EC-06`, `SC-06` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` |
| `REQ-07` | `CON-03` | `EC-07`, `SC-06` | `CHK-04` | `EVID-04` |
| `REQ-08` | `ASM-02` | `EC-01`, `EC-08`, `SC-01`, `SC-02` | `CHK-01`, `CHK-05`, `CHK-06` | `EVID-01`, `EVID-05`, `EVID-06` |

### Acceptance Scenarios

- `SC-01` An operator uses the default format for a normal findings/fix/clean/finalize run and receives concise permanent lines with non-zero severities, readable durations, finalization steps and `Done`.
- `SC-02` An automation caller explicitly selects `kv` mode and receives the existing stable event records without heartbeat additions.
- `SC-03` Human mode on a TTY shows one updating elapsed line during each Codex-backed stage, clears it before permanent completion/diagnostic output and respects no-color behavior.
- `SC-04` Human mode with redirected output emits no implicit liveness or ANSI sequences; an explicit heartbeat emits bounded newline records at the configured interval.
- `SC-05` Review findings exhaustion, operational failure and exhausted CI recovery each render the documented terminal line and preserve exit codes `1`, `2` and `3` respectively.
- `SC-06` Stage completion racing a timer tick, context cancellation and output-write failure leave no liveness goroutine or late write and preserve the applicable terminal semantics.

### Negative Coverage

- `NEG-01` Invalid log format, heartbeat duration below the permitted minimum, negative duration, invalid color value or heartbeat combined with `kv` fails configuration before a Codex invocation.
- `NEG-02` TTY/control/color capabilities never introduce ANSI bytes into non-TTY output.
- `NEG-03` Raw Codex stdout fixtures and free-form diagnostics never appear in workflow stdout.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `EC-01`, `NEG-01` | `go test ./internal/config ./internal/app` | New sources, precedence, defaults, `config` display and invalid values are deterministic. | `artifacts/ft-009/verify/chk-01/` |
| `CHK-02` | `EC-02`, `EC-03` | `go test ./internal/event ./internal/workflow` | Golden catalog covers both renderers and every workflow terminal path; baseline `kv` records remain compatible. | `artifacts/ft-009/verify/chk-02/` |
| `CHK-03` | `EC-04`, `EC-05`, `EC-06`, `NEG-02` | Deterministic terminal/clock tests plus `go test -race ./internal/event ./internal/workflow` | Timer, shimmer, heartbeat, clearing and stop ordering pass without races or late writes. | `artifacts/ft-009/verify/chk-03/` |
| `CHK-04` | `EC-06`, `EC-07`, `NEG-03` | `go test ./internal/codex ./internal/workflow ./internal/app` with cancellation and failing-writer fixtures | Failures remain bounded; stderr/stdout isolation and raw-output capture hold. | `artifacts/ft-009/verify/chk-04/` |
| `CHK-05` | `EC-01`, `EC-08` | `make docs-lint` and semantic contract read-through | README, config table, event catalog and dependent Memory Bank docs agree. | `artifacts/ft-009/verify/chk-05/` |
| `CHK-06` | `EC-08` | `go test ./... && go vet ./... && git diff --check` | Full local repository verification is green. | `artifacts/ft-009/verify/chk-06/` |

### Test matrix

| Check ID | Evidence IDs | Evidence path |
| --- | --- | --- |
| `CHK-01` | `EVID-01` | `artifacts/ft-009/verify/chk-01/` |
| `CHK-02` | `EVID-02` | `artifacts/ft-009/verify/chk-02/` |
| `CHK-03` | `EVID-03` | `artifacts/ft-009/verify/chk-03/` |
| `CHK-04` | `EVID-04` | `artifacts/ft-009/verify/chk-04/` |
| `CHK-05` | `EVID-05` | `artifacts/ft-009/verify/chk-05/` |
| `CHK-06` | `EVID-06` | `artifacts/ft-009/verify/chk-06/` |

### Evidence

- `EVID-01` Configuration/app test log and effective-setting samples.
- `EVID-02` Human/kv golden renderer and terminal-path test log.
- `EVID-03` Deterministic liveness plus race-test log.
- `EVID-04` Cancellation, write-failure and output-isolation test log.
- `EVID-05` Documentation lint output and reviewed contract diff.
- `EVID-06` Full test, vet and diff-check log plus required CI URL at delivery closure.

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | Configuration/app test log and samples | test runner | `artifacts/ft-009/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Renderer/workflow golden test log | test runner | `artifacts/ft-009/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Liveness and race-test log | test runner | `artifacts/ft-009/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Cancellation/write/output-isolation log | test runner | `artifacts/ft-009/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Docs lint and semantic review record | test runner/reviewer | `artifacts/ft-009/verify/chk-05/` | `CHK-05` |
| `EVID-06` | Full local verification and CI references | test runner/CI | `artifacts/ft-009/verify/chk-06/` | `CHK-06` |

### Execution Evidence Status

| Evidence ID | Status | Concrete carrier |
| --- | --- | --- |
| `EVID-01` | pass | `TestLoggingConfiguration`, `TestLoggingConfigurationPrecedence`, `TestInvalidLoggingConfiguration`, app startup/config tests; `go test ./internal/config ./internal/app` |
| `EVID-02` | pass | `TestHumanEventCatalog`, `TestHumanHappyPath`, `TestHumanTerminalPaths` and unchanged structured event tests; `go test ./internal/event ./internal/workflow` |
| `EVID-03` | pass | heartbeat/transient/shimmer/stop tests plus `go test -race ./internal/event ./internal/workflow` |
| `EVID-04` | pass | cancellation, liveness/permanent writer failure and human raw-output isolation tests across app/workflow/codex/runner suites |
| `EVID-05` | pass | root README and dependent Memory Bank convergence; `make docs-lint` |
| `EVID-06` | pass | `go test ./...`, `go test -race ./internal/event ./internal/workflow`, `go vet ./...`, `make docs-lint`, `git diff --check`; PR [#13](https://github.com/dapi/code-converge/pull/13), required [Verify run](https://github.com/dapi/code-converge/actions/runs/29901470990) |
