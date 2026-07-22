---
title: "FT-020: Decision Log"
doc_kind: feature-support
doc_function: decision_log
purpose: "Аудируемый журнал FPF reasoning, review-improve циклов и human gates для FT-020."
derived_from:
  - brief.md
  - https://github.com/dapi/code-converge/issues/20
status: active
audience: humans_and_agents
---

# FT-020: Decision Log

## Artifact Contract

- **Role:** provenance companion for FPF decisions, feature-document review and gate status.
- **Owns:** source facts, alternatives, reasoning, confidence, document conflicts and human gates.
- **Must not define:** requirements, public CLI contract, selected solution or execution sequence. Accepted facts move to `brief.md`, `design.md`, `implementation-plan.md` or the root README.

## FPF Method

The review keeps problem requirements, public contract, solution and execution as separate bounded contexts. It uses the canonical reasoning cycle: observe the issue and current public/code facts, identify a decision only where evidence leaves a material choice, compare only evidence-backed alternatives, and either promote a grounded result to its canonical owner or stop at a human gate. This prevents an absent help payload from being invented in the design or plan.

## Decision Entries

### `DL-01` — Route and assurance floor

- **Status:** resolved; promoted to `brief.md`.
- **Question:** Which delivery flow and validation profile apply?
- **Facts:** Issue #20 explicitly says Feature Flow applies because the public CLI argument, exit and output contract change. It is one independently verifiable root-command delivery-unit. Validation policy requires at least `standard` for a changed public CLI/output/exit contract; the issue and current documents identify no security, financial, persistent-data, migration, concurrency, release/deployment or cross-system trigger.
- **Alternatives:** Small Change/low-risk; Feature Flow/standard; Feature Flow/high-risk.
- **FPF reasoning:** The routing decision and assurance floor are distinct. The issue excludes Small Change, while standard is the policy default for an executable public contract that has no documented high-risk trigger. High-risk cannot be selected merely from uncertainty about prose content.
- **Result:** FT-020 follows Feature Flow with `standard` validation and no downgrade.
- **Confidence:** high; direct issue, routing and validation-policy facts.

### `DL-02` — Root help payload is unresolved

- **Status:** superseded by `DL-03`; initially escalated as `HG-01` and referenced by `DEC-01`.
- **Question:** What exact root usage/help text must successful `-h` and `--help` write to stdout?
- **Facts:** Issue #20 requires “the existing root usage/help text”, stdout, exit `0`, no error and no operational stages. Current `internal/app` suppresses Go flag output and only emits `code-converge: usage: code-converge [flags] [config]` to stderr for an invalid positional argument. The root README owns the public CLI contract but contains no root-help section or text. The code and tests contain no successful root-help output.
- **Alternatives:** (A) emit the existing one-line usage fragment, normalized for successful stdout help; (B) define a fuller root help text including flags and commands; (C) adopt an exact text supplied by the product owner.
- **FPF reasoning:** The issue proves the aliases, success, destination and early exit, but it does not define whether “usage/help” means the existing one-line error fragment or a new multi-line help contract. The repository has no existing successful root-help text from which option A or B can be derived. The alternatives have different user-visible stdout and documentation/test oracles. Selecting one would create a public requirement rather than resolve it from evidence.
- **Result:** Do not create `design.md` or `implementation-plan.md`, and do not change the root README or code, until a human selects the payload.
- **Confidence:** high that the decision is unresolved; direct issue, README, app code and test evidence.

### `DL-03` — Agent-safe root help contract

- **Status:** resolved; promoted to `brief.md`, `design.md` and the root README; closes `HG-01`.
- **Question:** Which root-help payload delivers conventional aliases without creating an unstable interface for humans or agents?
- **Facts:** Issue #20 requires successful `-h`/`--help`, stdout-only output and no operational stages. The current error-path usage already names the root grammar as `code-converge [flags] [config]`. The root README owns public CLI text. Feature FT-009 establishes that machine consumers need deterministic semantic output and that stdout contracts should not gain accidental variability. The command has many flags, while the issue does not ask for a new flag inventory or subcommand help.
- **Alternatives:** (A) a stable one-line usage payload; (B) a dynamically rendered or manually duplicated list of all flags/commands; (C) retain error-path-only usage and accept aliases without output.
- **FPF reasoning:** **Abduction:** A is the smallest new public contract compatible with the issue and the existing grammar. **Deduction:** A predicts identical, one-line, newline-terminated stdout for both aliases, no stderr and an early return before configuration/runner/updater construction; it does not create coupling between help output and future flags. B creates a new, volatile agent-facing parsing surface and claims content the issue does not require; C contradicts the requested usage/help output. **Induction:** deterministic app tests are the evidence method for both aliases and no side effects.
- **Result:** `-h` and `--help` print exactly `usage: code-converge [flags] [config]\n` to stdout and exit `0`; no prefix, diagnostics, workflow event or dynamic flag list is emitted. This resolves `DEC-01` and permits downstream artifacts.
- **Confidence:** high; issue acceptance, existing grammar and established deterministic-output constraint support the minimal contract. The selection is an explicitly recorded FPF abductive decision, tested through `CHK-01`.

## Review-Improve Cycles

### Cycle 1 — bootstrap package and public-contract review

- **Review scope:** issue #20; Task Routing; Feature Flow and artifact catalog; validation/testing policy; root README; current root app/parser/tests; product, domain and architecture owners; all instantiated FT-020 documents.
- **Critical:** none.
- **Important:** `IMP-01` The exact successful root help payload is undefined. Issue #20 refers to existing root usage/help text, but the only current usage text is an error-path stderr fragment and neither code nor README establishes a successful root-help contract. A design or implementation plan would have to invent a user-visible output contract.
- **FPF closures:** `DL-01` resolves Feature Flow/standard. `DL-02` determines that `IMP-01` cannot be resolved from the available evidence and must be escalated.
- **Changes:** created bootstrap `README.md`, canonical `brief.md` and this decision log; recorded the routing rationale, standard validation floor, complete verify/evidence contract and deferred downstream artifacts; registered the package in the feature index.
- **Minor:** none changed.
- **Human gate:** yes, `HG-01`; the cycle stops here as required.

### Cycle 2 — FPF contract convergence

- **Review scope:** issue #20, root CLI grammar/error usage, README ownership, agent-consumption constraint, FT-020 brief and decision log.
- **Critical:** none.
- **Important:** `IMP-01` is resolved by `DL-03`; no other important document inconsistency remains.
- **FPF closures:** `DL-03` selects the minimal stable usage line and early-return predictions.
- **Changes:** promoted the exact payload to `brief.md`, unblocked and created `design.md` and `implementation-plan.md`, and changed the feature index from blocked to active.
- **Minor:** none changed.
- **Human gate:** no.

### Cycle 3 — implementation and local convergence

- **Review scope:** complete FT-020 package, root README, `internal/app` dispatch/test boundary and the built binary's two aliases.
- **Critical:** none.
- **Important:** none. The implementation recognizes the aliases before every operational dependency, the exact root README payload matches `CTR-01`, and one deterministic table test covers stdout, stderr, exit code, runner and updater behavior for both aliases.
- **FPF closures:** induction for `DL-03`: `go test ./internal/app` and an independently built-binary smoke check corroborate its predicted stable output and stage-free path.
- **Changes:** added root-help early return and `rootUsage`, alias table test, root public-contract section, `design.md` and `implementation-plan.md`; changed delivery state to `in_progress`.
- **Verification:** `go test ./internal/app`; `go test ./...`; `go vet ./...`; `go build -o /tmp/code-converge-ft020 ./cmd/code-converge`; both binary aliases with stdout/stderr comparison; `make docs-lint`; `git diff --check` all pass.
- **Minor:** none changed.
- **Human gate:** no.

## Human Gate

### `HG-01` — Exact stdout root-help contract

- **Question:** What exact text should `code-converge -h` and `code-converge --help` print to stdout?
- **Available facts:** Issue #20 requires both aliases, stdout, exit `0`, no error and no review/update stages. The current code has no successful help path and suppresses Go flag output. Its only usage string is the error-path `code-converge: usage: code-converge [flags] [config]`; the root README currently documents no root-help text.
- **Options:**
  - **A — Minimal existing usage:** print `usage: code-converge [flags] [config]` (or explicitly retain the `code-converge:` prefix) as the complete successful help payload.
  - **B — Full root help:** provide a new exact multi-line payload that includes the root usage plus the flags/commands you expect users to see.
  - **C — Custom text:** provide the exact stdout payload, including whether a trailing newline and the `code-converge:` prefix are required.
- **Risk of a wrong choice:** the CLI has a public stdout contract. Guessing the payload can produce misleading help, make the README and deterministic tests encode an unintended interface, and create a backwards-compatibility commitment not supported by issue #20.
- **Needed from a human:** choose A, B or C. For A, confirm prefix and newline; for B/C, provide the exact text or the required list of flags/commands it must contain.

**Resolution:** the user explicitly authorized FPF to resolve the choice, including the agent-consumption constraint. `DL-03` selects option A with no `code-converge:` prefix and a trailing newline.
