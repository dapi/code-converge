---
title: Domain Model
doc_kind: domain
doc_function: canonical
purpose: Conceptual model of code-converge workflow entities and verdicts.
derived_from:
  - ../dna/governance.md
  - glossary.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - domain_model
  - domain_concepts
---

# Domain Model

| Concept | Kind | Represents | Key relationships |
| --- | --- | --- | --- |
| Run | aggregate | One invocation of `code-converge` | Contains ordered stages and review cycles. |
| Review phase | value | A bounded review/fix convergence sequence | Starts initially and again after a successful CI recovery. |
| Review cycle | value | One review followed by an optional finding fix | Belongs to a review phase. |
| Finding | value | A code-review remark parsed from the schema-valid Codex final response | Has one normalized severity; contributes to cycle counts. |
| Stage | stateful operation | Review, fix findings, publish, wait for CI, or CI fix | Produces a typed result and duration. |
| CI outcome | value | `success`, `skipped`, `failed`, or `timeout` | Determines terminal success, CI recovery, or operational exit `2`. |
| Configuration value | value | Option plus source | Resolves once per run and is shown by `code-converge config`. |

## Boundaries

- Codex, Git remotes, hosting providers, and CI systems are external. `code-converge` owns deterministic Git/GitHub invocation, retry, and classification; Codex owns review and remediation. No particular hosting provider is a required Memory Bank or domain boundary.
- A finding is not a persistent issue tracker item; it exists as a classified result for the current run.
