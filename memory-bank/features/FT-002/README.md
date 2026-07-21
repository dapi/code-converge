---
title: "FT-002: Complete Reviewer CLI Feature Package"
doc_kind: feature
doc_function: index
purpose: "Навигация по документации delivery-единицы issue #2. Сначала читать canonical brief; decision log хранит review/FPF provenance, не подменяя canonical owners."
derived_from:
  - ../../dna/governance.md
  - brief.md
status: active
audience: humans_and_agents
---

# FT-002: Complete Reviewer CLI

## О разделе

Пакет ведёт issue [#2](https://github.com/dapi/reviewer/issues/2) по Feature Flow как одну проверяемую delivery-unit. Canonical problem space и verify contract находятся в `brief.md`. Solution и execution artifacts появляются только после закрытия material human gates.

## Аннотированный индекс

- [`brief.md`](brief.md)
  Canonical owner scope, constraints, validation profile, blockers и acceptance/evidence contract.
- [`decision-log.md`](decision-log.md)
  Feature-local журнал review/FPF reasoning и human gates. Не владеет requirements, selected solution или execution sequence.
- [`design.md`](design.md)
  Canonical solution owner: module boundaries, Codex protocols, workflow invariants, release baseline, failure and backout semantics.
- [`implementation-plan.md`](implementation-plan.md)
  Derived execution sequence, test surfaces, checkpoints and evidence mapping.
