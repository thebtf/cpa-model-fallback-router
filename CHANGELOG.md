# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

## [0.1.3] - 2026-06-28

### Fixed

- Fall back when CPA reports `unknown provider for model ...` without a numeric HTTP status, which happens when a matching primary provider/account has been manually disabled or is no longer registered.
- Return an explicit `model-fallback-router` executor target from `model.route` instead of relying on host-side `self` target normalization.

### Verification

- `go test ./...`
- Local CPA `internal/pluginhost` acceptance with the rebuilt Windows/amd64 DLL.
- Docker Linux/amd64 CPA `internal/pluginhost` acceptance with the rebuilt Linux `.so`.
## [0.1.2] - 2026-06-28

### Added

- Add configurable primary-model cooldown through global `fallback.cooldown_seconds` and rule-level `rules[].cooldown_seconds` settings. The default is `60` seconds; `0` disables cooldown.
- Skip the primary model during an active cooldown and route matching requests directly to configured fallback models.

### Fixed

- Treat CPA auth-unavailable, model-cooldown, no-active-auth, and operator-disabled-account errors as fallback-eligible when CPA does not preserve a numeric HTTP status.
- Avoid duplicate fallback attempts when a fallback model resolves to the same model as the primary request.

### Notes

- Cooldown is scoped by source format, fallback rule, and primary model because CPA's pinned plugin SDK does not expose the selected auth account id to executor callbacks.

## [0.1.1] - 2026-06-27

### Changed

- Publish release artifacts in the official CPA plugin store layout: one zip archive per platform plus `checksums.txt`.
- Set plugin metadata author and repository to the public standalone repository.

## [0.1.0] - 2026-06-27

### Added

- Initial standalone release of the CPA model fallback router plugin.
- Transparent fallback from matching requested models to configured fallback model names.
- Redoc-rendered configuration reference.
