---
title: Domain Glossary
doc_kind: domain
doc_function: canonical
purpose: Canonical code-converge workflow terminology and naming distinctions.
derived_from:
  - ../dna/governance.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - ubiquitous_language
  - domain_terms
---

# Domain Glossary

These terms are used consistently across product, feature, engineering, and operations documents. Runtime identifiers may use language-appropriate forms, but must preserve the same meaning.

## Terms

| Term | Meaning | Context | Do not confuse with |
| --- | --- | --- | --- |
| `run` | One invocation of the main `code-converge` workflow from start to a terminal outcome. | Workflow, logs, exit policy | A single Codex subprocess invocation |
| `stage` | One review, fix-findings, publish, CI, or CI-fix operation within a run. | Workflow and timing | A deployment environment |
| `review` | The stage that asks the configured agent to inspect the current repository and reports zero or more findings. | Review workflow | A hosted change-request approval or human review |
| `finding` | One code-review issue reported for the current review. It contributes to the review's total and one severity bucket. | Review result and metrics | A persistent issue-tracker item |
| `severity` | The finding classification counted as `critical`, `high`, `medium`, `low`, or `unknown` in the public reporting contract. | Review metrics | Agent reasoning effort or process exit status |
| `clean review` | A completed review with zero findings. | Transition into host-owned publication | A successful overall run |
| `review cycle` | One review attempt and, when permitted and needed, its following fix-findings attempt. | Cycle limit and trend reporting | A CI-recovery attempt or the whole run |
| `fix findings` | The stage that asks the agent to address findings from the preceding review. | Review loop | CI recovery |
| `publication` | The host-owned stage after a clean review that safely commits eligible work, pushes, and reuses or creates one pull request. | Publication workflow | A Codex stage or a local checkpoint |
| `CI wait` | Host polling of applicable check-runs for the exact published head SHA. | CI workflow | A general repository-health query |
| `CI timeout` | The CI wait deadline elapsed before a terminal classification; it is operational, not red CI. | CI workflow | A failed check or CI recovery |
| `CI recovery` | The Fix-CI stage entered after deterministic CI polling finds a failed check; a successful recovery returns the run to review. | CI failure path | Re-running CI without reviewing resulting changes |
| `effective configuration` | The resolved value and source for each setting after precedence is applied. | `code-converge config` and run setup | A single config file's contents |

## Naming Rules

- Use `finding`, not `remark`, `comment`, or `issue`, when referring to a review result counted by the workflow.
- Use the stage names `review`, `fix-findings`, `publish`, `ci`, and `fix-ci` in externally visible records unless the public log contract changes.
- Do not use `success` without identifying whether it means a successful stage, finalization verdict, or terminal run outcome.

## Ambiguous Terms

| Term | Allowed meaning | Forbidden / overloaded meaning | Replacement |
| --- | --- | --- | --- |
| `cycle` | Review cycle as defined above | Whole run or CI recovery | `run`, `review cycle`, or `CI recovery` |
| `success` | Qualified success of a named stage or run | Any agent process that exited without proving the required outcome | `stage success` or `run success` |
| `CI failed` | A completed applicable check with an unaccepted conclusion | Timeout, provider failure, or cancellation | Name the process/integration failure explicitly |
| `code-converge` | The CLI/project | A human code code-converge | `human code-converge` for the person |

## Source Documents

- [`model.md`](model.md)
- [`rules.md`](rules.md)
- [`states.md`](states.md)
- [`../../README.md`](../../README.md)

No external domain research, legal definition, or legacy glossary has been supplied.
