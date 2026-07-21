---
title: Memory Bank Integration Protocol
status: active
owner: reviewer
---

# Memory Bank Integration Protocol

## Goal

Make Memory Bank the versioned, project-specific governance and knowledge layer for `reviewer`. The root README remains the canonical public CLI contract; Memory Bank owns the deeper project context, engineering rules, operational constraints, lifecycle records, and delivery evidence. It guides humans and agents but is not a runtime dependency of the CLI.

## Implementation plan

1. Import `memory-bank/` from `https://github.com/dapi/memory-bank` as a local template snapshot and record its exact source revision in `memory-bank/UPSTREAM.md`.
2. Add a root `AGENTS.md` that makes `memory-bank/README.md` the agent entrypoint and requires task routing before non-trivial work.
3. Adapt the persistent `product/`, `domain/`, `engineering/`, and `ops/` layers to the actual `reviewer` contract. Do not invent unknown product facts; record unknowns as explicit open questions.
4. Keep delivery collections such as `features/`, `prd/`, and `epics/` empty until the project has an independently justified real task. Integration itself must not manufacture an issue or feature package.
5. Add reproducible local documentation checks. A hosted CI job is optional and only applies when the repository uses a compatible hosting/CI system.
6. Keep runtime integration separate: a future, independently scoped change may pass applicable Memory Bank context to Codex prompts.

## Acceptance criteria

| Area | Done when |
| --- | --- |
| Import | `memory-bank/` is present and `UPSTREAM.md` records the upstream URL, immutable revision, import date, and update policy. |
| Agent entrypoint | Root `AGENTS.md` directs agents to `memory-bank/README.md` and task routing before implementation. |
| Reviewer context | Product, domain, engineering, and operations documents describe the CLI's workflow, agent contracts, configuration, exit codes, logging, and known open questions without unsupported claims. |
| Delivery neutrality | The integration creates no issue, PRD, epic, feature package, implementation plan, PR, or delivery evidence merely to prove adoption. |
| Documentation integrity | `make docs-lint` exits `0`; Memory Bank navigation and links in the root documentation entrypoints are valid. |
| Automation | A pinned, repeatable local lint command exists. Hosted CI integration is optional and is not an adoption gate. |
| Scope boundary | The released `reviewer` binary does not require Memory Bank at runtime. Any prompt/finalization integration is separately scoped. |
| Adoption validation | A new agent can start from `AGENTS.md`, navigate to the correct project context, and run the local documentation checks without creating a task or delivery artifact. |

## Non-goals

- Replacing issue trackers, Git, repository hosting, change requests, CI, or `reviewer` itself.
- Copying project-specific details back to the generic upstream template.
- Making the template's Go linter part of the `reviewer` executable.
