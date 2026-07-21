---
title: Product Vision
doc_kind: product
doc_function: canonical
purpose: Confirmed product promise, experience principles, non-goals, and the intentionally bounded scope of code-converge.
derived_from:
  - ../dna/governance.md
  - context.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - product_vision
  - product_strategy_principles
---

# Product Vision

`code-converge` aims to make the local agent-development loop repeatable and observable: review the current repository, address findings, publish the result, and recover red CI through one CLI workflow. Codex is the supported agent, while Git hosting, CI, and the coding agent remain external systems.

The documented CLI is the complete intended product, not a minimum version of a broader platform. `code-converge` is expected to remain a small, bounded utility. This document must not be used to infer a hosted product, business model, market expansion, support for additional agents, or a continuing feature roadmap.

## Product Promise

An operator should receive an explicit terminal result rather than having to interpret agent prose or manually coordinate every transition. During a run, one-line progress records expose review finding counts, severity, and stage durations.

## Strategic Bets

| Bet ID | Bet | Why now | Evidence | Review cadence |
| --- | --- | --- | --- | --- |
| `BET-01` | Deliver the complete local review → fix → finalization → CI-recovery loop as one bounded utility. | This is the intended product outcome, not a stepping stone to broader scope. | [`context.md`](context.md), [`../../README.md`](../../README.md) | No ongoing feature cadence is planned. |
| `BET-02` | Treat explicit outcomes and observable review trends as part of the product contract. | Operators must distinguish success from unresolved findings or stage failure. | [`metrics.md`](metrics.md), [`../../README.md`](../../README.md) | Revisit when implementation evidence exists. |

## Experience Principles

- `XP-01` A run never reports success unless it reaches the documented successful terminal state.
- `XP-02` Important stage progress and completion facts remain visible as one-line stdout records.
- `XP-03` Configuration is inspectable before execution, including the source of each effective value.

## Product Non-Goals

- `PNG-01` Replacing Codex, Git, Git hosting, CI, or a task tracker.
- `PNG-02` Hosted dashboards, persistent metric storage, or cross-run analytics.
- `PNG-03` Support for non-Codex agents.
- `PNG-04` A business model or enterprise operating model; neither has been defined.

## Decision Rules

- Prefer work required to make a core workflow in [`context.md`](context.md) correct, observable, and testable over unvalidated expansion.
- A change to CLI behavior, configuration, exit codes, logs, or external-agent integration must update its canonical contract and use the repository's feature flow.
- Do not prioritize a new segment, channel, agent, or hosted surface without product evidence and an explicit scope decision.

## Source Documents

- [`context.md`](context.md)
- [`metrics.md`](metrics.md)
- [`../../README.md`](../../README.md)

No broader product strategy or continuing expansion roadmap is planned. Any future scope expansion would require a new explicit product decision.
