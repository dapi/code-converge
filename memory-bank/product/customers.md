---
title: Customers And Users
doc_kind: product
doc_function: canonical
purpose: Confirmed code-converge users, their jobs, current evidence, and assumptions that must remain unvalidated until researched.
derived_from:
  - ../dna/governance.md
  - context.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - product_customers
  - user_segments
  - jobs_to_be_done
---

# Customers And Users

The product specification identifies teams using coding agents as the intended audience. No interviews, usage analytics, buyer research, or segment prioritization evidence has been supplied.

## Segments

| Segment ID | Segment | Job To Be Done | Current Pain | Success Signal | Evidence |
| --- | --- | --- | --- | --- | --- |
| `SEG-01` | Developers or teams using coding agents in Git repositories | Move an agent-assisted change from review through fixes and publication, including green required CI when it applies | The steps are manually coordinated and stage outcomes are easy to misinterpret | A run exposes progress and reaches an explicit documented terminal result | Product specification only; customer validation is `unknown` |

## Users And Actors

| Actor ID | Actor | Uses product how | Decision power | Notes |
| --- | --- | --- | --- | --- |
| `ACT-01` | CLI operator | Runs `code-converge` in the target Git repository, selects configuration, and reads stdout/exit status | Operator; purchasing authority is `unknown` | Supplies local agent authentication and Git/hosting access required by the selected actions |

Если actor становится участником устойчивого сценария, use case фиксируй в [`../use-cases/README.md`](../use-cases/README.md).

## Research Inputs

- Current evidence: [`context.md`](context.md) and the public contract in [`../../README.md`](../../README.md).
- Customer interviews: none supplied.
- Usage analytics, support tickets, sales notes, and usability studies: none supplied.

## Assumptions

- `ASM-01` The intended segment experiences enough repeated manual coordination to adopt a dedicated CLI — `unvalidated`.
- `ASM-02` A locally installed Codex integration is an acceptable default for the intended segment — `unvalidated`.
- `ASM-03` One-line stdout events and exit codes are sufficient for initial operator observability — specified, but not user-validated.

## Must Not Assume

- `NA-01` Do not assume a buyer, purchasing process, pricing model, or enterprise rollout requirement.
- `NA-02` Do not assume demand for hosted execution, a graphical UI, additional agents, or persistent analytics.
- `NA-03` Do not infer user satisfaction or time savings from the existence of the specification.
