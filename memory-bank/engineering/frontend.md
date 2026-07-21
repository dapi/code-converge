---
title: Frontend Engineering
doc_kind: engineering
doc_function: canonical
purpose: Records frontend applicability for the current code-converge CLI.
derived_from:
  - ../dna/governance.md
  - ../product/context.md
  - ../../README.md
status: active
audience: humans_and_agents
---

# Frontend Engineering

`code-converge` currently has no web, desktop, mobile, or other graphical frontend. Its user surface is a command-line interface, stdout records, process exit status, and local configuration files. The root [`README.md`](../../README.md) owns those public contracts; engineering and domain documents only interpret or implement them.

## UI Surfaces

- Graphical UI surfaces: N/A.
- Frontend code and stack: N/A.
- Design system and shared UI components: N/A.
- The CLI/output surface is covered by the project README and [`architecture.md`](architecture.md).

## Component And Styling Rules

N/A. Do not introduce a frontend framework, design system, UI component hierarchy, or CSS conventions without an explicit product and architecture decision.

## Interaction Patterns

N/A. No browser or native interaction pattern exists.

## Localization

No localization contract is defined. The built-in prompts currently include both English and Russian text, but prompt language is configuration/content, not a frontend i18n system.

If a graphical UI is proposed, update this document and replace the current N/A record in [`UI Design Guide`](ui-design-guide/README.md) as part of that explicitly governed change.
