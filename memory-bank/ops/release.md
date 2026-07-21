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

No official hosted `reviewer` release has been published yet. The repository can produce deterministic installable artifacts, but a push to the default branch, green change-request CI, or local `dist/` directory is not by itself an official product release.

## Release Flow

The supported build matrix is macOS and Linux on AMD64 and ARM64. Each release unit is a normalized `tar.gz` containing one `reviewer` binary plus the aggregate `SHA256SUMS`. The current distribution path is manual checksum verification, extraction, and copy to an operator-owned directory on `PATH`.

## Release Commands

```sh
VERSION=<version> make dist
```

This produces local artifacts only. Creating a GitHub Release, signing, or publishing to a package manager remains an explicit external action requiring its own approval.

## Release Test Plan

- Run `make verify`.
- Run `make dist` twice from the same revision and compare `SHA256SUMS`.
- Validate every checksum from inside `dist/`.
- Extract and run `reviewer config` from a native target archive; CI performs Linux AMD64 smoke.
- Retain the commit SHA, checksum manifest, required-CI run, and smoke result as release evidence.

## Rollback

The CLI has no persistent state or migration. Backout replaces or removes the installed binary and restores the prior verified artifact. Local `dist/` is disposable and can be rebuilt from the source revision.

## Unresolved Release Decisions

- Versioning and changelog policy beyond the caller-supplied artifact version.
- Signing policy and any package-manager or hosted release channel.
- Official release approval owner.

These unresolved official-publication decisions do not block deterministic local artifacts. Resolve them before claiming or publishing an official release.
