---
title: Release And Deployment
doc_kind: ops
doc_function: canonical
purpose: Canonical record of the current reviewer release state and unresolved release decisions.
derived_from:
  - ../dna/governance.md
status: active
audience: humans_and_agents
---

# Release And Deployment

No official `reviewer` release or deployment flow exists yet. The repository contains the product specification and Memory Bank; the Go module, binary build, versioning policy, release artifacts, and distribution channels are not established. A push to the default branch or green change-request CI is not by itself an official product release.

## Release Flow

N/A until a release change defines and validates the process.

## Release Commands

None. The project must not advertise or automate release creation, package publication, or production deployment until commands and approval boundaries are designed and tested.

## Release Test Plan

No release test-plan format exists. Before the first official release, the owning change must define artifact identity, supported platforms, build reproducibility, acceptance/smoke checks, and evidence storage.

## Rollback

N/A because there is no deployed reviewer service or published release unit. Source changes remain recoverable through normal Git history, but that is not a substitute for a future release rollback policy.

## Unresolved Release Decisions

- Versioning and changelog policy.
- Minimum Go version and supported build targets.
- Artifact formats, signing/checksums, and distribution channel.
- Release approval owner, CI gates, and rollback unit.

Resolve these through an explicitly governed release/deployment change; do not infer them from generic Go or repository-hosting conventions.
