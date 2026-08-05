---
title: "FT-042: Document review prompt design"
doc_kind: feature
doc_function: canonical
purpose: "Selected CLI, prompt-resolution and document-review behavior for FT-042."
derived_from:
  - brief.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/42
status: active
audience: humans_and_agents
must_not_define:
  - ft_042_scope
  - ft_042_acceptance_criteria
  - implementation_sequence
---

# FT-042: Document review prompt design

## Design pack

| Artifact | Role | Owns |
| --- | --- | --- |
| `design.md` | Feature solution | `SOL-*`, `SD-*`, contracts, invariants and failure modes |

## C4 applicability decision

`C4-00: not required` — existing CLI, configuration, repository snapshot and Codex adapter components retain their boundaries; the change adds local policy within them.

## Selected solution

- `SOL-01` Add mutually exclusive CLI flags `--review-prompt-file <path>`, `--review-prompt <name>`, and `--document-review`, parsed before configuration loading; their values are runtime review choices, not YAML/environment configuration.
- `SOL-02` Resolve a selected prompt once: a file path must name a readable `.md` regular file (relative to the invocation directory; absolute paths are allowed); a name must match `[A-Za-z0-9][A-Za-z0-9_-]*` and maps only to `<repo>/.code-converge/<name>.md`; document review chooses `<repo>/.code-converge/default.md` when present, otherwise the built-in prompt. No other fallback is allowed.
- `SOL-03` Add `init-document-review-prompt [--force]` as a no-workflow command. It creates the project directory as needed, writes the built-in document-review prompt with owner-only permissions, refuses an existing target without `--force`, and has no configuration dependency.
- `SOL-04` Extend the review target with an eligible changed-file list computed from the same private merge-base snapshot. Document review permits only `.md` paths and excludes `memory-bank/prompts/`; an empty list returns a clean review result without invoking Codex.
- `SOL-05` Compose the selected review prompt with the existing immutable snapshot/schema instructions. The built-in document prompt explicitly names the eligible paths and asks only for consistency, contradictions, material open questions, and Memory Bank principles.
- `SOL-06` Add `--document-fix-prompt-file <path>`, valid only with `--document-review`. It replaces the built-in document fix prompt; it conflicts with `--fix-prompt-file`. Ordinary fix behavior remains unchanged.

## Alternatives and trade-offs

| ID | Alternative | Decision |
| --- | --- | --- |
| `ALT-01` | One overloaded selector argument | Rejected: individual flags make conflicts and provenance deterministic. |
| `ALT-02` | Always use project default or silently fall back after a selected file error | Rejected: explicit selection must fail closed; only absence of the document-mode default has a built-in fallback. |
| `TRD-01` | Permit explicit absolute/relative prompt paths but constrain names to project-local files | Chosen: direct path fulfills the requested file mode; names remain safe and reproducible. |

## Architecture coverage

| Aspect | Status | Canonical refs | Note |
| --- | --- | --- | --- |
| Components | covered | `SOL-01`–`SOL-06` | app parses commands; config resolves file contents; repository derives scope; adapter composes prompts. |
| Connectors | covered | `CTR-01` | Local filesystem read/write and existing Codex stdin are synchronous. |
| Configuration | covered | `SD-01` | New selectors deliberately do not join persistent YAML/env precedence. |
| Behavioral semantics | covered | `INV-01`–`INV-04`, `FM-01`–`FM-03` | Selection, scope and fix transitions are deterministic. |
| Quality / evolution | covered | `TRD-01`, `RB-01` | No migration; removal is a source revert. |

## Accepted local decisions, contracts and invariants

- `SD-01` Prompt selectors are CLI-only to avoid hidden precedence and to keep ordinary config output stable.
- `CTR-01` The adapter receives a resolved prompt plus a review target; user prompt text supplements but cannot replace the snapshot, schema, or no-mutation instructions.
- `INV-01` Ordinary review and `--fix-prompt-file` preserve their current behavior when document flags are absent.
- `INV-02` Any invalid selector, conflict, or selected-file read error exits `2` before Codex starts.
- `INV-03` Document mode never passes non-Markdown or excluded prompt-catalog files to Codex.
- `INV-04` Empty document scope is clean and non-publishing behavior remains owned by the existing workflow.
- `FM-01` Missing, unreadable, directory, non-Markdown, invalid-name, or write failure → actionable operational exit `2`.
- `FM-02` Two review selectors, document fix without document mode, or both fix selectors → actionable operational exit `2`.
- `FM-03` No eligible documents → structured `scope-empty` diagnostic and no Codex process.
- `RB-01` Backout is one source revert; no config migration, repository mutation, or remote state is introduced.

## Design verification

| Analysis | Required | Method / result |
| --- | --- | --- |
| Contract compatibility | yes | Preserve default flags/config/schema; table tests and README comparison required. |
| State / transition completeness | yes | Workflow tests cover clean empty scope and document findings/fix/review. |
| Failure propagation | yes | Table tests cover every selector and export failure; errors fail before Codex. |
| Concurrency / ordering | no | Existing sequential workflow only; no shared mutable state is added. |
| Security boundaries | no | Prompt content is local user input; no auth/trust boundary changes. |
| Capacity / latency | no | One additional Git file-list call is bounded by the existing review snapshot. |
| Migration / evolution safety | yes | Default remains ordinary review; no persistent setting is introduced. |

## Traceability

| Requirement | Solution refs |
| --- | --- |
| `REQ-01`–`REQ-02` | `SOL-01`, `SOL-02`, `TRD-01`, `INV-02` |
| `REQ-03`–`REQ-04` | `SOL-04`, `SOL-05`, `CTR-01`, `INV-03`, `INV-04` |
| `REQ-05` | `SOL-03`, `FM-01` |
| `REQ-06` | `SOL-06`, `FM-02` |
| `REQ-07` | `SD-01`, `RB-01` |
