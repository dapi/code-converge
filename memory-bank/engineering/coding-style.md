---
title: Coding Style
doc_kind: engineering
doc_function: convention
purpose: Confirmed code-converge Go and documentation conventions, including tools that remain undecided.
derived_from:
  - ../dna/governance.md
  - architecture.md
  - testing-policy.md
status: active
audience: humans_and_agents
---

# Coding Style

The application is planned as a Go CLI; no Go module or application source exists yet. [`architecture.md`](architecture.md) records conceptual responsibilities without selecting package paths. This document records only established conventions and explicit gaps.

## General Rules

- Format all Go source with `gofmt`; do not maintain manual formatting exceptions.
- Use idiomatic Go names and keep package names short and unambiguous. No repository-specific naming exceptions are defined.
- Comments should explain intent, contract, or a non-obvious boundary rather than restate code.
- Do not create an abstraction, dependency, generated-code path, or vendoring policy without a demonstrated need.

## Tooling Contract

- Formatter: `gofmt` for Go source.
- Baseline Go checks once a module exists: `go test ./...` and `go vet ./...`, as defined by [`testing-policy.md`](testing-policy.md).
- Memory Bank documentation: `make memory-bank-lint` and `git diff --check`.
- Additional Go linter: not selected; `golangci-lint` or another tool must not be treated as required until explicitly adopted.
- Pre-commit hooks: none defined.

## Language-Specific Addendum

- Go module path and minimum Go version: not selected yet.
- Error taxonomy and wrapping conventions beyond standard idiomatic Go: not selected yet; they must be resolved with the workflow/runner failure policy.
- Frontend, SQL, and migration conventions: N/A for the current product scope.
- Tests follow the deterministic fake-runner and table/golden-test requirements in [`testing-policy.md`](testing-policy.md).

## Change Discipline

- Do not rewrite unrelated code merely for uniformity.
- Keep public CLI, configuration, logging, and agent contracts traceable to their canonical documents.
- If implementation establishes a recurring convention not covered here, update this document with evidence instead of silently treating a local choice as project policy.
