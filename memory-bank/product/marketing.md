---
title: Marketing And Positioning
doc_kind: product
doc_function: canonical
purpose: Current reviewer positioning, available channel evidence, and claims that remain unsupported.
derived_from:
  - ../dna/governance.md
  - context.md
  - customers.md
status: active
audience: humans_and_agents
canonical_for:
  - product_positioning
  - product_messaging
  - go_to_market_context
---

# Marketing And Positioning

`reviewer` is currently specified as a local CLI and has a source repository. There is no approved distribution model, go-to-market plan, launch date, pricing, or competitive research.

## Positioning

| Audience | Current alternative | Product difference | Proof |
| --- | --- | --- | --- |
| `SEG-01` | Manually invoking a coding agent and coordinating Git, hosted change-request, and CI steps | One command orchestrates the documented loop and exposes stage outcomes, finding trends, and terminal status | Product specification; implementation proof is not available yet |

## Messaging

- `MSG-01` Agentic CLI for automated review → fix → publish → CI loops, powered by Codex by default.
- `MSG-02` The workflow stays local and reports explicit outcomes and progress instead of leaving failures implicit in raw agent output.

## Channels

| Channel | Audience | Goal | Constraint | Owner |
| --- | --- | --- | --- | --- |
| Project source repository | Developers evaluating or contributing to the project | Discovery, source distribution, and documentation | No release artifacts or launch plan are defined | Repository owner; product marketing owner is `unknown` |

## Competitive Alternatives

- `ALT-01` The confirmed status quo is manual coordination of agent review, fixes, Git/hosting actions, and CI recovery.
- `ALT-02` Direct competitors and substitute tools have not been researched; do not claim differentiation beyond the specified workflow.

## Launch Constraints

- `LC-01` The MVP must be implemented and meet its documented acceptance/evidence contract before it is described as working software.
- `LC-02` An official distribution or release requires a separately defined release process; none exists today.
- `LC-03` Do not claim production readiness, adoption, reliability, time savings, or superiority without implementation and user evidence.
