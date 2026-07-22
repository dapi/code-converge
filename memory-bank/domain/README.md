---
title: Domain Documentation Index
doc_kind: domain
doc_function: index
purpose: Навигация по domain-level документации code-converge. Читать для определения языка workflow, правил, состояний и границ предметной области.
derived_from:
  - ../dna/governance.md
status: active
audience: humans_and_agents
---

# Domain Documentation Index

Каталог `memory-bank/domain/` хранит предметную модель `code-converge`: язык workflow, сущности запуска, правила, состояния и границы. Этот слой описывает то, что должно оставаться истинным независимо от технической реализации.

Domain-документы не определяют market positioning, product metrics, UI design system, concurrency pattern, deployment config или implementation sequence.

## На Какие Вопросы Отвечает Domain

- Какие понятия существуют в предметной области и что они означают?
- Какие сущности, value objects, actors или aggregates важны для reasoning?
- Какие бизнес-правила и инварианты нельзя нарушать?
- Какие состояния и переходы допустимы?
- Какие domain events являются бизнес-значимыми фактами?
- Где проходят bounded contexts и language boundaries?

## Граница С `product/`

| Layer | Отвечает на вопросы | Не отвечает на вопросы |
| --- | --- | --- |
| `product/` | Зачем существует продукт, для кого он, какие outcomes и metrics важны | Какие domain entities, states, invariants и events существуют |
| `domain/` | Что истинно в предметной области и какие правила обязана соблюдать система | Почему именно эта аудитория приоритетна, как продукт позиционируется, какой roadmap выбран |

Пример для `code-converge`:

- Product: уменьшить ручную координацию agent-development loop.
- Domain: finalization начинается только после review без findings.

## Граница С Engineering

- `domain/context-map.md` описывает business bounded contexts и language ownership.
- `engineering/architecture.md` описывает code/module boundaries, runtime patterns, concurrency, error handling и configuration ownership.
- Если документ отвечает на вопрос "какое бизнес-правило истинно?", он принадлежит `domain/`.
- Если документ отвечает на вопрос "как это безопасно реализовать в системе?", он принадлежит `engineering/`.

## Аннотированный Индекс

- [Glossary](glossary.md) — ubiquitous language, термины, запрещенные двусмысленности и canonical names.
- [Domain Model](model.md) — ключевые domain concepts, relationships, ownership и model notes.
- [Domain Rules](rules.md) — бизнес-правила, инварианты, policies и rule ownership.
- [States](states.md) — lifecycle states, allowed transitions и terminal states.
- [Events](events.md) — отсутствие отдельного domain-event integration contract и граница с operational stdout presentation.
- [Context Map](context-map.md) — единый domain context code-converge и его границы с внешними системами.
