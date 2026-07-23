---
title: "FT-022: Implementation Plan"
doc_kind: feature
doc_function: derived
purpose: "Grounded execution plan for FT-022 that realizes the accepted schema-constrained review protocol without redefining scope, solution or validation profile."
derived_from:
  - brief.md
  - design.md
  - decision-log.md
  - ../../engineering/testing-policy.md
status: archived
audience: humans_and_agents
must_not_define:
  - ft_022_scope
  - ft_022_selected_design
  - ft_022_acceptance_criteria
  - ft_022_blocker_state
  - ft_022_validation_profile
---

# FT-022: Implementation Plan

## Цель текущего плана

Replace the review-stage terminal-stream parser input with the accepted schema-constrained Codex final-response file, preserve the existing review target and public workflow contract, and close the high-risk validation/evidence obligations under the explicit approval recorded in `DL-09`.

## Grounding / Support References

| Document | Role in this plan | Facts reused | Conflict action |
| --- | --- | --- | --- |
| `brief.md` | canonical problem / validation / verify owner | `REQ-*`, `SC-*`, `NEG-*`, `CHK-*`, `EVID-*`, high-risk floor | Update `brief.md` first for scope/acceptance/evidence changes. |
| `design.md` | canonical solution owner | `SOL-*`, `C4-*`, `SD-*`, `CTR-*`, `INV-*`, `FM-*`, `RB-*` | Update `design.md` first for protocol/contract/failure changes. |
| `decision-log.md` | reasoning provenance | routing conflict, evidence boundary and capability decision | Record new material choice, then promote it to the canonical owner. |
| `../FT-015/design.md` | delivered contract baseline | exact structured review shape and strict validation boundary | Do not rewrite historical evidence; change FT-022 owners if the new carrier needs different semantics. |
| `../../../README.md` | current public CLI contract | public review, stdout and exit behavior | Update atomically with delivered implementation; README wins for public wording. |
| `../../engineering/architecture.md` | current component/process boundary | adapter/runner/repository/workflow responsibilities | Update after implementation preserves or deliberately changes the current boundary. |

## Current State / Reference Points

| Path / module | Current role | Why relevant | Reuse / mirror |
| --- | --- | --- | --- |
| `internal/codex/adapter.go` | Runs `codex review`, parses `Result.Stdout`, preserves report; has a nil-`ReviewScope` legacy branch; finalization already uses schema/message temp files | Primary change surface and local structured-output pattern | Reuse strict review parser and finalization workspace/error style; remove the second unscoped protocol. |
| `internal/codex/adapter_test.go` | Parser matrix, invocation recording, finalization file fakes | Nearest deterministic coverage | Extend runner fakes to write review message files and assert channel authority/cleanup. |
| `internal/repository/review.go` | Resolves selected base/merge base once and refreshes the private wrapper-backed snapshot per review | `REQ-04` target owner | Preserve `ReviewTarget.BaseCommit`, `MergeBase`, wrapper `Env`, inherited-Git `UnsetEnv`, `Scope` and lifecycle unchanged. |
| `internal/runner/runner.go` | Executes local process in configured cwd, supplies stdin/env, captures stdout/stderr, enriches non-zero errors | Defines channel and failure boundary; process env alone can be filtered before Codex-spawned tools | Keep capture/error behavior; add the targeted Codex config override in adapter args, not runner-wide policy. |
| `internal/app/app_test.go` | Fake Git/Codex runner and end-to-end clean/no-change/finalization paths | Public event/exit regression surface | Teach fake Codex to distinguish review/finalize output files; retain event assertions. |
| `internal/workflow/workflow.go` and tests | Own transitions, counters, budgets and exit codes | `REQ-06` compatibility owner | No planned production change; run/update regressions only if fixture interface requires it. |
| `README.md` | Public command/result/stdout contract | Currently states the pre-FT-022 `codex review`/plain-text behavior | Replace only delivered review-protocol wording/diagram; keep event/exit contracts. |
| `memory-bank/engineering/architecture.md` | Active component/process contract | Currently states ordinary `codex review` without supplied schema | Update current-state connector description after code is delivered. |
| `memory-bank/engineering/testing-policy.md` | Required boundary/fixture inventory | Currently requires plain-text review classification fixtures | Update to final-response/schema/channel fixtures while retaining relevant legacy parser tests only if code remains. |
| `memory-bank/prd/PRD-001-code-converge-cli.md` and `memory-bank/domain/model.md` | Active product/domain descriptions of ordinary report parsing | Directly dependent current-state wording | Align with schema-valid final response without redefining public fields. |

## Test Strategy

The validation profile is owned by `brief.md`. Its high-risk obligations are realized below; no automated test may start a real Codex session or mutate a real remote.

| Test surface | Canonical refs | Existing coverage | Planned automated coverage | Required local suites / commands | Required CI suites / jobs | Manual-only gap / justification | Manual-only approval ref |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Strict review schema/parser | `CTR-02`, `INV-02`, `INV-03`, `FM-04`, `NEG-03` | FT-015 clean/findings and rejected JSON matrix | Assert schema required/additional-properties/priority contract; retain duplicate/case/trailing validation | `go test ./internal/codex` | repository required Verify job | none | none |
| Final-response channel | `SD-01`, `SD-04`, `CTR-04`, `FM-02`–`FM-04`, `NEG-01`–`NEG-04` | Finalization output-file fakes only | Table clean/findings/missing/empty/malformed/incomplete; valid stdout/stderr conflicts; non-zero with valid file | `go test ./internal/codex` | repository required Verify job | none | none |
| Invocation/target boundary | `SOL-01`, `SOL-02`, `SD-03`, `SD-06`–`SD-08`, `CTR-01`, `CTR-03`, `INV-04`, `INV-06`, `FM-07`–`FM-09` | Current args/model/effort and repository target tests | Assert nil/missing/duplicate target fields fail before invocation; otherwise assert `exec` flags, TOML-quoted shell-policy override, unique paths, stdin selected-base/merge-base/private-index semantics, exact env and post-return cleanup with fake executable | `go test ./internal/codex ./internal/repository` | repository required Verify job | none | none |
| Workflow/app compatibility | `SOL-05`, `CTR-05`, `INV-05`, `REQ-06` | Clean/findings/failure/no-change and event tests | Update fake runner output-file behavior; assert unchanged records/counters/budgets/exits and no raw stream in stdout | `go test ./internal/workflow ./internal/app` | repository required Verify job | none | none |
| Documentation/evolution | `SD-05`, `FM-05`, `RB-02`, `REQ-08` | docs lint and current public owners | Semantic read-through plus link/frontmatter lint; document capability-based failure with no invented version floor | `make docs-lint` | repository required Verify job | none | none |
| Full convergence/backout | `RB-01`, `RB-02`, `EC-07` | repository full checks | Full Go/vet/docs/diff, simplify pass, independent code-converge and CI | `go test ./... && go vet ./... && make docs-lint && git diff --check` | all required jobs | Independent review is external evidence but not a manual test gap. | `AG-01` before execution; publication authorization separately required |

## Open Questions / Ambiguities

None after grounded design. A newly observed Codex schema/capability conflict triggers `STOP-01` or `STOP-02`; it must not be resolved inside a code step.

## Environment Contract

| Area | Contract | Used by | Failure symptom |
| --- | --- | --- | --- |
| setup | Go toolchain, Git repository and writable OS temp directory; no production secrets or remote mutation needed for local tests | `STEP-01`–`STEP-07` | Temp/schema setup or repository fixtures cannot be created. |
| Codex capability | Local discovery currently shows `codex-cli 0.145.0` with `exec --output-schema` and `--output-last-message`; automated coverage uses a fake executable/runner | `STEP-02`–`STEP-05` | Required flags/schema behavior cannot be represented without changing `CTR-*`. |
| test | Fakes must capture args/stdin/env/stdout/stderr/exit and write only named output files; real Codex sessions are forbidden in automated tests | `CHK-01`–`CHK-03` | Test outcome depends on credentials, network, live model output or real repository mutation. |
| access / publication | Code edits require `AG-01`; commit/push/PR/CI actions require `AG-02` publication authority | `STEP-01`, `STEP-08` | Required approval/reference is absent. Both gates are satisfied by the 2026-07-23 delivery instruction recorded in `DL-09`. |

## Preconditions

| Precondition ID | Canonical ref | Required state | Used by steps | Blocks start |
| --- | --- | --- | --- | --- |
| `PRE-01` | `brief.md`, `design.md`, `decision-log.md` | Core owners active; review-improve final status `done`; no open human gate | `STEP-01`–`STEP-08` | yes |
| `PRE-02` | high-risk decision in `brief.md`; `AG-01` | Human explicitly approves transition from Plan Ready to Execution | `STEP-01`–`STEP-08` | yes |
| `PRE-03` | `CTR-02`, `SD-02`, FT-015 contract | Exact schema/parser baseline remains accepted | `STEP-02`–`STEP-05` | yes |
| `PRE-04` | `CON-02`, `CON-05`, `SD-03`, `SD-06`–`SD-08`, `INV-04`, `INV-06` | Existing review target tests remain green and production wiring provides a non-nil `ReviewScope` with selected base, merge base, wrapper-first `PATH` and inherited-Git transport removals before adapter changes | `STEP-03`–`STEP-05` | yes |

## Design Realization Mapping

| Canonical solution refs | Owner | Realization target | Steps | Checks | Evidence |
| --- | --- | --- | --- | --- | --- |
| `SOL-01`, `TRD-03`, `FM-01`, `FM-06` | `design.md` | Per-review temp workspace/schema/result paths in `internal/codex` | `STEP-02`, `STEP-04` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `SOL-02`, `SD-03`, `SD-06`–`SD-08`, `CTR-01`, `CTR-03`, `INV-04`, `INV-06`, `FM-07`–`FM-09`, `C4-01` | `design.md` | Required-target guard, merge-base prompt, targeted shell-policy override, adapter invocation and unchanged `ReviewScope` handoff | `STEP-03`, `STEP-04` | `CHK-02`, `CHK-03` | `EVID-02`, `EVID-03` |
| `SOL-03`, `SD-02`, `CTR-02`, `INV-02`, `INV-03`, `FM-03`, `FM-04` | `design.md` | Final-response read and strict structured parser path | `STEP-02`, `STEP-04` | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` |
| `SOL-04`, `SD-01`, `SD-04`, `CTR-04`, `INV-01`, `FM-02` | `design.md` | Runner-result handling and channel-conflict tests | `STEP-03`, `STEP-04`, `STEP-05` | `CHK-01`–`CHK-03` | `EVID-01`–`EVID-03` |
| `SOL-05`, `CTR-05`, `INV-05`, `FM-05` | `design.md` | Existing workflow/app result boundary and public events | `STEP-05`, `STEP-06` | `CHK-03`, `CHK-04` | `EVID-03`, `EVID-04` |
| `SD-05`, `TRD-01`, `TRD-02`, `RB-01`, `RB-02` | `design.md` | Capability failure, documentation and release convergence | `STEP-06`–`STEP-08` | `CHK-04`, `CHK-05` | `EVID-04`, `EVID-05` |

## Workstreams

| Workstream | Implements | Result | Owner | Dependencies |
| --- | --- | --- | --- | --- |
| `WS-01` | `SOL-01`–`SOL-04`, `SD-06`–`SD-08`, `CTR-01`–`CTR-04`, `INV-01`–`INV-04`, `INV-06`, `FM-01`–`FM-09` | Strict output-file review adapter, required-target guard, merge-base scope, private-index tool binding and complete boundary fixtures | agent after approval | `PRE-01`–`PRE-04` |
| `WS-02` | `SOL-05`, `CTR-05`, `INV-05`, `REQ-06`, `REQ-07` | Unchanged workflow/app behavior proven with updated fakes | agent after approval | `WS-01` |
| `WS-03` | `REQ-08`, `SD-05`, `RB-01`, `RB-02` | Public/current-state docs, full evidence, independent convergence and publication | agent/human | `WS-01`, `WS-02`, `AG-01` |

## Approval Gates

| Approval Gate ID | Trigger | Applies to | Why approval is required | Approver / evidence |
| --- | --- | --- | --- | --- |
| `AG-01` | Transition from Plan Ready to any implementation/code edit | `STEP-01`–`STEP-08` | The canonical profile is high-risk because result protocol/failure semantics cross the external process trust boundary and can select a false clean transition. | Approved by the user's 2026-07-23 end-to-end implementation instruction; recorded in `DL-09`. |
| `AG-02` | Commit, push, PR creation or other remote publication | `STEP-08` | Publication is externally effective and requires scope confirmation in the execution turn. | Approved by the same 2026-07-23 instruction to commit, push, create/update the PR and drive CI; recorded in `DL-09`. |

## Порядок работ

| Step ID | Actor | Implements | Goal | Touchpoints | Artifact | Verifies | Evidence IDs | Check command / procedure | Blocked by | Needs approval | Escalate if |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `STEP-01` | agent | high-risk profile, `RB-01` | Confirm approval, clean baseline and unchanged target behavior | issue/decision log; existing tests | approval/baseline record | `CHK-03`, `CHK-05` | `EVID-03`, `EVID-05` | Record `AG-01`; run affected baseline tests | `PRE-01`–`PRE-04` | `AG-01` | Approval or baseline evidence is absent. |
| `STEP-02` | agent | `SOL-01`, `SOL-03`, `SD-02`, `CTR-02`, `FM-01`, `FM-03`, `FM-04`, `FM-06` | Add exact schema, temp workspace and strict file parser path | `internal/codex/adapter.go`, nearest tests | schema/workspace/parser change | `CHK-01` | `EVID-01` | Focused table tests | `STEP-01` | none after `AG-01` | Schema cannot match FT-015 without new semantics. |
| `STEP-03` | agent | `SOL-02`, `SOL-04`, `SD-01`, `SD-03`–`SD-08`, `CTR-01`, `CTR-03`, `CTR-04`, `FM-02`, `FM-05`, `FM-07`–`FM-09` | Remove the unscoped fallback, validate base/merge-base, bind the exact private index into Codex tool policy, replace review invocation and establish sole-carrier/channel behavior | `internal/codex/adapter.go`, runner fakes | one required-scope merge-base protocol | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` | Nil/target guard plus fake args/stdin/env/stream/output-file matrix | `STEP-02` | none after `AG-01` | Correctness requires terminal fallback, target reconstruction, persistent config mutation or invented version floor. |
| `STEP-04` | agent | `INV-01`–`INV-04`, `INV-06`, `REQ-01`–`REQ-05`, `REQ-07` | Complete adapter negative/isolation/cleanup coverage | `internal/codex/adapter_test.go` | exhaustive focused regression suite | `CHK-01`, `CHK-02` | `EVID-01`, `EVID-02` | `go test ./internal/codex ./internal/repository` | `STEP-03` | none | Any missing/duplicate target value invokes Codex, spawned-tool override diverges from process env/prompt, or any required failure yields counters/clean. |
| `STEP-05` | agent | `SOL-05`, `CTR-05`, `INV-05`, `REQ-06` | Update fakes and prove unchanged workflow/app outcomes | `internal/app/app_test.go`, affected workflow tests; production workflow only if required | compatibility regression | `CHK-03` | `EVID-03` | `go test ./internal/workflow ./internal/app` | `STEP-04` | none | A public event/exit/transition must change. |
| `STEP-06` | agent | `REQ-08`, `SD-05`, `FM-05`, `RB-02` | Update public and directly dependent current-state documentation | README; architecture; testing policy; PRD/domain current owners; FT-022 evidence | converged delivered contract | `CHK-04` | `EVID-04` | `make docs-lint` plus semantic owner read-through | `STEP-05` | none | Updating a historical feature or new public/config contract appears necessary. |
| `STEP-07` | agent | all accepted refs | Run functional validation, separate simplify review and acceptance audit | complete diff | local verification and review records | `CHK-01`–`CHK-05` | `EVID-01`–`EVID-05` | Focused tests; full Go/vet/docs/diff; trace audit | `STEP-06` | none | A high-risk obligation or canonical trace lacks evidence. |
| `STEP-08` | agent/human | `RB-01`, `RB-02`, `EC-07` | Publish only when authorized; obtain independent review and required CI | Git branch/PR/CI | commit/PR/CI/review evidence | `CHK-05` | `EVID-05` | Approved commit/push/PR; independent code-converge; required CI | `STEP-07`, `AG-02` | `AG-02` | Critical/high finding, red CI or unauthorized external action remains. |

## Parallelizable Work

- `PAR-01` After `STEP-03` stabilizes the adapter contract, focused adapter negative fixtures and app fake-runner adaptation may be prepared independently if they do not edit the same helper types.
- `PAR-02` Canonical public/current-state documentation must wait for the delivered behavior and cannot run in parallel with unresolved protocol changes.
- `PAR-03` Functional validation, simplify review and acceptance audit are separate ordered passes even when performed in one session.

## Checkpoints

| Checkpoint ID | Refs | Condition | Evidence IDs |
| --- | --- | --- | --- |
| `CP-01` | `STEP-03`, `CTR-01`–`CTR-04`, `INV-01` | Invocation uses `exec` and only the named final-response file can classify. | `EVID-01`, `EVID-02` |
| `CP-02` | `STEP-04`, `FM-01`–`FM-09`, `NEG-01`–`NEG-05` | Complete target/failure/channel matrix fails closed and temp paths are isolated. | `EVID-01`, `EVID-02` |
| `CP-03` | `STEP-05`, `INV-05`, `EC-06` | Existing workflow/app event, counter, budget and exit assertions remain green. | `EVID-03` |
| `CP-04` | `STEP-07`, `RB-01`, `EC-07` | All local/profile/document/trace checks pass before any publication. | `EVID-01`–`EVID-05` |
| `CP-05` | `STEP-08`, `RB-02` | Independent review has no critical/high finding and required CI is green. | `EVID-05` |

## Execution Risks

| Risk ID | Risk | Impact | Mitigation | Trigger |
| --- | --- | --- | --- | --- |
| `ER-01` | Fake runner writes the same message for review and finalization or parses command text ambiguously | False-positive app coverage | Distinguish invocation by subcommand/flags and assert each output path separately. | App tests pass while wrong stage file is populated. |
| `ER-02` | Reusing `ParseReview` retains plain-text fallback | Schema-invalid prose could classify clean | Call the strict structured path directly and add allowlisted-prose-in-file rejection. | A clean prose fixture passes. |
| `ER-03` | The result file is read after runner failure | Partial output can select a false result | Return immediately on runner error; test valid file plus non-zero exit. | Non-zero fixture returns counts/clean. |
| `ER-04` | Current-state docs are updated before behavior or historical FT-015 is rewritten | Documentation claims undelivered behavior or loses provenance | Update active owners atomically in `STEP-06`; leave FT-015 unchanged. | Diff changes README/architecture before code or edits FT-015 semantics. |
| `ER-05` | High-risk profile is treated as satisfied by focused tests alone | Missing independent/failure/backout assurance | Enforce `AG-01`, `CP-04`, `CP-05` and `EVID-05`. | Execution/publication proceeds without approval or independent review. |
| `ER-06` | Unit tests preserve the nil-scope legacy branch because it is convenient to instantiate | A second terminal-stream protocol survives outside production wiring | Make nil scope fail fast and construct a real/fake prepared scope for successful review tests. | A successful adapter review has no `ReviewTarget`. |
| `ER-07` | Tests assert process env but omit Codex-spawned tool policy or inherited-variable removal | User shell filters can hide the scoped wrapper, while caller Git transports can redirect the repository/index | Assert the exact invocation-local `shell_environment_policy.set.PATH`, login-shell override and `ReviewTarget.UnsetEnv` handoff. | Args omit/misquote the wrapper path or the runner retains inherited Git transports. |
| `ER-08` | Prompt compares the private index directly with the selected base tip | Diverged target-branch commits contaminate the review and change existing scope | Assert prompt uses `ReviewTarget.MergeBase` for `git diff --cached` and keeps `BaseCommit` only as provenance. | Base commit and merge base differ in a fixture but prompt uses the base tip. |

## Stop Conditions / Fallback

| Stop ID | Related refs | Trigger | Immediate action | Safe fallback state |
| --- | --- | --- | --- | --- |
| `STOP-01` | `CTR-02`, `SD-02`, `FM-05` | Supported Codex rejects the exact FT-015 schema or emits a materially different required shape | Stop implementation; record evidence and return to `design.md`/decision log. | Pre-change adapter and planned delivery state |
| `STOP-02` | `SD-01`, `CTR-04`, `INV-01` | Correct classification appears to require stdout/stderr parsing or fallback prose | Stop; do not weaken carrier authority; request a human decision if issue evidence cannot resolve it. | Pre-change fail-closed behavior |
| `STOP-03` | `SD-03`, `SD-07`, `SD-08`, `INV-04`, `FM-09`, `CON-02`, `CON-05` | `codex exec` cannot preserve the prepared merge-base/private-index target through the prompt, process and invocation-local tool-policy binding | Stop before changing repository scope or public review semantics; return to design/human gate. | Existing `ReviewScope` unchanged |
| `STOP-04` | `INV-05`, `REQ-06` | Implementation requires a public event, counter, budget or exit-code change | Stop and update problem scope through a separately approved decision. | Existing public contract unchanged |
| `STOP-05` | `RB-01`, `EC-07` | Approval, required local evidence, independent review or required CI is missing/failing | Do not publish or close the delivery. | Plan Ready or unmerged branch |

## Plan-local Evidence

| Evidence ID | Artifact | Producer | Path contract | Reused by checkpoints |
| --- | --- | --- | --- | --- |
| `EVID-06` | Feature-document review-improve final report | documentation reviewer | `memory-bank/features/FT-022/feature-review-report.md` | `PRE-01`, `CP-04` |
| `EVID-07` | Simplify-review verdict | implementer/reviewer | `artifacts/ft-022/verify/simplify-review/` or PR review record | `CP-04`, `CP-05` |

## Готово для приемки

- All workstreams are complete or stopped through a documented `STOP-*`.
- `AG-01` and any externally effective `AG-02` have concrete approval references.
- `CHK-01`–`CHK-05` have concrete `EVID-01`–`EVID-05` carriers.
- Functional validation, simplify review and acceptance audit are distinct recorded passes.
- README and directly dependent active Memory Bank owners agree with the implementation; FT-015 remains historical.
- Independent code-converge has no critical/high finding and required CI is green.
- Final acceptance and `delivery_status` transition are performed only against `brief.md`.
