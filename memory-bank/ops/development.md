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

The repository currently contains its public specification and adapted Memory Bank. The Go module and implementation commands do not exist yet and are outside the Memory Bank adoption scope.

## Prerequisites

- Go `1.21.13` or a compatible newer toolchain for the current documentation checks. This is a tooling pin, not the future application's minimum supported Go version.
- The application's minimum Go version remains undecided and must be selected when implementation begins.
- Local Git access for development. Remote-hosting credentials are not needed for Memory Bank adoption.
- `codex` installed and authenticated for manual Codex adapter verification.

## Documentation checks

```sh
make docs-lint
git diff --check
```

## Future implementation checks

Once the Go module exists, the expected baseline is:

```sh
go test ./...
go vet ./...
make docs-lint
```

Hosted CI and manual publication tests are optional future validation surfaces. If later required, use a disposable repository and non-default branch; never exercise commit/push/change-request behavior against a production repository merely to prove local setup.
