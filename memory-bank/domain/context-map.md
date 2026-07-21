---
title: Domain Context Map
doc_kind: domain
doc_function: canonical
purpose: Reviewer domain-context ownership and its boundary with external agent, Git, repository-hosting, and CI systems.
derived_from:
  - ../dna/governance.md
  - glossary.md
  - model.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - bounded_contexts
  - domain_context_map
---

# Domain Context Map

The current product has one domain context. Codex, Git, repository hosting, and CI are external systems, not reviewer-owned bounded contexts. Runtime modules and process connectors are documented in [`../engineering/architecture.md`](../engineering/architecture.md).

## Bounded Contexts

| Context | Owns language / rules for | Upstream contexts | Downstream contexts | Must not know |
| --- | --- | --- | --- | --- |
| `Review Orchestration` | Run, stage, review cycle, finding/severity, finalization verdict, workflow transitions, exit outcomes, and configuration precedence | No other reviewer-owned context | No other reviewer-owned context | Internal state or credentials of Codex, Git, repository hosting, or CI |

## Context Relationships

| Relationship ID | Upstream | Downstream | Contract | Notes |
| --- | --- | --- | --- | --- |
| N/A | The project has no relationship between multiple internal bounded contexts. | N/A | N/A | External process/integration contracts are engineering boundaries, not additional domain contexts. |

## Shared Kernel / Published Language

- Shared kernel: N/A while reviewer has one domain context.
- Published language: the root [`README.md`](../../README.md) solely owns public CLI option names, exit codes, finalization verdicts, and stdout fields. Domain documents only interpret their meaning.

## Boundary Rules

- `Review Orchestration` owns interpretation of agent results and selection of the next workflow state; external systems own their own state.
- Credentials are environment prerequisites and must not become domain data or be exposed in output.
- Technical packages may split the single domain context by responsibility without creating new bounded contexts; those boundaries belong to [`../engineering/architecture.md`](../engineering/architecture.md).

## Open Boundary Questions

- No unresolved internal bounded-context ownership question is currently known.
- Exact external-process protocols and failure semantics are engineering/contract decisions, not context-map questions.
