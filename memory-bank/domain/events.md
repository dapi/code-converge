---
title: Domain Events
doc_kind: domain
doc_function: canonical
purpose: Records that code-converge currently has no separate domain-event integration contract and distinguishes it from stdout operational records.
derived_from:
  - ../dna/governance.md
  - model.md
  - rules.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - domain_events
  - business_events
---

# Domain Events

`code-converge` is specified as a sequential local CLI. It currently publishes no asynchronous, persisted, or cross-process domain events. Workflow transitions are canonical in [`states.md`](states.md).

The one-line stdout records required by the product are operational records, not a domain-event integration. Their externally visible fields, ordering, and encoding are owned solely by the root [`README.md`](../../README.md); domain and engineering documents may interpret or implement that contract but do not redefine it.

## Events

| Event ID | Event | Meaning | Producer | Consumers | Minimal facts |
| --- | --- | --- | --- | --- | --- |
| N/A | No domain event is currently defined. | N/A | N/A | N/A | N/A |

## Event Rules

- Do not promote an stdout line to a domain event merely because it is named `event` in its encoding.
- A future event integration requires an explicit contract for meaning, producer, consumers, minimum facts, compatibility, and delivery semantics.
- If a future event changes allowed transitions, update [`states.md`](states.md); if it transfers responsibility across contexts, update [`context-map.md`](context-map.md).

## Delivery Semantics

N/A while there is no domain-event integration. Technical stdout/process ordering and error handling belong to [`../engineering/architecture.md`](../engineering/architecture.md) and the canonical owner of the CLI/logging contract.
