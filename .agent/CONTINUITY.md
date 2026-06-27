# Continuity State

**Last Updated:** 2026-06-27
**Session:** Standalone project extraction and initial release for CPA model fallback router.

## Current State

The plugin is now a standalone GitHub project.

- Local path: `D:\Dev\cpa-model-fallback-router`
- GitHub repo: `https://github.com/thebtf/cpa-model-fallback-router`
- Visibility: public
- Default branch: `main`
- Initial source commit: `28d691c Initial standalone CPA model fallback router`
- Workflow fix commit: `6076b7d ci: use available macOS runner for darwin amd64 release`
- Release tag: `v0.1.0`
- Release URL: `https://github.com/thebtf/cpa-model-fallback-router/releases/tag/v0.1.0`

The source was extracted from the working CPA example plugin committed at `35b46f23 feat(plugin): add model fallback router example` in `D:\Dev\forks\CLIProxyAPI\.worktrees\fallback-router-plugin`.

The plugin had already been validated in the user's CPA deployment before extraction. The production bring-up artifact was `D:\tmp\model-fallback-router.so` with SHA256 `4CD200D40F40B0967E76D59C26795C9145C9741D4A37586E69057DEA7DBB10C5`.

## Release Evidence

`v0.1.0` is published as a normal GitHub release, not draft and not prerelease.

Release workflow evidence:

- Successful release workflow run: `28296570065`
- Successful test workflow after workflow fix: `28296552249`
- Initial tag-push release workflow run `28296356615` was cancelled because `macos-13` queued indefinitely for `darwin-amd64`.
- Fix: `darwin-amd64` now runs on `macos-14`; manual release dispatch for existing tag `v0.1.0` succeeded without rewriting the tag.

Published assets:

- `model-fallback-router-linux-amd64.so`
- `model-fallback-router-linux-arm64.so`
- `model-fallback-router-darwin-amd64.dylib`
- `model-fallback-router-darwin-arm64.dylib`
- `model-fallback-router-windows-amd64.dll`
- `SHA256SUMS.txt`

Local verification before publishing:

```powershell
go mod tidy
gofmt -w .
.\scripts\build.ps1 -GOOS windows -GOARCH amd64
```

This ran `go test ./...` and built `dist/model-fallback-router-windows-amd64.dll` locally.

## Scope

This repository owns:

- Standalone Go module for the CPA native plugin.
- Onboarding files: `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.agent/*`.
- Redoc docs under `docs/`.
- GitHub Actions release automation for platform-specific shared libraries.

## Current Limitation

CPA's plugin host callback currently does not expose selected auth/provider metadata. The plugin can route globally or by source format/requested model, but cannot yet route only for a selected auth kind such as Anthropic OAuth.

## Next

No required next step. Optional future improvements:

1. Add Windows arm64 when a reliable cgo compiler path is confirmed.
2. Add provider/auth-kind scoping after CPA exposes selected-provider metadata to plugins.
3. Keep public-facing docs and release notes accurate now that the repository is public.

## Blockers

None.

## Resumability Test

```powershell
cd D:\Dev\cpa-model-fallback-router
git status --short --branch
git log --oneline -3
gh release view v0.1.0 --json tagName,url,isDraft,isPrerelease,assets
```

Expected: `main` tracks `origin/main`; only ignored `.serena/` and `dist/` may appear with `--ignored`; release `v0.1.0` exists with five platform assets plus `SHA256SUMS.txt`.
