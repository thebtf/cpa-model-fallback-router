STACKS: [GO]

# AGENTS.md

## Project Overview

`cpa-model-fallback-router` is a native CLIProxyAPI plugin. It builds a c-shared library that CPA loads through the plugin ABI. The plugin retries matching model requests with configured fallback model names when the primary model fails with fallback-eligible status, quota, or transport errors.

## Commands

```bash
go test ./...
go build -buildmode=c-shared -o dist/model-fallback-router.so .
```

```powershell
.\scripts\build.ps1 -GOOS windows -GOARCH amd64
```

```bash
./scripts/build.sh
```

## Rules

- Keep the plugin ABI imports pinned through `go.mod`; do not vendor CPA source into this repository.
- Do not commit generated shared libraries (`.so`, `.dll`, `.dylib`) or generated C headers.
- Keep user-facing documentation in English unless a file is explicitly language-specific.
- Prefer small Go changes with focused tests for matching, fallback policy, config parsing, and host-call behavior.
- Run `gofmt -w .` after Go edits.
- Run `go test ./...` before commits.
- Release artifacts must be built by GitHub Actions from a tag.
- Do not write to production CPA config paths from this repository.

## Prioritization

1. Correct fallback behavior and CPA plugin ABI compatibility.
2. Safe install and rollback path for the user's Docker/Unraid CPA deployment.
3. Reproducible releases and checksums.
4. Documentation clarity.

## Instruction Hierarchy

System, developer, and user instructions override this file. Repo-local instructions override package-level habits. Keep `.agent/CONTINUITY.md` current during long sessions, but do not treat it as a replacement for live git/test evidence.

## Loaded Personas

No project-specific personas are required. Use the default senior Go maintainer posture: evidence-first, small changes, explicit verification.
