---
title: Domain Glossary
doc_kind: domain
doc_function: canonical
purpose: Canonical reviewer workflow terminology and naming distinctions.
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
| `run` | One invocation of the main `reviewer` workflow from start to a terminal outcome. | Workflow, logs, exit policy | A single Codex subprocess invocation |
| `stage` | One review, fix-findings, finalization, or CI-fix operation within a run. | Workflow and timing | A deployment environment |
| `review` | The stage that asks the configured agent to inspect the current repository and reports zero or more findings. | Review workflow | A hosted change-request approval or human review |
| `finding` | One code-review issue reported for the current review. It contributes to the review's total and one severity bucket. | Review result and metrics | A persistent issue-tracker item |
| `severity` | The finding classification counted as `critical`, `high`, `medium`, `low`, or `unknown` in the public reporting contract. | Review metrics | Agent reasoning effort or process exit status |
| `clean review` | A completed review with zero findings. | Transition into finalization | A successful overall run |
| `review cycle` | One review attempt and, when permitted and needed, its following fix-findings attempt. | Cycle limit and trend reporting | A CI-recovery attempt or the whole run |
| `fix findings` | The stage that asks the agent to address findings from the preceding review. | Review loop | CI recovery |
| `finalization` | The stage after a clean review that asks the agent to commit, push, create a hosted change request when needed, and establish the CI result. | Publication workflow | Process cleanup or merely exiting the CLI |
| `finalization verdict` | One of `SUCCESS`, `CI_FAILED`, or `FAILED`, used to select the next workflow transition. | Finalization | The CLI process exit code |
| `CI recovery` | The fix-CI stage entered after finalization reports `CI_FAILED`; a successful recovery returns the run to review. | CI failure path | Re-running CI without reviewing resulting changes |
| `effective configuration` | The resolved value and source for each setting after precedence is applied. | `reviewer config` and run setup | A single config file's contents |

## Naming Rules

- Use `finding`, not `remark`, `comment`, or `issue`, when referring to a review result counted by the workflow.
- Use the stage names `review`, `fix-findings`, `finalize`, and `fix-ci` in externally visible records unless the public log contract changes.
- Do not use `success` without identifying whether it means a successful stage, finalization verdict, or terminal run outcome.

## Ambiguous Terms

| Term | Allowed meaning | Forbidden / overloaded meaning | Replacement |
| --- | --- | --- | --- |
| `cycle` | Review cycle as defined above | Whole run or CI recovery | `run`, `review cycle`, or `CI recovery` |
| `success` | Qualified success of a named stage or run | Any agent process that exited without proving the required outcome | `stage success`, `SUCCESS` verdict, or `run success` |
| `CI failed` | The `CI_FAILED` finalization verdict when publication succeeded but CI is red | Any failure to invoke, inspect, or repair CI | Name the process/integration failure explicitly |
| `reviewer` | The CLI/project | A human code reviewer | `human reviewer` for the person |

## Source Documents

- [`model.md`](model.md)
- [`rules.md`](rules.md)
- [`states.md`](states.md)
- [`../../README.md`](../../README.md)

No external domain research, legal definition, or legacy glossary has been supplied.
