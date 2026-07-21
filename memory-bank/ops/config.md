---
title: Configuration Contract
doc_kind: ops
doc_function: canonical
purpose: Operational configuration locations and prerequisites for code-converge.
derived_from:
  - ../domain/rules.md
  - ../../README.md
status: active
audience: humans_and_agents
canonical_for:
  - operational_configuration
---

# Configuration Contract

The root [`README.md`](../../README.md) solely owns configuration source precedence, option names, and built-in values. This document records operational interpretation without maintaining a second copy of that public contract.

`code-converge config` prints each effective value and its source. If the effective value differs from its built-in default, it prints that default too.

`codex` authentication and credentials for any configured Git remote or hosting provider are environment prerequisites, not `code-converge` configuration values. Provider-specific credentials are required only when the selected finalization workflow needs them. The application must not log secrets or token values.
