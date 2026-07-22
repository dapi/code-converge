---
title: Memory Bank Upstream Record
doc_kind: project
doc_function: reference
purpose: Records the immutable upstream template revision and the downstream update policy.
derived_from:
  - dna/governance.md
status: active
audience: humans_and_agents
---

# Memory Bank Upstream Record

## Imported template

- Upstream: <https://github.com/dapi/memory-bank>
- Immutable source revision: `2e7324181af3034d9e22a411eb977b6729fae2b8`
- Imported: 2026-07-21
- Import method: local snapshot of upstream `memory-bank/` directory, followed by downstream adaptation. This is not a byte-for-byte mirror; inapplicable project placeholders may be removed.

## Downstream ownership

This copy is the project-specific Memory Bank for `code-converge`. Its `product/`, `domain/`, `engineering/`, `ops/`, `adr/`, `features/`, and lifecycle records are owned by this repository and evolve with it.

## Update policy

Do not overwrite this directory wholesale from upstream. For every upstream update:

1. Compare the new template with the recorded revision.
2. Select only generic governance or template improvements that remain compatible with `code-converge`'s project-specific documents.
3. Preserve downstream facts and records.
4. Update this file with the imported revision and run `make memory-bank-lint` from the repository root.

## Rights note

The copy into `code-converge` is authorized by the owner of the `dapi/memory-bank` and `dapi/code-converge` repositories. This has nothing to do with GitHub specifically: copyright and reuse permissions apply the same way on any hosting service or local filesystem. The imported source revision contains no public `LICENSE` file, so this record does not grant third parties a general right to reuse the snapshot; add an upstream license only if such a grant is intended.
