# Configurable document review prompts

`code-converge` reviews code by default. Since v1.1.0 the review stage also supports
explicit, deterministic document review and custom review prompts.

## What is it for

Teams that keep specifications, plans, and knowledge bases (including the Memory Bank)
in Markdown next to the code can now run the same automated review-and-fix loop on
those documents: consistency checks, contradiction detection, unresolved material
questions — reviewed by Codex with the same strict findings schema as code review.

## Review prompt selection

At most one selector may be given; conflicting selections exit `2` with no fallback:

| Selector | Source |
| --- | --- |
| *(none)* | Built-in code-review prompt (unchanged default) |
| `--review-prompt-file <path>` | Any readable Markdown file (absolute or relative to the current directory) |
| `--review-prompt <name>` | Only `.code-converge/<name>.md`; names may contain letters, digits, `_`, `-` |
| `--document-review` | `.code-converge/default.md` if it exists, otherwise the built-in document prompt |

Named prompts live in `.code-converge/`, so they are versioned and reviewed together
with the project. Missing, unreadable, non-Markdown, or invalid selections fail
predictably with exit code `2`.

## Document review mode

`--document-review` reviews only changed `.md` files in the merge-base-to-worktree
snapshot, excluding `memory-bank/prompts/**`. If no eligible documents changed, the
run completes cleanly without invoking Codex. Document mode is review-only: after a
clean scoped review it exits successfully without publishing or waiting for CI.

Bootstrap a project template with:

```sh
code-converge init-document-review-prompt         # writes .code-converge/default.md
code-converge init-document-review-prompt --force # overwrite an existing file
```

Findings in document mode are fixed with the built-in document-fix instruction, or
with `--document-fix-prompt-file <path>` (requires `--document-review`, conflicts
with `--fix-prompt-file`).

## Examples

```sh
code-converge --document-review
code-converge --review-prompt security-audit
code-converge --review-prompt-file ./prompts/api-review.md
code-converge --document-review --document-fix-prompt-file ./prompts/doc-fix.md
```

See the [root README](../README.md#document-review-prompts) for the full CLI and
configuration contract.
