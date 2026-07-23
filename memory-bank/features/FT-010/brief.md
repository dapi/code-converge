---
title: "FT-010: Interactive agent-output view"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem, blocker, and verification owner for issue #10."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../engineering/validation-profiles.md
  - ../../engineering/architecture.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/10
status: active
delivery_status: in_progress
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-010: Interactive agent-output view

## What

### Problem

The workflow presents progress to a terminal, while raw output from the active Codex process is captured only after that process finishes and is deliberately excluded from workflow stdout. An operator cannot inspect the active agent output during a running stage without losing the workflow log.

### Outcome

An interactive terminal run can show the continuing workflow log and the active agent's arriving output in separate panes, without restarting or interrupting the workflow. Non-interactive execution retains its current machine-readable stdout behavior and does not expose raw agent output there.

### Scope

- `REQ-01` During an eligible interactive terminal run, pressing `i` opens a split view without restarting or interrupting the current workflow stage.
- `REQ-02` The upper pane continues to present workflow log records, including stage transitions and completion events, while the view is open.
- `REQ-03` The lower pane presents output from the currently active Codex process as it arrives.
- `REQ-04` A stage change clearly identifies or replaces the active agent stream so output from separate processes is not mixed.
- `REQ-05` The feature defines behavior for resize, pane scrolling, long lines, ANSI/control sequences, process completion/errors, and pressing `i` when no agent is active.
- `REQ-06` Terminal state is restored cleanly on exit, interruption, and panic; returning from the view leaves the invoking shell usable.
- `REQ-07` Non-interactive execution preserves the existing workflow stdout event format, does not require a TTY, and keeps agent output excluded from machine-readable workflow stdout.
- `REQ-08` Tests and documentation describe interactive-mode prerequisites and key bindings.

### Non-scope

- `NS-01` Changing the workflow state machine, review/fix/finalization policy, exit-code meanings, or structured event schema in non-interactive mode.
- `NS-02` Forwarding raw agent output into non-interactive or machine-readable workflow stdout.
- `NS-03` Changing Codex sandbox, approval, network, timeout, or agent prompt policy.

### Constraints and assumptions

- `CON-01` The root README is canonical for public CLI behavior and stdout/logging contracts. It currently describes human and `kv` log formats, interactive liveness on an interactive stdout terminal, and no ANSI controls in non-TTY output; it has no interactive agent-output-view contract.
- `CON-02` The current architecture keeps workflow policy, subprocess mechanics, and progress presentation separate. The runner captures process stdout/stderr after completion and must not forward raw Codex output to workflow stdout.
- `CON-03` The issue requires deterministic tests for input handling, pane updates, stage transitions, process completion, cleanup, plus a manual terminal smoke test.
- `CON-04` The issue explicitly identifies activation, close-key behavior, stream representation, scrollback, inactive-stage behavior, and terminal library/capabilities as open questions.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | This feature changes the public operator interaction model, requires a new terminal presentation boundary, changes subprocess-output handling, and needs explicit cleanup/failure semantics. | `design.md`, after `DEC-01` is resolved. |

## Artifact Routing Decision

| Artifact | Decision | Trigger / reason | Route / owner |
| --- | --- | --- |
| `decision-log.md` | selected | FPF reasoning and human-gate provenance must remain auditable without defining the public contract. | feature-local provenance |
| `design.md` | selected | It records the accepted terminal behavior, stream safety, and lifecycle semantics. | `design.md` |
| `implementation-plan.md` | selected | Execution needs coordinated runner, terminal runtime, presentation, test, and documentation sequencing. | `implementation-plan.md` |
| `ADR-001` | selected | The cross-platform terminal capability/raw-mode boundary is reusable beyond this feature. | `../../adr/ADR-001-interactive-terminal-runtime.md` |

## Validation Profile Decision

| Profile | Triggers / rationale | Downgrade approval |
| --- | --- | --- |
| `standard` | The executable operator behavior and terminal presentation change; non-interactive stdout compatibility must be proved. No available fact activates a security/auth, financial, persistent-data, migration, or cross-system trigger for `high-risk`. | `none` |

## Blocking decisions

| Decision ID | Question | Why it blocks | Owner / state |
| --- | --- | --- |
| `DEC-01` | What exact public contract governs eligibility/activation, closing, output representation, scrollback, inactive-stage behavior, and minimum terminal capabilities/library? | These choices determine observable operator behavior, terminal-safety semantics, public documentation, architecture boundaries, and the deterministic test matrix. | Resolved by `DL-03`; selected solution lives in `design.md` and terminal-runtime boundary in `ADR-001`. |

## Verify

### Exit criteria

- `EC-01` The accepted interactive contract opens and closes the view without interrupting a running stage and restores the terminal for subsequent shell output.
- `EC-02` The view keeps workflow records and current agent output distinct, current, and unmixed across stage transitions.
- `EC-03` Resize, scrolling, long-line, control-sequence, completion, error, inactive-stage, interruption, and panic behavior match the accepted contract.
- `EC-04` Non-interactive human and `kv` output preserve the pre-feature machine-readable stdout contract and exclude agent output.
- `EC-05` Documentation, design, implementation plan, tests, manual smoke evidence, and actual behavior agree.

### Acceptance scenarios

- `SC-01` Given an eligible terminal and an active Codex stage, when the operator presses `i`, the accepted split view opens and the stage continues running.
- `SC-02` Given the split view is open, when workflow records and active-agent output arrive, the respective panes update without mixing their content.
- `SC-03` Given the active stage changes or finishes, the lower pane follows the accepted replacement/identification and completion behavior without displaying a prior process as active.
- `SC-04` Given a resize, long line, control sequence, scrolling action, inactive stage, process error, interruption, or panic, the view follows the accepted terminal-safety and cleanup behavior.
- `SC-05` Given stdout is non-interactive or `--log-format=kv` is used, the run retains the existing stdout event contract and no raw agent output appears in that stream.

### Negative coverage

- `NEG-01` Opening, closing, or repainting the view must not restart, cancel, or alter workflow transitions.
- `NEG-02` Output from separate agent processes must not be silently coalesced as one active stream.
- `NEG-03` A non-interactive run must not emit terminal-control sequences or raw agent output into workflow stdout.
- `NEG-04` Cleanup failure must not leave terminal mode or subsequent shell output corrupted.

### Checks

| Check ID | Covers | How to check | Expected result | Evidence path |
| --- | --- | --- | --- | --- |
| `CHK-01` | `EC-01`–`EC-03`, `SC-01`–`SC-04`, `NEG-01`, `NEG-02`, `NEG-04` | Deterministic terminal-presentation, input, stream, transition, and cleanup tests using fakes. | Accepted interactive behavior and cleanup matrix passes without a live Codex session. | `artifacts/ft-010/verify/chk-01/` |
| `CHK-02` | `EC-04`, `SC-05`, `NEG-03` | Existing and new event/app/workflow golden and fake-executable tests. | Non-interactive stdout contract remains compatible and contains no agent stream. | `artifacts/ft-010/verify/chk-02/` |
| `CHK-03` | `EC-01`–`EC-03` | Manual terminal smoke procedure derived from the accepted contract. | Open, live updates, close, resize, completion, interrupt, and shell restoration are observed. | `artifacts/ft-010/verify/chk-03/` |
| `CHK-04` | `EC-05` | `make docs-lint` and semantic contract review. | Root README, architecture, feature docs, and evidence contract converge. | `artifacts/ft-010/verify/chk-04/` |
| `CHK-05` | `EC-05` | `go test ./... && go vet ./... && git diff --check`, plus required CI. | Required local verification and CI are green. | `artifacts/ft-010/verify/chk-05/` |

### Evidence contract

| Evidence ID | Artifact | Producer | Path contract | Reused by checks |
| --- | --- | --- | --- | --- |
| `EVID-01` | Deterministic interactive presentation and cleanup matrix | test runner | `artifacts/ft-010/verify/chk-01/` | `CHK-01` |
| `EVID-02` | Non-interactive stdout compatibility report | test runner | `artifacts/ft-010/verify/chk-02/` | `CHK-02` |
| `EVID-03` | Manual terminal smoke record | operator/reviewer | `artifacts/ft-010/verify/chk-03/` | `CHK-03` |
| `EVID-04` | Documentation lint and convergence record | test runner/reviewer | `artifacts/ft-010/verify/chk-04/` | `CHK-04` |
| `EVID-05` | Full local and CI verification record | test runner/CI | `artifacts/ft-010/verify/chk-05/` | `CHK-05` |

### Execution evidence status

| Evidence ID | Status | Concrete carrier |
| --- | --- | --- |
| `EVID-01` | pass | `go test ./internal/runner ./internal/terminal ./internal/event ./internal/codex ./internal/workflow ./internal/app` |
| `EVID-02` | pass | `go test ./...` including existing app/event/workflow non-interactive fixtures |
| `EVID-03` | pass | `TERM=xterm` pseudo-TTY smoke with a fake Codex: sent `i` to open/close; captured alternate-screen enter/restore, workflow pane, and arriving agent output in `/tmp/ft010-interactive-smoke.log`. |
| `EVID-04` | pass | `make docs-lint` |
| `EVID-05` | local pass; CI pending | `go test ./...`, `go vet ./...`, `make docs-lint`, `git diff --check` |
