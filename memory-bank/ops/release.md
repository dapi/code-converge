---
title: Release And Deployment
doc_kind: ops
doc_function: canonical
purpose: Canonical code-converge release contract covering preparation, GitHub publication, verification, approval and rollback.
derived_from:
  - ../dna/governance.md
  - ../../README.md
status: active
audience: humans_and_agents
---

# Release And Deployment

`code-converge` is distributed as versioned archives through GitHub Releases. There is no server deployment. A push to the default branch, green change-request CI, or local `dist/` directory is not by itself an official product release; publication requires an approved semantic-version tag.

## Release Flow

1. Add user-facing changes under `## [Unreleased]` in `CHANGELOG.md`.
2. Run `make release-patch`, `make release-minor`, or `make release-major` from a clean worktree.
3. The command updates `VERSION` and the changelog, runs `make verify` and `make dist`, creates `Release vX.Y.Z`, and creates an annotated tag.
4. Publish the prepared commit and tag with `git push origin master --follow-tags` after operator approval.
5. GitHub Actions verifies tag identity, reruns all checks, builds and validates artifacts, smoke-tests Linux AMD64, then creates the GitHub Release.

The supported build matrix is macOS and Linux on AMD64 and ARM64. Each platform archive is a normalized `tar.gz` containing one `code-converge` binary; the release also contains aggregate `SHA256SUMS`. Installation remains manual checksum verification, extraction, and copy to an operator-owned directory on `PATH`.

## Release Commands

```sh
VERSION=<version> make dist
make print-version
make release-patch
make release-minor
make release-major
```

`make dist` produces local artifacts only. The `release-*` commands prepare a local commit and tag but do not push them. A tag push is an explicit external action requiring operator approval and triggers automated GitHub publication. Signing and package-manager publication are not supported.

## Release Test Plan

- Run `make verify`.
- Run `make dist` twice from the same revision and compare `SHA256SUMS`.
- Validate every checksum from inside `dist/`.
- Extract and run `code-converge config` from a native target archive; CI performs Linux AMD64 smoke.
- Retain the commit SHA, checksum manifest, required-CI run, and smoke result as release evidence.
- Confirm the GitHub Release contains exactly four archives plus `SHA256SUMS` and that its tag matches `VERSION`.

## Rollback

The CLI has no persistent state or migration. Backout replaces or removes the installed binary and restores the prior verified release. Publish a fixed patch release instead of mutating existing versioned assets. If urgent withdrawal is necessary, removing or marking the GitHub Release requires separate operator approval. Local `dist/` is disposable and can be rebuilt from the source revision.

## Unresolved Release Decisions

- Signing policy and any package-manager channel.
- Named approval owner beyond the repository operator who pushes the release tag.

These decisions do not block GitHub Release publication under the explicit tag-push approval gate.
