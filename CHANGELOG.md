# Changelog

All notable changes to this project are documented here.

This project follows Semantic Versioning.

## [Unreleased]

## [0.4.0] - 2026-07-23

### Added

- Added conventional `-h` and `--help` aliases for the root command.
- Added strict structured Codex review-result handling through `codex exec --output-schema`, removing dependence on terminal streams for review classification.

### Changed

- Review integration now fails closed when the required structured-result capabilities are unavailable or the response is malformed.

## [0.3.1] - 2026-07-22

### Changed

- Enriched human progress with local timestamps, retry-attempt context, resolved model/reasoning effort, and clearer review, fix, CI-recovery, and liveness status lines.
- In interactive terminals, the live stage indicator now replaces the duplicate permanent stage-start line.

## [0.3.0] - 2026-07-22

### Added

- Added an explicit human workflow log format with readable findings, durations, finalization steps, terminal outcomes, interactive elapsed-time shimmer, and optional newline heartbeat while preserving `kv` as the default.
- Added `code-converge update` with checksum verification, interactive confirmation and an unattended `--yes`/`-y` path for safely replacing the installed binary from a compatible GitHub Release.
- Added strict support for structured Codex review responses and successful no-change completion without attempting finalization or an empty commit.
- Expanded review scope to the intended pull-request base through the current worktree, including committed, staged, unstaged and untracked changes.

### Changed

- Made the human-readable workflow log format the built-in default; use `--log-format=kv` for the stable machine-readable event stream.

### Fixed

- Updated pinned GitHub Actions to Node 24-based releases and keyed Go caches from `go.mod`, eliminating release and verification workflow warnings.

## [0.2.0] - 2026-07-22

### Added

- Added `fast` and `best` model profiles with per-stage model and reasoning-effort resolution and explicit override support.
- Added effective model and reasoning-effort fields to stage progress records.

### Changed

- Renamed the CLI, configuration namespace, module, installer, archives, and repository identity from `reviewer` to `code-converge`; this is a clean break and old names are not supported.

## [0.1.0] - 2026-07-21

### Added

- Added tag-triggered GitHub Release automation for the supported binary matrix.
- Added `code-converge --version` output with the embedded release version.
- Added checksum-verifying one-line installation for macOS and Linux on AMD64/ARM64.
