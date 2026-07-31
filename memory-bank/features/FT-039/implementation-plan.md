---
title: "FT-039: Implementation plan"
doc_kind: feature
doc_function: derived
purpose: "Execution sequence and verification for deterministic publication and CI orchestration."
derived_from:
  - brief.md
  - design.md
status: active
audience: humans_and_agents
---

# FT-039: Implementation plan

1. Remove finalization adapter/configuration and add `ci-timeout` resolution with precedence tests.
2. Add repository publication and GitHub-check polling with fake-runner coverage for Git identity, push, PR, head pinning, retries and cancellation.
3. Replace workflow finalization transitions/events with deterministic publication/CI transitions and retain Fix CI recovery.
4. Update public/canonical documentation and run `go test ./...`, `go vet ./...`, `make docs-lint`, and `git diff --check`.
