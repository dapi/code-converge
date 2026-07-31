---
title: "FT-039: Deterministic delivery orchestration"
doc_kind: feature
doc_function: canonical
purpose: "Canonical problem, scope, validation profile and verification contract for GH-39."
derived_from:
  - ../../flows/feature.md
  - ../../engineering/testing-policy.md
  - ../../../README.md
  - https://github.com/dapi/code-converge/issues/39
status: active
delivery_status: in_progress
audience: humans_and_agents
must_not_define:
  - implementation_sequence
  - solution_space
---

# FT-039: Deterministic delivery orchestration

## What

Codex currently owns deterministic commit, push, pull-request and CI operations. Its sandbox and session lifetime make those host-process responsibilities unreliable. Code Converge must perform and classify them after a clean review, retaining Codex for review and remediation.

## Scope

- `REQ-01` Remove the Codex Finalize stage and obsolete finalize configuration without silent no-op compatibility.
- `REQ-02` After a clean review, repository code safely commits only a clean worktree, resolves branch/remote, pushes, and finds or creates one matching GitHub PR.
- `REQ-03` Wait for checks pinned to the published head SHA, classifying green/skipped, failed, timeout, provider failure and cancellation deterministically.
- `REQ-04` Add `--ci-timeout`, `CODE_CONVERGE_CI_TIMEOUT`, and `.code-converge/ci-timeout`, defaulting to `60m` under existing precedence.
- `REQ-05` A failed check enters Fix CI; timeout is operational and never invokes Fix CI.

## Non-Scope

- `NS-01` Other hosting providers, CI providers, or broader Codex remediation redesign.
- `NS-02` Automatically committing a dirty worktree that existed before publication.

## Design Requirement Decision

| Decision | Reason | Downstream owner |
| --- | --- | --- |
| `Design required: yes` | CLI/config/event contracts, workflow transitions, provider connector and timeout semantics change. | `design.md` |

## Validation Profile Decision

Validation profile: `standard`.

Triggers / rationale: public workflow/configuration/event contracts and GitHub integration require end-to-end fake coverage.

Downgrade approval: none.

## Verify

| Scenario | Observable result |
| --- | --- |
| `SC-01` | Clean reviewed changes are committed/pushed and one matching PR is reused or created without a Codex finalizer. |
| `SC-02` | A failed head-pinned check promptly starts Fix CI; a clean fix returns to review and publication. |
| `SC-03` | All successful/skipped head-pinned checks succeed; no checks skip; deadline emits timeout/exit 2. |
| `SC-04` | Dirty worktree, ambiguous identity, provider errors and cancellation fail safely. |

| Check | Evidence |
| --- | --- |
| `CHK-01` | `go test ./...` |
| `CHK-02` | `go vet ./...` |
| `CHK-03` | `make docs-lint` |
| `CHK-04` | `git diff --check` |
