# v0.2.0 - Execution transform and ordered fallback chains

Non-native models can answer an agent harness with narration instead of using its tools. v0.2.0 adds an opt-in execution transform that makes those turns action-oriented while keeping provider calls, credentials, and protocol conversion inside CLIProxyAPI. It also documents and proves multi-stage fallback across any CPA-supported model names.

## What's Changed

- Transform eligible Claude, OpenAI Chat, and OpenAI Responses requests from structural tool availability, without prompt-phrase triggers.
- Require a tool call when configured and provide an audited `ExitContinuationTool` for legitimate structured exits.
- Bypass post-tool continuation turns so a completed tool call does not force another one.
- Unwrap audited exits only for the exit tool injected by the plugin, without intercepting an operator-defined tool with the same name.
- Preserve CPA's built-in cross-protocol translation by passing distinct request and response protocols to the host executor.
- Support direct same-model transform rules as well as failure fallback rules.
- Fall back on recognized context-window `400` errors while preserving terminal handling for ordinary `400`, `404`, and `422` responses.
- Preserve that context-window fallback before the first stream payload even when CPA's stream bridge loses the numeric status.

## Multi-stage fallback

Add this as an early rule to try Kimi, then Grok, then GPT when each preceding attempt fails with a fallback-eligible error:

```yaml
- name: kimi_to_grok_then_luna
  models: ["kimi-*"]
  primary_model: "$requested"
  fallback_models:
    - grok-4.5
    - gpt-5.6-luna
```

Rules are first-match-wins. Omitting `source_formats` makes this rule protocol-global, and quota status `429` advances the chain by default.

## Validation

- `go test ./...`
- Focused three-attempt executor regression with two `429` responses before success.
- Release-candidate native package and CPA pluginhost smoke tests.

## SDK Compatibility Notes

The plugin remains a CPA host-executor client: it makes no direct provider calls and handles no provider credentials. The execution transform is disabled by default and can be enabled globally or per rule.

## Assets

The release keeps the CPA plugin store artifact layout: one zip per supported platform plus `checksums.txt`. Each zip contains exactly one root-level dynamic library named for the platform.
