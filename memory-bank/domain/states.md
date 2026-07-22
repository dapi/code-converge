---
title: Workflow States
doc_kind: domain
doc_function: canonical
purpose: Defines the code-converge run state machine.
derived_from:
  - model.md
  - rules.md
status: active
audience: humans_and_agents
canonical_for:
  - state_machine
  - state_transitions
---

# Workflow States

```mermaid
stateDiagram-v2
  [*] --> Review
  Review --> FixFindings: findings and fix budget remaining
  FixFindings --> Review: success
  Review --> Finalize: clean report and changes exist
  Review --> Exit0: clean report and no changes
  Review --> Exit1: findings after final fix
  Review --> Exit2: command/report failure
  FixFindings --> Exit2: command failure
  Finalize --> Exit0: SUCCESS
  Finalize --> FixCI: CI_FAILED and recovery budget remains
  Finalize --> Exit3: CI_FAILED and recovery budget exhausted
  Finalize --> Exit2: FAILED
  FixCI --> Review: success
  FixCI --> Exit3: failure
```

CI transitions are applicable only when the target repository has required CI. When no required CI exists, finalization reports success with the CI step marked `skipped`. Hosting-provider-specific behavior is an adapter concern, not a domain state.
