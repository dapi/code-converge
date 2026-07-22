---
title: Product Metrics
doc_kind: product
doc_function: canonical
purpose: Defines observable code-converge run metrics and their ownership.
derived_from:
  - context.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - product_metrics
---

# Product Metrics

| Metric | Definition | Source | Interpretation |
| --- | --- | --- | --- |
| Findings total | Findings in a completed review | `findings_total` in `kv`; human review summary | Must be visible across cycles; it is not by itself a success metric. |
| Findings by severity | Counts for critical/high/medium/low/unknown | `findings_*` in `kv`; non-zero buckets in human output | Shows how the remaining risk profile changes; monotonic decrease is not required. |
| Stage duration | Wall duration of a completed stage | `duration_ms` in `kv`; readable duration in human output | Measures review, fixing, finalization, and CI-recovery cost. |
| Run outcome | Final exit code and total duration | terminal event | Distinguishes successful completion from defined failure modes. |

Metrics are emitted to stdout for a single run. Persistent collection, dashboards, and cross-run aggregation are out of scope until separately designed.
