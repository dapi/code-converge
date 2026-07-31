---
title: "FT-038: YAML Configuration Design"
doc_kind: feature
doc_function: canonical
purpose: "Selected YAML configuration contract and compatibility decision for FT-038."
derived_from:
  - brief.md
  - ../../../README.md
status: active
audience: humans_and_agents
must_not_define:
  - problem_scope
  - execution_sequence
---

# FT-038: YAML Configuration Design

- `SOL-01` Resolve only `<config-dir>/config.yaml` for file configuration. The resolver retains source metadata and existing precedence.
- `SOL-02` Decode a strict flat mapping whose key inventory is the public settings table. Scalar values are validated by their existing typed setting validators; unknown, duplicate, malformed and nested YAML fails with file-and-line diagnostics.
- `SOL-03` Represent prompts with `fix-prompt-file`, `finalize-prompt-file` and `ci-fix-prompt-file`; each value is a path resolved relative to its YAML directory unless absolute.
- `SD-01` No legacy read, migration or fallback is allowed.
- `SD-02` This breaking file-format change requires the next SemVer major release; it is recorded under Unreleased until that release is prepared.

## C4 Applicability Decision

`C4-00: not required` — the change remains within the `internal/config` resolver and adds no runtime, storage or integration boundary.

## Architecture Coverage

| Aspect | Decision |
| --- | --- |
| Components | `internal/config` owns decoding, source attribution and validation. |
| Connectors | N/A; local filesystem reads already exist. |
| Configuration | Project and user directories bind to one named document each. |
| Behavioral semantics | Precedence and CLI/environment contracts remain invariant; legacy files are excluded. |
| Evolution | Strict keys catch configuration drift; major release avoids silent compatibility claims. |
