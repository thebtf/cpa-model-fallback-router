# Continuity State

**Last Updated:** 2026-06-28
**Session:** Disabled-provider fallback fix, PR review, v0.1.3 release, and runtime confirmation.

## Active Goal Contract (verbatim)

Status: none. The release/debug objective is complete.

## Current State

The standalone CPA native plugin `model-fallback-router` is shipped as `v0.1.3` and the user confirmed the updated plugin was installed and works in the CPA runtime.

- Local path: `D:\Dev\cpa-model-fallback-router`
- GitHub repo: `https://github.com/thebtf/cpa-model-fallback-router`
- Branch: `main`
- Release/tag target: `48817ee3229b1d9e75bb549c866fd5ed9cefbea5` (`v0.1.3`)
- HEAD subject: `fix: fall back on disabled provider errors`
- Latest release tag: `v0.1.3`
- Latest release URL: `https://github.com/thebtf/cpa-model-fallback-router/releases/tag/v0.1.3`
- PR: `https://github.com/thebtf/cpa-model-fallback-router/pull/2` merged via squash at `2026-06-28T01:18:45Z`
- Working tree at save time: clean. After the local session-save checkpoint commit, `main` may be ahead of `origin/main` by 1 docs-only commit unless that checkpoint has been pushed.

## Done

- Diagnosed the live failure from CPA logs and SDK/pluginhost behavior: when the primary Claude provider/account is manually disabled, CPA can return `unknown provider for model ...` to native plugins without a numeric HTTP status.
- Patched fallback classification so no-status model/provider-unavailable errors are fallback-eligible.
- Patched `model.route` to return an explicit `model-fallback-router` executor target instead of relying on host-side `self` target normalization.
- Added regression coverage for explicit executor routing and no-status `unknown provider` fallback classification.
- Built and tested the native plugin locally before release:
  - `go test ./...`
  - Windows/amd64 package build through `scripts/package-release.ps1`
  - CPA `internal/pluginhost` acceptance against the packaged Windows DLL, registered as `0.1.3`
  - Docker Linux/amd64 CPA `internal/pluginhost` acceptance against the rebuilt `.so`, registered as `0.1.3`
- Opened PR #2, invoked MCP PR review, fixed the Gemini review note about the redundant classifier token, resolved the thread, and merged after checks/review were green.
- Published annotated tag `v0.1.3` and verified remote tag parity.
- GitHub Actions release workflow `28307464007` completed successfully for linux/darwin/windows builds plus publish.
- Verified the published release body is specific to the disabled-provider fallback fix, not the old generic artifact-layout text.
- Verified published assets include five platform zips plus `checksums.txt`; downloaded `checksums.txt` is in sha256sum format.
- User reported after updating and checking CPA runtime: the new plugin works.

## Evidence Ledger

- PR #2: `MERGED`, review decision `APPROVED`, merge commit `48817ee3229b1d9e75bb549c866fd5ed9cefbea5`.
- Main CI on merge commit: workflow `test`, run `28307407472`, success.
- Release workflow: run `28307464007`, success, tag branch `v0.1.3`, head SHA `48817ee3229b1d9e75bb549c866fd5ed9cefbea5`.
- Remote tag evidence:
  - annotated tag object `e04f8dfbf6f7465a2dd6f9a0d9c49d206cad75d8 refs/tags/v0.1.3`
  - peeled commit `48817ee3229b1d9e75bb549c866fd5ed9cefbea5 refs/tags/v0.1.3^{}`
- Release assets:
  - `checksums.txt`
  - `model-fallback-router_0.1.3_linux_amd64.zip`
  - `model-fallback-router_0.1.3_linux_arm64.zip`
  - `model-fallback-router_0.1.3_darwin_amd64.zip`
  - `model-fallback-router_0.1.3_darwin_arm64.zip`
  - `model-fallback-router_0.1.3_windows_amd64.zip`
- Published `checksums.txt` lines:
  - `7e5f19a2babe53d4adcb1952f67d165c24ba91dee59091bce100fb5e4ee49863  model-fallback-router_0.1.3_darwin_amd64.zip`
  - `a2c480bf37b85f9dc8272bae39fe1343ae066b341c082550ba5f6c252c456192  model-fallback-router_0.1.3_darwin_arm64.zip`
  - `b2ddf07ebf0f9b39631ba021ec370ff62c735285cd1f5137cac5d5feeead1aab  model-fallback-router_0.1.3_linux_amd64.zip`
  - `ea22b06ce79bd4fa9c84a0bbe6d9b1828e8a56815d9fba18af695f7c3f01a111  model-fallback-router_0.1.3_linux_arm64.zip`
  - `c54a006b3dc6a2c6e8a9b6734f30a4fb88f92097c8a241e2e8b4ea2417ea8415  model-fallback-router_0.1.3_windows_amd64.zip`

## Now

No active implementation or release task is in progress. The repository is clean and release `v0.1.3` is shipped.

## Next

No required next step.

Optional follow-ups only:

1. Watch CPA Plugins Store PR #10 until upstream review/merge, if store listing is still desired.
2. Add provider/auth-kind scoped fallback only after CPA exposes selected provider/auth metadata to plugin executors.
3. Consider adding Windows arm64 release assets when a reliable cgo compiler path is confirmed.

## Blockers

No blockers for the shipped `v0.1.3` fix.

External/optional blockers:

- CPA Plugins Store PR #10 depends on upstream maintainer review/merge.
- Provider/auth-kind scoped fallback depends on CPA exposing selected provider/auth metadata to plugin executors.

## Mutation Boundary

Do not write to production CPA paths such as `A:\cliproxy` from this repository. Runtime update was performed and verified by the user, not by repo automation.

## Resumability Test

A future agent should start by reading `.agent/session-state/current.json`, then this file, then run `git status --short --branch` and `git show --no-patch --format='%H %D%n%s' HEAD`. Expected release state: tag `v0.1.3` points to `48817ee3229b1d9e75bb549c866fd5ed9cefbea5`. Local `main` may either equal `origin/main` at that release commit or be ahead by one docs-only session-save commit. If asked to continue, verify `gh release view v0.1.3` and only pursue optional follow-ups explicitly requested by the user; do not recreate, retag, or redeploy `v0.1.3` without new evidence of a release defect.
