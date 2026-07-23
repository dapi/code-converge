---
title: Feature Packages Index
doc_kind: feature
doc_function: index
purpose: Навигация по instantiated feature packages. Читать, чтобы найти существующую delivery-единицу или понять, где создавать новую.
derived_from:
  - ../dna/governance.md
  - ../flows/feature.md
  - ../flows/feature-artifact-catalog.md
status: active
audience: humans_and_agents
---

# Feature Packages Index

Каталог `memory-bank/features/` хранит instantiated feature packages вида `FT-XXX/`.

## Rules

- Каждый package создается по правилам из [`../flows/feature.md`](../flows/feature.md).
- Optional problem, solution, execution и review artifacts выбираются по [`../flows/feature-artifact-catalog.md`](../flows/feature-artifact-catalog.md); каталог является меню, а не checklist.
- Bootstrap package начинается с `README.md` и `brief.md`; после `Problem Ready` в него добавляется `design.md`, если `brief.md` фиксирует `Design required: yes`; `implementation-plan.md` появляется после готовности нужных upstream owners.
- Для bootstrap и downstream-документов используй шаблоны из [`../flows/templates/feature/`](../flows/templates/feature/).
- Если работа требует roadmap, risk register и нескольких delivery subissues, сначала создай или обнови epic package в [`../epics/README.md`](../epics/README.md).
- По умолчанию feature ссылается на общий product context из [`../product/context.md`](../product/context.md), а при изменении предметных правил также на соответствующие документы из [`../domain/README.md`](../domain/README.md).
- Если feature реализует или существенно меняет устойчивый сценарий проекта, она должна ссылаться на соответствующий `UC-*` из [`../use-cases/README.md`](../use-cases/README.md).

## Naming

- Базовый формат: `FT-XXX/`
- Вместо `XXX` используй идентификатор, принятый в проекте: issue id, ticket id или другой стабильный ключ
- Один package = одна delivery-единица

## Current state

- [`FT-002/README.md`](FT-002/README.md) — complete Code-Converge CLI delivery for issue #2.
- [`FT-003/README.md`](FT-003/README.md) — automated GitHub Release delivery.
- [`FT-005/README.md`](FT-005/README.md) — fast/best model and reasoning-effort profiles for issue #5.
- [`FT-007/README.md`](FT-007/README.md) — completed project, CLI, configuration and release identity migration for issue #7.
- [`FT-009/README.md`](FT-009/README.md) — planned human/kv progress formats and bounded liveness indicators for issue #9.
- [`FT-010/README.md`](FT-010/README.md) — complete interactive agent-output terminal view delivery for issue #10.
- [`FT-012/README.md`](FT-012/README.md) — active self-update command delivery for issue #12, including verified release replacement and its public CLI contract.
- [`FT-014/README.md`](FT-014/README.md) — complete diagnostic session logs with retention, opt-out and private human path handoff for issue #14.
- [`FT-015/README.md`](FT-015/README.md) — complete strict structured Codex review classification and successful no-change completion for issue #15.
- [`FT-016/README.md`](FT-016/README.md) — planned complete branch-and-worktree review scope for issue #16; blocked at a public-contract human gate.
- [`FT-020/README.md`](FT-020/README.md) — complete conventional root `-h`/`--help` aliases for issue #20.
- [`FT-022/README.md`](FT-022/README.md) — complete schema-constrained Codex review result channel for issue #22.
- [`FT-024/README.md`](FT-024/README.md) — completed local checkpoints for successful findings fixes, with publication deferred to clean-review finalization for issue #24.
