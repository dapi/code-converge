---
title: Product Roadmap
doc_kind: product
doc_function: canonical
purpose: Current reviewer product horizon and the explicit absence of an approved post-MVP roadmap.
derived_from:
  - ../dna/governance.md
  - context.md
  - vision.md
  - metrics.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - product_roadmap
  - product_themes
---

# Product Roadmap

Only the MVP described by the current product specification is approved. No post-MVP product themes or dates have been supplied.

## Horizons

| Horizon | Theme | Intended outcome | Current owner | Dependency | Status |
| --- | --- | --- | --- | --- | --- |
| `now` | Reviewer MVP | A local Go CLI performs the documented review/fix/finalization/CI-recovery workflow with observable results | [`../../README.md`](../../README.md) | Product decisions, implementation, tests, and acceptance evidence | active |
| `next` | `unknown` | No approved outcome | `unknown` | Product evidence and an explicit prioritization decision | uncommitted |
| `later` | `unknown` | No approved outcome | `unknown` | Product evidence and an explicit strategy decision | uncommitted |

## Roadmap Rules

- Roadmap theme описывает product intent, а не implementation plan.
- Если тема требует отдельного delivery lifecycle, создай соответствующий governed artifact только после явного scope decision.
- Если тема меняет предметную модель, сначала обнови [`../domain/model.md`](../domain/model.md), [`../domain/rules.md`](../domain/rules.md) или [`../domain/context-map.md`](../domain/context-map.md).

## Open Bets

- No post-MVP bet is currently accepted.
- `OQ-01` Which user evidence and MVP usage signals should inform the next product theme?
- `OQ-02` Who owns post-MVP product prioritization and review cadence?
