# Continuity State

**Last Updated:** 2026-06-27
**Session:** Standalone project extraction, store-compatible release, and Redoc documentation pass for CPA model fallback router.

## Current State

The plugin is now a standalone public GitHub project.

- Local path: `D:\Dev\cpa-model-fallback-router`
- GitHub repo: `https://github.com/thebtf/cpa-model-fallback-router`
- Visibility: public
- Default branch: `main`
- Initial source commit: `28d691c Initial standalone CPA model fallback router`
- Store-compatible release commit: `573fe55 ci: publish CPA store compatible release assets`
- Latest release tag: `v0.1.1`
- Latest release URL: `https://github.com/thebtf/cpa-model-fallback-router/releases/tag/v0.1.1`
- CPA Plugins Store PR: `https://github.com/router-for-me/CLIProxyAPI-Plugins-Store/pull/10`

The source was extracted from the working CPA example plugin committed at `35b46f23 feat(plugin): add model fallback router example` in `D:\Dev\forks\CLIProxyAPI\.worktrees\fallback-router-plugin`.

The plugin had already been validated in the user's CPA deployment before extraction. The production bring-up artifact was `D:\tmp\model-fallback-router.so` with SHA256 `4CD200D40F40B0967E76D59C26795C9145C9741D4A37586E69057DEA7DBB10C5`.

## Release Evidence

`v0.1.1` is published as a normal GitHub release, not draft and not prerelease.

Release workflow evidence:

- Successful test workflow on `main`: `28298628773`
- Successful release workflow for `v0.1.1`: `28298674720`
- Remote tag `v0.1.1` exists on `origin`.

Published assets:

- `checksums.txt`
- `model-fallback-router_0.1.1_linux_amd64.zip`
- `model-fallback-router_0.1.1_linux_arm64.zip`
- `model-fallback-router_0.1.1_darwin_amd64.zip`
- `model-fallback-router_0.1.1_darwin_arm64.zip`
- `model-fallback-router_0.1.1_windows_amd64.zip`

Local and downloaded verification:

```powershell
go test ./...
.\scripts\package-release.ps1 -Version 0.1.1 -GOOS windows -GOARCH amd64
```

The local Windows zip contains `model-fallback-router.dll` at the archive root. The downloaded Linux amd64 release zip contains `model-fallback-router.so` at the archive root. Its SHA256 matched `checksums.txt`.

## Store Submission Evidence

Official store clone: `D:\Dev\CLIProxyAPI-Plugins-Store`.

The registry branch `add-model-fallback-router` adds `model-fallback-router` to `registry.json` with repository `https://github.com/thebtf/cpa-model-fallback-router`, MIT license, and fallback/router tags.

PR state after creation:

- PR: `https://github.com/router-for-me/CLIProxyAPI-Plugins-Store/pull/10`
- State: `OPEN`
- Mergeable: `MERGEABLE`
- Status checks: none configured at creation time

## Documentation Evidence

`nvmd-platform:docs --redoc` pass updated public docs after the store-compatible release:

- `README.md` now includes Quick Start, Features, Configuration, Commands, Architecture Overview, Troubleshooting, Releases, Documentation, and License.
- `CONTRIBUTING.md` records local development, release asset contract, documentation rules, and PR checklist.
- `docs/openapi.yaml` remains the Redoc source at version `0.1.1`.
- `.agent/reports/redoc-2026-06-27.md` records the redoc pass, source-backed claims, walkthrough, quality score, and evidence to refresh before future releases.

## Scope

This repository owns:

- Standalone Go module for the CPA native plugin.
- Onboarding files: `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.agent/*`.
- Redoc docs under `docs/`.
- GitHub Actions release automation for CPA Plugins Store compatible assets.

## Current Limitation

CPA's plugin host callback currently does not expose selected auth/provider metadata. The plugin can route globally or by source format/requested model, but cannot yet route only for a selected auth kind such as Anthropic OAuth.

## Next

No required next step. Optional future improvements:

1. Add Windows arm64 when a reliable cgo compiler path is confirmed.
2. Add provider/auth-kind scoping after CPA exposes selected-provider metadata to plugins.
3. Watch CPA Plugins Store PR #10 until review/merge.

## Blockers

None.

## Resumability Test

```powershell
cd D:\Dev\cpa-model-fallback-router
git status --short --branch
git log --oneline -5
gh release view v0.1.1 --json tagName,url,isDraft,isPrerelease,assets
gh pr view 10 --repo router-for-me/CLIProxyAPI-Plugins-Store --json number,title,url,state,mergeable,reviewDecision,statusCheckRollup
```

Expected: `main` tracks `origin/main`; only ignored `.serena/` and `dist/` may appear with `--ignored`; release `v0.1.1` exists with five platform zips plus `checksums.txt`; store PR #10 remains open or has been merged by maintainers.
