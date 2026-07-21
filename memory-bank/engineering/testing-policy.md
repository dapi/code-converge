---
title: Testing Policy
doc_kind: engineering
doc_function: canonical
purpose: Required validation for reviewer behavior and documentation changes.
derived_from:
  - architecture.md
  - validation-profiles.md
status: active
audience: humans_and_agents
canonical_for:
  - testing_policy
---

# Testing Policy

When implementation begins, it must use deterministic tests with a fake runner and a fake `codex` executable. Automated tests must not invoke a real Codex session, mutate a real remote, create a change request, or wait for hosted CI.

| Change surface | Minimum automated evidence |
| --- | --- |
| Workflow transition or exit code | Table-driven state-machine tests for positive, terminal, and malformed/error cases. |
| Codex command boundary | Fake-executable tests for arguments, model/reasoning configuration, prompt/stdin, cwd, stdout/stderr capture, exit status, and malformed output; timeout and cancellation tests become required if those policies are selected. |
| Review report classification | Plain-text fixtures for clean, findings with every supported priority, unknown priority, ambiguous report, and non-zero exit. |
| Configuration | Table tests covering every source and precedence conflict. |
| Stdout event schema | Golden tests proving one line per record and required fields. |
| Repository documentation | `make docs-lint`. |

Before publishing an implementation change, run `go test ./...`, `go vet ./...`, `make docs-lint`, and `git diff --check` when the corresponding project artifacts exist. Hosted CI is additional evidence only when the repository has it. Record unavailable commands as an explicit gap, never as a passing check.
