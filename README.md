# CPA Model Fallback Router

A CLIProxyAPI plugin that retries a failed model request through fallback model names. It is meant for cases where clients ask for a Claude model but the selected upstream quota expires, and you want CPA to transparently retry with another model such as `gpt-5.4`.

The plugin is intentionally model-name based. It asks CPA to execute the original requested model first, then retries the configured fallback models when the first attempt fails with a fallback-eligible status or transport/quota error.

## Compatibility

- Built for the CPA native plugin ABI from `github.com/router-for-me/CLIProxyAPI/v7`.
- Tested during extraction against the CPA v7.2.x plugin API.
- The current CPA host callback does not expose the selected auth record or auth kind to plugin executors. Because of that, this plugin can scope by requested model and inbound source format, but cannot yet scope a rule to `anthropic oauth` versus an Anthropic API key. Add that only after CPA exposes selected-provider metadata to plugins.

## Install

1. Enable CPA plugins and mount a persistent plugin directory into the CPA container.
2. Download the release asset for the CPA host platform.
3. Rename the asset to the CPA plugin basename for that platform:
   - Linux: `model-fallback-router.so`
   - macOS: `model-fallback-router.dylib`
   - Windows: `model-fallback-router.dll`
4. Put it in the configured plugin directory.
5. Add the plugin config under `plugins.configs.model-fallback-router`.

For an official Linux Docker image, the final mounted path usually looks like this inside the container:

```text
/app/plugins/model-fallback-router.so
```

## CPA Configuration

```yaml
plugins:
  enabled: true
  dir: "/app/plugins"
  configs:
    model-fallback-router:
      enabled: true
      priority: 100

      rules:
        - name: claude_quota_to_gpt54
          source_formats:
            - claude
            - anthropic
          models:
            - claude-*
          primary_model: "$requested"
          fallback_models:
            - gpt-5.4

      fallback:
        enabled: true
        fallback_on_status:
          - 401
          - 403
          - 408
          - 409
          - 429
          - 500
          - 502
          - 503
          - 504
        no_fallback_on_status:
          - 400
          - 404
          - 422
```

## Configuration Rules

- `rules[].models` matches the client-requested model with `*` wildcards.
- `rules[].source_formats` optionally limits the inbound protocol. `anthropic` is normalized to `claude`.
- Omit `source_formats` to make a rule protocol-global.
- `primary_model` defaults to `$requested`, which means the original requested model.
- `fallback_models` are tried in order after a fallback-eligible failure.
- Rule-level `fallback_on_status` and `no_fallback_on_status` override the global fallback status lists for that rule.
- Non-streaming requests can fall back after a failed response.
- Streaming requests fall back only if the failure happens before the first payload chunk is emitted.
- If CPA loses the numeric HTTP status but the error text clearly indicates rate limiting or quota exhaustion, the plugin treats the failure as fallback eligible.

## Build Locally

Windows PowerShell:

```powershell
.\scripts\build.ps1 -GOOS windows -GOARCH amd64
```

Linux/macOS shell:

```bash
./scripts/build.sh
```

Direct Go command for the current platform:

```bash
go test ./...
go build -buildmode=c-shared -o dist/model-fallback-router.so .
```

Cross-compiling a cgo shared library requires a C compiler for the target platform. The GitHub release workflow handles the supported release targets.

## Releases

Push a semver tag to build and publish release assets:

```bash
git tag v0.1.0
git push origin main v0.1.0
```

The release workflow builds:

- `model-fallback-router-linux-amd64.so`
- `model-fallback-router-linux-arm64.so`
- `model-fallback-router-darwin-amd64.dylib`
- `model-fallback-router-darwin-arm64.dylib`
- `model-fallback-router-windows-amd64.dll`
- `SHA256SUMS.txt`

## Documentation

Open `docs/index.html` to view the Redoc-rendered configuration reference. The source spec is `docs/openapi.yaml`.
