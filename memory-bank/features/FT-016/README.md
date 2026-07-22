---
title: "FT-016: Review branch commits and working-tree changes against the intended PR base"
doc_kind: feature
doc_function: index
purpose: "Навигация по feature package issue #16: полный review scope от merge-base выбранного base до текущего worktree."
derived_from:
  - ../../flows/feature.md
  - brief.md
status: active
audience: humans_and_agents
---

# FT-016: Review branch commits and working-tree changes against the intended PR base

Пакет ведёт [issue #16](https://github.com/dapi/code-converge/issues/16) как одну delivery-unit по Feature Flow. `brief.md` остаётся единственным owner problem space и verify contract; selected solution живёт в `design.md`, а execution sequencing — в `implementation-plan.md`.

## Аннотированный индекс

- [`brief.md`](brief.md) — canonical problem/verify owner, route, profile и feature decisions.
- [`decision-log.md`](decision-log.md) — FPF reasoning, доступные факты, варианты и gate provenance.
- [`design.md`](design.md) — selected public contract, architecture coverage, Git/provider boundaries и failure semantics.
- [`implementation-plan.md`](implementation-plan.md) — grounded execution sequence, checkpoints и evidence realization.

`HG-01` закрыт FPF decision `DL-03`; feature готова к execution planning.
