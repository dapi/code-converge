---
title: Domain Rules
doc_kind: domain
doc_function: canonical
purpose: Invariants and terminal verdict rules for code-converge runs.
derived_from:
  - model.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - domain_invariants
  - business_rules
---

# Domain Rules

- `RULE-01`: Finalization starts only after a completed review with zero findings.
- `RULE-02`: `max-cycles` limits fix-findings attempts in one review phase. The final allowed fix is followed by a verification review; remaining findings then exit `1`.
- `RULE-03`: A successful CI fix starts a new review phase with a fresh review budget, preserving the possibility that the fix introduced findings. `max-ci-recoveries` bounds these restarts.
- `RULE-04`: Only finalization may produce `SUCCESS`, `CI_FAILED`, or `FAILED`; an unrecognized agent response is not any of these verdicts.
- `RULE-05`: A successful finalization exits `0` when required CI is green or CI is not applicable. Operational/finalization failure exits `2`; failed or exhausted CI recovery exits `3`.
- `RULE-06`: Each successfully classified review emits total findings and zero-filled counts for `critical`, `high`, `medium`, `low`, and `unknown`. A failed or ambiguous review emits no unreliable counters.
- `RULE-07`: Each completed stage emits an elapsed duration; the terminal event emits total run duration and exit code.
- `RULE-08`: Effective configuration follows the precedence contract owned by [`../../README.md`](../../README.md).

Detailed user-facing values remain owned by [`../../README.md`](../../README.md); this document defines their domain interpretation rather than maintaining a second option or exit-code table.
