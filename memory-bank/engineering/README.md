---
title: Engineering Documentation Index
doc_kind: engineering
doc_function: index
purpose: Навигация по code-converge engineering contracts and conventions.
derived_from:
  - ../dna/governance.md
status: active
audience: humans_and_agents
---

# Engineering Documentation Index

Каталог `memory-bank/engineering/` содержит инженерные границы и правила `code-converge`. Не выбранные инструменты или ещё не существующие runtime surfaces должны оставаться явно `unknown` или N/A.

- [Engineering Architecture Patterns](architecture.md) — code/module boundaries, runtime patterns, concurrency, error handling и configuration ownership. Domain bounded contexts живут отдельно в [`../domain/context-map.md`](../domain/context-map.md).
- [Frontend Engineering](frontend.md) — records that the current CLI has no graphical frontend surface.
- [UI Design Guide](ui-design-guide/README.md) — explicit N/A record for the current CLI-only product.
- [Testing Policy](testing-policy.md) — правила тестирования, обязательные automated tests, sufficient coverage. Отвечает на вопрос: когда feature обязана иметь test cases и когда допустим manual-only verify.
- [Validation Profiles](validation-profiles.md) — независимая от delivery flow глубина validation: taxonomy, risk triggers, minimum evidence contract и canonical owner решения.
- [Autonomy Boundaries](autonomy-boundaries.md) — границы автономии агента: автопилот, супервизия, эскалация. Отвечает на вопрос: что агент может делать сам, а где должен остановиться и спросить.
- [Coding Style](coding-style.md) — confirmed Go/documentation tooling and conventions; undecided tools remain explicit.
- [Git Workflow](git-workflow.md) — current default branch, verification expectations, and unspecified repository policies.
- [ADR](../adr/README.md) — instantiated Architecture Decision Records проекта.
