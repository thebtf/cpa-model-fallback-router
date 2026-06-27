# Contributing

Thanks for improving CPA Model Fallback Router. Keep changes small, tested, and compatible with the CPA native plugin ABI.

## Development Setup

Requirements:

- Go 1.26 or newer.
- A platform C compiler that can build cgo shared libraries.
- PowerShell 7 or Windows PowerShell for `scripts/package-release.ps1`.

Run the main verification command before opening a PR:

```bash
go test ./...
```

For release packaging changes, also run at least one local package smoke:

```powershell
.\scripts\package-release.ps1 -Version 0.1.1 -GOOS windows -GOARCH amd64
```

## Release Asset Contract

The CPA Plugins Store installer expects release assets in this shape:

- Release tag: `v<version>`, for example `v0.1.1`.
- One `checksums.txt` asset.
- One zip per supported target named `<id>_<version>_<goos>_<goarch>.zip`.
- Each zip contains exactly one root-level dynamic library named `model-fallback-router.so`, `model-fallback-router.dylib`, or `model-fallback-router.dll`.

Do not rename release artifacts without checking the official store installer contract first.

## Documentation Changes

Update `README.md` when behavior, installation, configuration, or release packaging changes. Update `docs/openapi.yaml` when configuration schema or examples change. Open `docs/index.html` locally to inspect the Redoc rendering.

## Pull Request Checklist

- Behavior changes have focused tests.
- `go test ./...` passes.
- Public docs match the behavior being shipped.
- Release packaging changes include a local package smoke.
- No secrets, tokens, auth files, or local production config paths are committed.
