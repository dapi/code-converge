---
title: Stages And Non-Local Environments
doc_kind: ops
doc_function: canonical
purpose: Canonical record that reviewer has no hosted runtime stages and defines the boundary with local execution and external CI.
derived_from:
  - ../dna/governance.md
status: active
audience: humans_and_agents
---

# Stages And Non-Local Environments

`reviewer` is specified as a local CLI. It has no production, staging, beta, preview, sandbox, hosted control plane, service endpoint, or application database. A Git remote, repository hosting, and CI belong to the target repository and are external systems, not deployment stages of `reviewer`.

## Environment Inventory

| Environment | Purpose | Access path | Notes |
| --- | --- | --- | --- |
| Local operator machine | Run the CLI against a selected Git repository | Shell in the target repository | Requires local agent authentication and Git/hosting access for enabled actions |
| Non-local reviewer runtime | N/A | N/A | No hosted runtime exists |

## Common Operations

There are no remote console, cluster, database, or hosted-log operations. Local development commands are in [`development.md`](development.md). Mutating Git/hosting actions use the operator's existing credentials and must be limited to the authorized workflow stage.

## Credentials And Access

- `reviewer` credentials are not provisioned by a hosted environment.
- Agent authentication and Git/hosting credentials are operator prerequisites described in [`config.md`](config.md).
- The repository must not store tokens, private keys, or copied credentials, and operational output must not expose them.

## Version And Health Checks

N/A. There is no deployed version, health endpoint, smoke URL, or application dashboard. Future released binaries require a version-reporting contract defined by the release owner.

## Logs And Observability

Runtime progress is emitted to the invoking process's stdout according to the CLI contract. No central log store, metrics backend, trace system, error tracker, or dashboard is defined.

## Test Data And Smoke Targets

N/A. No staging tenant, seed user, or project-managed test account exists. Deterministic implementation tests must use fakes rather than live agent, repository-hosting, or CI systems.

## Adoption Checklist

- [x] Non-local reviewer environments are explicitly recorded as absent.
- [x] Local execution and external CI are distinguished from application stages.
- [x] Health/version checks are explicitly N/A until a release contract exists.
- [x] Current stdout observability and the absence of hosted backends are recorded.
- [x] Generic production/staging commands and credentials have been removed.
