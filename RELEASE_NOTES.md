# v0.1.3 - Disabled-provider fallback fix

This patch fixes the path seen when a matching primary provider or account is manually disabled in CLIProxyAPI. CPA can report `unknown provider for model ...` to native plugins without preserving a numeric HTTP status; previous builds stopped after the primary attempt. v0.1.3 treats that host callback error as fallback-eligible, so configured fallback models are tried normally.

## What's Changed

- Fallback now continues when CPA returns `unknown provider for model ...` with no HTTP status.
- `model.route` now returns an explicit `model-fallback-router` executor target instead of relying on host-side `self` target normalization.
- Added regression coverage for the no-status `unknown provider` error text.
- Added local CPA pluginhost acceptance coverage with the real native plugin binary on Windows and Linux.

## Validation

- `go test ./...`
- `scripts/build.ps1 -GOOS windows -GOARCH amd64`
- CPA `internal/pluginhost` acceptance against the rebuilt Windows/amd64 DLL.
- Docker Linux/amd64 CPA `internal/pluginhost` acceptance against the rebuilt Linux `.so`.

## SDK Compatibility Notes

The route/executor behavior was checked against the local CPA v7.2.43 pluginhost SDK path. The plugin still delegates all model execution to CPA through `host.model.execute`; no direct provider calls or credential handling were added.

## Assets

The release keeps the CPA plugin store artifact layout: one zip per supported platform plus `checksums.txt`. Each zip contains exactly one root-level dynamic library named for the platform.
