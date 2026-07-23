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

- `RULE-01`: Finalization starts only after a completed review with zero findings and either Git status confirms staged, unstaged or untracked changes or the run created a local findings-fix checkpoint. A clean worktree with no checkpoint exits successfully without finalization.
- `RULE-02`: Before an automatic findings-fix stage, Git status determines checkpoint eligibility. A clean worktree may receive one local checkpoint commit after a successful fix; a dirty worktree still receives remediation but skips the checkpoint to avoid capturing pre-existing work. Checkpoints never push, checkpoint-operation failures are operational, and publication remains finalization after clean review.
- `RULE-03`: `max-cycles` limits fix-findings attempts in one review phase. The final allowed fix is followed by a verification review; remaining findings then exit `1`.
- `RULE-04`: A successful CI fix starts a new review phase with a fresh review budget, preserving the possibility that the fix introduced findings. `max-ci-recoveries` bounds these restarts.
- `RULE-05`: Only finalization may produce `SUCCESS`, `CI_FAILED`, or `FAILED`; an unrecognized agent response is not any of these verdicts.
- `RULE-06`: A successful finalization exits `0` when required CI is green or CI is not applicable. Operational/finalization failure exits `2`; failed or exhausted CI recovery exits `3`.
- `RULE-07`: Each successfully classified review emits total findings and zero-filled counts for `critical`, `high`, `medium`, `low`, and `unknown`. A failed or ambiguous review emits no unreliable counters.
- `RULE-08`: Each completed stage emits an elapsed duration; the terminal event emits total run duration and exit code.
- `RULE-09`: Effective configuration follows the precedence contract owned by [`../../README.md`](../../README.md).
- `RULE-10`: Each review resolves exactly one intended base and snapshots merge-base through the current worktree in a private index. Ambiguous or unresolved bases are operational failures; discovery never mutates the real index, worktree, remote refs or pull requests.

Detailed user-facing values remain owned by [`../../README.md`](../../README.md); this document defines their domain interpretation rather than maintaining a second option or exit-code table.
