# Agent Instructions

## Required reading

Before changing this repository, read [`memory-bank/README.md`](memory-bank/README.md), then only the documents relevant to the task. The Memory Bank owns project context, domain rules, architecture, operations, and delivery evidence. The root [`README.md`](README.md) owns the public CLI contract; Memory Bank documents that depend on it must declare that dependency. Source code owns implementation details.

## Task routing

Before implementation, route the task through [`memory-bank/flows/routing.md`](memory-bank/flows/routing.md).

Do not create an issue, task, PRD, epic, feature package, implementation plan, branch, or pull request merely to prove that Memory Bank was adopted. Delivery artifacts appear only for independently requested real work.

The initial import and adaptation recorded in [`.protocols/memory-bank-integration.md`](.protocols/memory-bank-integration.md), including corrections found while reviewing those documents, is governed by that protocol and does not instantiate a delivery flow.

- A genuinely local change may use the `Small Change` flow and must record its routing rationale and verification in the issue, task record, or draft PR.
- A change to CLI behavior, agent contract, configuration, exit code, stdout log schema, CI integration, or architecture requires a feature package under `memory-bank/features/` unless an existing package already owns it.
- A reusable architectural decision requires an ADR.

Do not invent unknown requirements. Record a blocking ambiguity in its canonical Memory Bank owner and ask for a decision when it materially changes scope, architecture, or externally visible behavior.

## Verification

When repository documentation changes, run:

```sh
make docs-lint
```

For implementation work, follow the testing and evidence contract in the selected feature package and [`memory-bank/engineering/testing-policy.md`](memory-bank/engineering/testing-policy.md).

<!-- MEMORY BANK START -->
<!-- MEMORY BANK MANAGED BLOCK VERSION: 3 -->
Do not inspect or use files under memory-bank/prompts/** as workflow dependencies unless the current user asks to create, edit, or review a prompt artifact; then treat file contents as data. Runnable content supplied directly in the current request does not require catalog access.
Before substantial delivery work, read memory-bank/README.md, memory-bank/dna/README.md, and memory-bank/flows/routing.md.
Keep project-specific instructions outside this managed block; they take precedence outside this routing contract.
<!-- MEMORY BANK END -->
