# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

## [0.2.0] - 2026-08-03

### Added

- Add an opt-in execution transform for agent-harness requests routed to non-native or same-model attempts, with tool-surface activation, configurable action-first instructions, required tool choice, privacy-safe telemetry, and an audited `ExitContinuationTool`.
- Add Claude, OpenAI Chat, and OpenAI Responses request transforms plus non-streaming audited-exit unwrapping for Claude and OpenAI Chat responses.
- Add a behavioral A/B emulator and a copyable multi-stage `kimi-* -> grok-4.5 -> gpt-5.6-luna` fallback example.

### Changed

- Keep cross-model request and response conversion inside CPA by carrying the mutated request protocol and downstream response protocol through the host executor.
- Activate execution transforms from the structural tool surface instead of prompt phrases, and bypass post-tool continuation turns.
- Document first-match-wins rule selection and ordered fallback behavior.

### Fixed

- Continue fallback on recognized context-window `400` errors while keeping other configured terminal `400` responses terminal.
- Preserve context-window fallback before the first stream payload even when CPA's stream bridge does not retain the numeric status.
- Keep same-model execution-transform rules usable during primary cooldown instead of producing an empty attempt plan.
- Unwrap an audited exit only when the plugin injected that attempt's exit tool, leaving operator-defined tools untouched.
- Include bounded upstream error response text in fallback classification when CPA preserves a status but not the provider error in the callback error.

### Verification

- `go test ./...`
- Focused three-attempt executor test for `kimi-* -> grok-4.5 -> gpt-5.6-luna` under two `429` responses.
- Native plugin package and CPA pluginhost smoke tests on the release candidate.

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
