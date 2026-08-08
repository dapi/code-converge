---
title: "FT-042: Configurable document review prompts"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem, scope, validation profile and verification contract for issue #42."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/42
status: active
delivery_status: done
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-042: Configurable document review prompts

## What

The review stage currently has one built-in code-review instruction. Operators need an explicit, deterministic document-review mode and explicit prompt selection without replacing ordinary code review by default.

## Scope

- `REQ-01` The CLI accepts at most one explicit review-prompt selection: a Markdown prompt path, a safe project-local prompt name, or document-review mode; no selector retains ordinary code review, while conflicting selections fail predictably with exit code `2` and no implicit fallback.
- `REQ-02` A named review prompt resolves only from `.code-converge/<name>.md`; invalid names and unreadable or missing selected files fail predictably with exit code `2`.
- `REQ-03` Document-review mode reviews only changed Markdown files in the current merge-base-to-worktree snapshot, excludes `memory-bank/prompts/**`, reports a clean, explicit empty scope without invoking Codex when none remain, and never publishes or waits for CI after a clean scoped review.
- `REQ-04` The built-in document-review instruction evaluates changed in-scope documents for consistency, contradictions, unresolved material questions, and Memory Bank principles while preserving the existing strict review-result schema and findings loop.
- `REQ-05` `code-converge init-document-review-prompt [--force]` writes the built-in document-review prompt to `.code-converge/default.md`; it refuses to overwrite an existing file unless `--force` is explicit and diagnoses creation errors with exit code `2`.
- `REQ-06` Document review uses either an explicitly selected document-fix prompt file or a built-in document-fix instruction; it remains isolated from ordinary `--fix-prompt-file`, and invalid combinations or unreadable files fail with exit code `2`.
- `REQ-07` The root README documents the final CLI/config contract, precedence, errors, export behavior, and document-scope limitation with runnable examples.

## Non-Scope

- `NS-01` Changing the default ordinary code-review behavior, result schema, finding priorities, review/fix budgets, publication, or CI workflow.
- `NS-02` Loading named prompts from arbitrary paths, other extensions, user-level config, environment variables, or an implicit search path.
- `NS-03` Reviewing non-Markdown files in document-review mode or treating `memory-bank/prompts/**` as document-review scope.
- `NS-04` Adding a reusable project-wide architecture rule or changing Memory Bank governance itself.

## Assumptions and Constraints

- `ASM-01` The owner decisions recorded in issue comments on 2026-08-05 are accepted source facts for this feature.
- `CON-01` Errors must fail closed and be visible through the existing operational-error contract (exit `2`), never by falling back to another selected prompt.
- `CON-02` The existing private review snapshot and strict JSON schema remain the compatibility boundary.

## Design Requirement Decision

`Design required: yes` — the delivery changes public CLI and configuration contracts, document scope, workflow behavior, and prompt/file-resolution semantics.

## Validation Profile Decision

Validation profile: `standard`.

Triggers / rationale: executable behavior changes public CLI/configuration and workflow contracts. No security, persistent-data, financial, concurrency, cross-system protocol, or production-release trigger applies.

Downgrade approval: none.

## Verify

| Scenario | Observable result |
| --- | --- |
| `SC-01` | Each supported review-prompt selector resolves its stated source, and any pair of selectors fails with exit `2` without invoking Codex. |
| `SC-02` | Named prompt validation accepts only safe names and reads only `.code-converge/<name>.md`; missing/unreadable prompt sources fail with actionable exit `2`. |
| `SC-03` | Document-review mode sends the schema-constrained review instruction for only changed eligible Markdown files; an empty eligible set completes clean without a Codex invocation. |
| `SC-04` | `init-document-review-prompt` creates the template once, rejects overwrite without `--force`, and overwrites only with `--force`. |
| `SC-05` | Document findings enter fix using the selected document-fix file or built-in document-fix prompt; ordinary review/fix prompt behavior is unchanged. |
| `SC-06` | README examples and contract agree with help/config behavior and project documentation lint passes. |

| Negative case | Observable result |
| --- | --- |
| `NEG-01` | `--document-fix-prompt-file` without `--document-review`, or together with `--fix-prompt-file`, exits `2` with an actionable diagnostic. |
| `NEG-02` | A malformed name, non-Markdown selected prompt, unreadable source, or document-review scope that includes excluded prompt artifacts cannot silently broaden scope or fall back; an explicitly supplied readable Markdown path may be outside the repository root. |

## Traceability

| Requirement | Acceptance scenarios | Negative coverage |
| --- | --- | --- |
| `REQ-01` | `SC-01` | `NEG-01`, `NEG-02` |
| `REQ-02` | `SC-02` | `NEG-02` |
| `REQ-03` | `SC-03` | `NEG-02` |
| `REQ-04` | `SC-03` | none |
| `REQ-05` | `SC-04` | `NEG-02` |
| `REQ-06` | `SC-05` | `NEG-01`, `NEG-02` |
| `REQ-07` | `SC-06` | none |

| Check ID | Covers | Required evidence |
| --- | --- | --- |
| `CHK-01` | `SC-01`–`SC-05`, `NEG-01`, `NEG-02` | Focused deterministic config/app/codex/workflow/repository tests using fake runner/executable. |
| `CHK-02` | all executable behavior | `go test ./...`; `go vet ./...`; `git diff --check`. |
| `CHK-03` | `REQ-07`, package artifacts | `make docs-lint`; semantic README/help/config read-through. |
| `CHK-04` | final changed behavior | Independent `codex review --base master` with findings triaged under the same convergence episode. |

- `EVID-01` Focused table-driven prompt selection, resolution, scope, export, and fix-stage tests.
- `EVID-02` Full local Go tests, vet, and diff integrity.
- `EVID-03` Documentation lint and contract read-through.
- `EVID-04` Clean independent implementation-review result and CI evidence for the published head.
