# Continuity State

**Last Updated:** 2026-06-27
**Session:** Standalone project extraction for CPA model fallback router.

## Current State

The project was extracted from the working CPA example plugin committed at `35b46f23 feat(plugin): add model fallback router example` in `D:\Dev\forks\CLIProxyAPI\.worktrees\fallback-router-plugin`.

The plugin was already validated in the user's CPA deployment before extraction. The production bring-up artifact was `D:\tmp\model-fallback-router.so` with SHA256 `4CD200D40F40B0967E76D59C26795C9145C9741D4A37586E69057DEA7DBB10C5`.

## Scope

Create and maintain a standalone GitHub project for the plugin:

- Go module at the repository root.
- Onboarding files: `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.agent/*`.
- Redoc docs under `docs/`.
- GitHub Actions release automation for platform-specific shared libraries.
- Initial GitHub release.

## Current Limitation

CPA's plugin host callback currently does not expose selected auth/provider metadata. The plugin can route globally or by source format/requested model, but cannot yet route only for a selected auth kind such as Anthropic OAuth.

## Next

1. Run `go mod tidy`.
2. Run `go test ./...`.
3. Build the local Windows shared library once.
4. Initialize git, commit, push to GitHub, tag `v0.1.0`, and verify the release workflow.

## Blockers

None yet. If GitHub repository creation fails, check `gh auth status` and create/push manually under the intended owner.

## Resumability Test

```powershell
cd D:\Dev\cpa-model-fallback-router
git status --short --branch
go test ./...
.\scripts\build.ps1 -GOOS windows -GOARCH amd64
```

Expected before release: source tree contains no committed `.so`, `.dll`, `.dylib`, or `.h` artifacts; tests pass; local build writes artifacts under `dist/`.
