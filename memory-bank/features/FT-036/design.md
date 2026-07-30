---
title: "FT-036: Discoverable CLI Help Design"
doc_kind: feature
doc_function: canonical
purpose: "Selected solution and public CLI help contract for FT-036."
derived_from:
  - brief.md
  - ../../flows/feature.md
  - ../../../README.md
status: active
audience: humans_and_agents
must_not_define:
  - problem_scope
  - execution_sequence
---

# FT-036: Discoverable CLI Help Design

## Design Pack

No separate design-pack artifact is required: the contract is a small, single-process dispatch change.

## Selected Design

- `SOL-01` Recognize supported help forms before command dispatch, configuration resolution, session setup, workflow creation, Codex runner use, or update runner use. Root help accepts only `-h` and `--help`; subcommand help accepts `config --help`/`config -h` and `update --help`/`update -h`.
- `SOL-02` Render deterministic human-oriented help from a single in-process command/option reference. Root output contains usage, commands, grouped global options, and a README/configuration pointer; command output contains purpose and syntax. The option reference is shared with flag registration where practical so a renamed or added global flag cannot silently drift from help.
- `SOL-03` Leave all non-help dispatch paths unchanged, including invalid arguments and workflow `kv` output.

## Design Decisions

| Decision | Outcome | Traceability |
| --- | --- | --- |
| `SD-01` | Help text is static in-process output and must not resolve effective config. | `REQ-01`, `REQ-03` |
| `SD-02` | The global option inventory is declared once for both flag binding and help rendering; descriptive grouping may remain presentation metadata. | `REQ-01`, `CON-01` |
| `SD-03` | Help is a successful command path only for recognized root/subcommand forms; malformed update syntax keeps the existing operational failure. | `REQ-02`, `NEG-01` |

## C4 Applicability Decision

- `C4-00: not required` — the change stays inside the existing `internal/app` CLI-dispatch component and introduces no component, runtime, storage, integration, or deployment boundary.

## Architecture Coverage Decision

| Aspect | Decision | Evidence |
| --- | --- | --- |
| Components | covered | `App.Run` dispatch and the existing flag registration are the only changed responsibilities. |
| Connectors | N/A | No inter-process, network, storage, or configuration connector is entered on help paths. |
| Configuration | covered | Help dispatch precedes `config.ResolveLogFormat` and `config.Load`. |
| Behavioral semantics | covered | Recognized help returns 0/stdout; invalid non-help arguments retain existing exit semantics. |
| Quality/evolution | covered | A shared option declaration and focused exact-output tests make the public text stable and reduce drift. |

## Contracts and Invariants

- `CTR-01` Root help output is stdout-only, exit 0, and includes root usage, `config` and `update` command synopses, global option groups, and a README/configuration reference.
- `CTR-02` Config help output is stdout-only, exit 0, and states its purpose plus valid invocation; update help additionally lists `--yes` and `-y`.
- `INV-01` A recognized help path never invokes config loading, session logging, workflow/Codex runner, or self-update.
- `INV-02` Machine-readable workflow records are emitted only by workflow paths, never by help.

## Failure Modes and Backout

- `FM-01` A future flag change could make help stale; `SD-02` and focused tests detect it before publication.
- `RB-01` The change is an additive local code/docs/test diff and can be reverted as one commit; no runtime state or rollout action exists.

## Design Verification

| Analysis class | Required | Method | Result / evidence |
| --- | --- | --- | --- |
| Contract compatibility | yes | Compare README contract, existing app tests, and proposed acceptance cases. | `CTR-01`/`CTR-02` retain existing root syntax and add successful documented paths. |
| State/transition completeness | yes | Enumerate root, config, update, invalid-update, and workflow dispatch paths. | `SOL-01` and `SD-03` cover each path. |
| Failure propagation | yes | Assert side-effecting collaborators are untouched on help. | `INV-01` is testable with fakes. |
| Concurrency/ordering | no | Single synchronous CLI dispatch; no shared state. | N/A. |
| Security boundaries | no | No credential, auth, or trust-boundary change. | N/A. |
| Capacity/latency | no | Rendering a bounded in-process text block has no material performance impact. | N/A. |
| Migration/evolution safety | yes | Share option registration inventory and preserve non-help branches. | `SD-02`, `SD-03`, `FM-01`. |
