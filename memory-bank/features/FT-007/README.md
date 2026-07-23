---
title: "FT-007: Rename Project to code-converge"
doc_kind: feature
doc_function: index
purpose: "Навигация по завершённому feature package переименования проекта, CLI и публичных контрактов из reviewer в code-converge."
derived_from:
  - ../../flows/feature.md
  - brief.md
status: active
audience: humans_and_agents
---

# FT-007: Rename Project to `code-converge`

Пакет вёл [issue #7](https://github.com/dapi/code-converge/issues/7) как одну delivery-unit по Feature Flow. Публичный CLI-контракт остаётся у корневого [`README.md`](../../../README.md), а этот пакет владеет feature-specific scope, solution decisions, execution handoff и evidence contract.

## Аннотированный индекс

- [`brief.md`](brief.md) — canonical problem space, scope, blocking decisions, validation profile и verify contract.
- [`decision-log.md`](decision-log.md) — FPF provenance, review-improve findings и human gate.
- [`design.md`](design.md) — solution-space owner для identity migration, compatibility policy и repository/release boundaries.
- [`implementation-plan.md`](implementation-plan.md) — archived execution sequence, test strategy и PR handoff.
