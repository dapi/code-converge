---
title: Product Context
doc_kind: product
doc_function: canonical
purpose: Product-wide context for code-converge.
derived_from:
  - ../dna/governance.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - project_product_context
  - product_problem_space
  - top_level_outcomes
---

# Product Context

`code-converge` is a local Go CLI for teams using coding agents. It turns the repeated manual loop of code review, fixing findings, publishing changes, and checking applicable CI into a single observable workflow. The utility uses Codex.

The product boundary is orchestration of the local agent-development loop. It does not replace Git hosting, CI, the coding agent, or a task tracker. It must make the outcome and unresolved failures explicit rather than claiming success from agent prose.

## Core workflows

- `WF-01` Run review; when findings exist, fix them and repeat within the cycle limit.
- `WF-02` After a clean review, commit and push, create a change request when the hosting workflow needs one, and check required CI when it exists.
- `WF-03` When applicable required CI is red, attempt a bounded CI fix and restart at review.
- `WF-04` Inspect effective settings before a run with `code-converge config`.

## Product constraints

- `PCON-01` The supported agent integration is the locally installed and authenticated Codex CLI.
- `PCON-02` Every important step remains observable on stdout through an explicitly selected human or structured format; structured review trend data includes all severity counts and millisecond duration, while human output keeps the total, non-zero severities and readable duration.
- `PCON-03` Publication actions are delegated to the finalization agent and require credentials for the configured Git remote or hosting provider. No particular provider is required by the product contract.

## Sources

The current public contract is [`../../README.md`](../../README.md). This document adds product context and does not duplicate its option tables, exit-code table, or log schema. No separate customer research or analytics dashboard has been supplied yet.
