# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added

- Add primary-model cooldown with `cooldown_seconds` so fallback-eligible primary failures can temporarily route later requests directly to fallback models.
- Treat CPA auth-unavailable, model-cooldown, and operator-disabled-account errors as fallback-eligible when no HTTP status is available.

## [0.1.1] - 2026-06-27

### Changed

- Publish release artifacts in the official CPA plugin store layout: one zip archive per platform plus `checksums.txt`.
- Set plugin metadata author and repository to the public standalone repository.

## [0.1.0] - 2026-06-27

### Added

- Initial standalone release of the CPA model fallback router plugin.
- Transparent fallback from matching requested models to configured fallback model names.
- Redoc-rendered configuration reference.
