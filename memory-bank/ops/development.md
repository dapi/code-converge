---
title: Development Environment
doc_kind: ops
doc_function: canonical
purpose: Local development prerequisites and verification commands for reviewer.
derived_from:
  - ../engineering/testing-policy.md
status: active
audience: humans_and_agents
canonical_for:
  - development_environment
---

# Development Environment

## Current state

The repository contains the public specification, governed Memory Bank, Go module, CLI implementation, deterministic tests, and distribution tooling.

## Prerequisites

- Go `1.21.13` or a compatible newer toolchain; this is the application's minimum supported build version.
- Local Git access for development. Remote-hosting credentials are needed only for publication work.
- `codex` installed and authenticated for manual Codex adapter verification.

## Required checks

```sh
make docs-lint
go test ./...
go vet ./...
git diff --check
```

The combined local baseline is:

```sh
make verify
git diff --check
```

`make dist` builds the supported artifact matrix. Hosted CI repeats verify, distribution, and Linux AMD64 archive smoke. Tests must not exercise commit/push/change-request behavior against a real remote merely to prove setup.
