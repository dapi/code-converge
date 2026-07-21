---
title: Product Roadmap
doc_kind: product
doc_function: canonical
purpose: Current bounded reviewer delivery horizon and the explicit absence of planned product expansion.
derived_from:
  - ../dna/governance.md
  - context.md
  - vision.md
  - metrics.md
  - ../prd/PRD-001-reviewer-cli.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - product_roadmap
  - product_themes
---

# Product Roadmap

The roadmap contains one product outcome: deliver the complete `reviewer` CLI described by the current specification. The utility is not a foundation for a broader platform, and no continuing feature roadmap is planned.

## Horizons

| Horizon | Theme | Intended outcome | Current owner | Dependency | Status |
| --- | --- | --- | --- | --- | --- |
| `now` | Complete reviewer CLI | A distributable local Go CLI performs the full documented review/fix/finalization/CI-recovery workflow with observable terminal results | [`../prd/PRD-001-reviewer-cli.md`](../prd/PRD-001-reviewer-cli.md) | One Feature Flow package, implementation, tests, acceptance, and distribution evidence | active |

## Roadmap Rules

- Roadmap theme описывает product intent, а не implementation plan.
- Deliver the documented CLI as one coherent delivery-unit. Internal checkpoints do not become separate feature packages or an Epic unless scope materially changes.
- Review/fix, finalization, CI recovery, configuration, observability, and packaging converge in the same feature-level acceptance contract.
- Work beyond the documented CLI is not implied follow-up; route it separately only after an explicit product decision.
- Если тема меняет предметную модель, сначала обнови [`../domain/model.md`](../domain/model.md), [`../domain/rules.md`](../domain/rules.md) или [`../domain/context-map.md`](../domain/context-map.md).

## Completion Boundary

- No post-completion product themes are planned.
- Completion requires the full CLI contract, reproducible acceptance evidence, and distributable artifact evidence; partial workflow slices are not product milestones.
- Future defect fixes or compatibility updates are routed when they arise and do not constitute an ongoing product roadmap.
