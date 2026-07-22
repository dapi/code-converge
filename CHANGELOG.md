# Changelog

All notable changes to this project are documented here.

This project follows Semantic Versioning.

## [Unreleased]

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
