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

## 2026-06-29 Addendum — Execution Transform Implementation

A new implementation task is complete but not released or committed in this session. The user explicitly said not to release without a separate command.

Current uncommitted work adds an opt-in `execution_transform` for Claude Code harness turns routed to GPT fallback attempts:

- Disabled by default globally.
- Rule-level config can enable it narrowly for Claude -> GPT fallback rules.
- Request transform can inject an execution envelope, add `ExitContinuationTool`, and set required tool-choice for supported Claude/OpenAI JSON shapes.
- Non-streaming responses that consist solely of `ExitContinuationTool` are unwrapped to normal assistant text; streaming response unwrap is intentionally not implemented.
- Metadata reports transform/audited-exit decisions without raw prompts, raw bodies, or full tool schemas.
- Host execution still delegates only through CPA `host.model.execute` / stream callbacks; no direct upstream calls or CPA ABI changes.

Verification performed:

- Built local Windows/amd64 shared plugin through `.\scripts\build.ps1 -GOOS windows -GOARCH amd64`; output was `dist/model-fallback-router-windows-amd64.dll`.
- Local CPA pluginhost proof was performed without production deployment: copied CPA SDK `v7.2.31` to `D:\tmp\cpa-pluginhost-smoke-execution-transform`, copied the rebuilt DLL as `_plugins\model-fallback-router.dll`, and ran `go test ./internal/pluginhost -run TestExternalModelFallbackRouter -count=1 -v`.
- The CPA smoke passed two tests:
  - `TestExternalModelFallbackRouterExecutionTransformSmoke`: CPA loaded the real DLL, route target was `model-fallback-router`, primary `claude-sonnet-4-5` failed with auth-unavailable, fallback `gpt-5.5` succeeded, only the fallback request was transformed, `Execution mode is active`/required tool choice/`ExitContinuationTool` were present, and metadata had `execution_transform` plus `fallback_used=true` without raw body leakage.
  - `TestExternalModelFallbackRouterAuditedExitSmoke`: a fallback response containing only `ExitContinuationTool` was unwrapped to normal Claude text and emitted `audited_exit.unwrapped=true`.
- `go test ./...` passed.
- `git diff --check` exited 0, with Windows CRLF normalization warnings only.
- TDD Prove-It: a temporary detection-bypass stub made the new transform tests fail, then the stub was restored.


Behavioral proof gap remains: current local smoke proves load/route/transform/unwrap mechanics, but not live GPT agent behavior. The task being solved is GPT behaving as an action-oriented Claude Code fallback executor, not merely receiving modified JSON. At this checkpoint the shell has no OPENAI_API_KEY, ANTHROPIC_API_KEY, CPA_BASE_URL, or CLIPROXY_API_KEY, so a real model-in-the-loop proof needs a non-prod model/CPA endpoint or explicit runtime authorization.

Behavioral smoke runner added: `scripts/smoke-behavioral-cpa.ps1`. It posts a Claude Code harness turn to a non-production CPA `/v1/messages` endpoint and classifies the response. PASS requires a real tool call or valid `ExitContinuationTool` audited exit plus verified/accepted fallback-path evidence; passive text is FAIL. Current evidence:

- `.agent/cpa-model-fallback-router-execution-middleware-tz/evidence/behavioral-smoke.cpa.json`: `BLOCKED_BY_ENDPOINT` because `CPA_BASE_URL` is not set.
- `.agent/cpa-model-fallback-router-execution-middleware-tz/evidence/behavioral-smoke.error-path.json`: `ERROR` from a loopback timeout, proving the runner writes error evidence without raw prompt/body leakage.

## 2026-06-29 Addendum — Local Docker Behavioral Proof

Behavioral proof gap is closed for the local non-production Docker path. The user authorized use of a copied Codex OAuth account from `A:\cliproxy\auth-dir`; the file was copied only into `D:\tmp\cpa-local-behavioral\auths`. Production config/runtime paths under `A:\cliproxy` were not modified.

Verified local Docker runtime:

- Pulled and ran vanilla CPA image `eceasy/cli-proxy-api:latest` (CPA reported `v7.2.47`, commit `00114be`, built `2026-06-29T11:00:58Z`). The earlier `ghcr.io/thebtf/cliproxyapi:latest` attempt was discarded as the wrong frozen fork image.
- Mounted isolated config/auth/plugin copies under `D:\tmp\cpa-local-behavioral` and exposed CPA only on `127.0.0.1:18317`.
- CPA logs show native plugin load/register for `model-fallback-router` from `plugins/model-fallback-router.so`.
- `/v1/models` returned Codex models including `gpt-5.5`.
- `scripts/smoke-behavioral-cpa.ps1 -BaseUrl http://127.0.0.1:18317 -ApiKey local-behavioral-smoke -AcceptOperatorConfiguredFallback -EvidencePath .agent\cpa-model-fallback-router-execution-middleware-tz\evidence\behavioral-smoke.local-docker-cpa.json -TimeoutSeconds 180` returned `PASS: real_tool_call; fallback path OPERATOR_ASSERTED_NON_PROD_FALLBACK`.
- CPA logs for the passing request id show `Use OAuth provider=codex ... for model gpt-5.5` followed by `200` on `POST /v1/messages`.
- Evidence files:
  - `.agent/cpa-model-fallback-router-execution-middleware-tz/evidence/behavioral-smoke.local-docker-cpa.json`
  - `.agent/cpa-model-fallback-router-execution-middleware-tz/evidence/behavioral-smoke.local-docker-cpa.logs.txt`

Final local verification after the Docker proof: `go test ./...` passed.

Cleanup after proof: the local container `cpa-mfr-behavioral` was stopped/removed, and the temporary copied OAuth JSON under `D:\tmp\cpa-local-behavioral\auths` was deleted. The original authorized auth file under `A:\cliproxy\auth-dir` was not modified.

## 2026-06-29 Addendum — Benefit A/B Emulator Proof

Product-value proof was added after the local Docker feasibility proof. The new regression `TestExecutionTransformSolvesPassiveGPTFallbackEmulator` emulates the original non-native harness failure mode deterministically:

- Baseline config with `execution_transform` disabled: primary Claude fails with `auth_unavailable`, fallback GPT emulator receives an untransformed request, and returns passive text: `I would inspect the saved session state...`. No `execution_transform` metadata is present.
- Fixed config with `execution_transform` enabled: the same fallback path receives execution envelope + `ExitContinuationTool` + required tool choice, and the emulator returns a real `Read` tool call. Metadata has `execution_transform.applied=true` and `forced_tools=true`.

Verification:

- `go test ./... -run TestExecutionTransformSolvesPassiveGPTFallbackEmulator -count=1 -v` passed.
- `go test ./...` passed.
- Evidence: `.agent/cpa-model-fallback-router-execution-middleware-tz/evidence/benefit-emulator.ab-test.txt`.
- Report: `.agent/reports/execution-transform-benefit-proof-2026-06-29.md`.

Interpretation: this proves benefit, not only feasibility. The middleware changes a reproduced passive-text fallback into an agentic tool call under the same fallback chain shape. No release/tag/publish/prod mutation was performed.

## 2026-06-29 Addendum — Harness Marker Reconfiguration

The execution transform detection was adjusted after reviewing the local Claude Code prompt corpus under `D:\Dev\_EXTRAS_\system_prompts_leaks\Anthropic\Claude Code`. The baseline detector now uses configurable `execution_transform.harness_markers` instead of relying on site-local nvmd trigger phrases.

Current behavior:

- `harness_markers` is available at global and rule-level `execution_transform` config.
- Plain markers match request text case-insensitively; `tool:<name>` markers match available request tool names.
- Default markers come from stable Claude Code harness surface: Claude Code identity, interactive software-engineering agent wording, terminal markdown, permission-mode/system-reminder hints, dedicated file/search tool guidance, parallel tool calls, `CLAUDE.md` / `.claude` / slash command markers, and common tool names.
- `execution_skill_names` and `trigger_phrases` remain supported as optional local hints, but the tests no longer require `CHECK_MY_STATUS_READY` or hardcoded nvmd command names for the base transform.
- CPA plugin metadata now exposes flat GUI-visible `execution_transform.*` fields supported by `pluginapi.ConfigField`, including `execution_transform.harness_markers` and `execution_transform.force_tools` enum values.

Verification after this adjustment:

- `go test ./...` passed.
- The A/B emulator path now proves transform behavior without `trigger_phrases`; the transform reason must start with `harness_`.
- Rebuilt Linux amd64 artifact via Docker `golang:1.26-bookworm`.
- Vanilla CPA Docker image `eceasy/cli-proxy-api:latest` / CPA `v7.2.47` loaded and registered the rebuilt plugin with a config using `harness_markers` and no `trigger_phrases`; local `/` probe returned HTTP 200.
- Artifact SHA256: `6692A2DB077AFB07F809373E3E065FF071C31965E2F9527799F8B4FD7EE28644` for both `dist/model-fallback-router.so` and `dist/model-fallback-router-execution-transform-linux-amd64.so`.
- Evidence/report:
  - `.agent/reports/execution-transform-harness-markers-proof-2026-06-29.md`
  - `.agent/cpa-model-fallback-router-execution-middleware-tz/evidence/harness-markers-go-test.txt`
  - `.agent/cpa-model-fallback-router-execution-middleware-tz/evidence/harness-markers-local-cpa-load.logs.txt`
  - `.agent/cpa-model-fallback-router-execution-middleware-tz/evidence/harness-markers-artifact-sha256.txt`

No release/tag/push/production config write was performed. Prod currently still needs an explicit operator step to copy the rebuilt `.so` and add `execution_transform.harness_markers` to the target CPA config.
