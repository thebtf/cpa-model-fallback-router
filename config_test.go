package main

import (
	"strings"
	"testing"
)

func TestDecodeConfigNormalizesRule(t *testing.T) {
	cfg, err := decodeConfig([]byte(`enabled: true
rules:
  - name: claude_quota
    source_formats:
      - anthropic
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
fallback:
  fallback_on_status:
    - 429
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("cfg.Enabled = false, want true")
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("len(cfg.Rules) = %d, want 1", len(cfg.Rules))
	}
	rule := cfg.Rules[0]
	if rule.SourceFormats[0] != "claude" {
		t.Fatalf("SourceFormats[0] = %q, want claude", rule.SourceFormats[0])
	}
	if rule.PrimaryModel != requestedModelToken {
		t.Fatalf("PrimaryModel = %q, want %q", rule.PrimaryModel, requestedModelToken)
	}
	if len(cfg.Fallback.NoFallbackOnStatus) == 0 {
		t.Fatal("NoFallbackOnStatus did not keep defaults")
	}
	if cfg.Fallback.CooldownSeconds != defaultFallbackCooldownSeconds {
		t.Fatalf("CooldownSeconds = %d, want %d", cfg.Fallback.CooldownSeconds, defaultFallbackCooldownSeconds)
	}
}

func TestDecodeConfigDisabledAllowsNoRules(t *testing.T) {
	cfg, err := decodeConfig(nil)
	if err != nil {
		t.Fatalf("decodeConfig(nil) error = %v", err)
	}
	if cfg.Enabled {
		t.Fatal("default config should be disabled")
	}
}

func TestDecodeConfigEnabledRequiresRules(t *testing.T) {
	_, err := decodeConfig([]byte(`enabled: true`))
	if err == nil || !strings.Contains(err.Error(), "requires at least one rule") {
		t.Fatalf("decodeConfig() error = %v, want missing rule error", err)
	}
}

func TestDecodeConfigRejectsInvalidStatus(t *testing.T) {
	_, err := decodeConfig([]byte(`enabled: true
rules:
  - name: bad
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
fallback:
  fallback_on_status:
    - 99
`))
	if err == nil || !strings.Contains(err.Error(), "invalid HTTP status") {
		t.Fatalf("decodeConfig() error = %v, want invalid status", err)
	}
}

func TestDecodeConfigCooldownPolicy(t *testing.T) {
	cfg, err := decodeConfig([]byte(`enabled: true
rules:
  - name: claude_quota
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
fallback:
  cooldown_seconds: 120
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	if cfg.Fallback.CooldownSeconds != 120 {
		t.Fatalf("CooldownSeconds = %d, want 120", cfg.Fallback.CooldownSeconds)
	}
	policy := fallbackPolicy(cfg, cfg.Rules[0])
	if policy.CooldownSeconds != 120 {
		t.Fatalf("fallbackPolicy cooldown = %d, want 120", policy.CooldownSeconds)
	}
}

func TestDecodeConfigRuleCooldownOverrideAllowsZero(t *testing.T) {
	cfg, err := decodeConfig([]byte(`enabled: true
rules:
  - name: claude_quota
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
    cooldown_seconds: 0
fallback:
  cooldown_seconds: 120
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	policy := fallbackPolicy(cfg, cfg.Rules[0])
	if policy.CooldownSeconds != 0 {
		t.Fatalf("rule cooldown override = %d, want 0", policy.CooldownSeconds)
	}
}

func TestDecodeConfigRejectsNegativeCooldown(t *testing.T) {
	_, err := decodeConfig([]byte(`enabled: true
rules:
  - name: bad
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
fallback:
  cooldown_seconds: -1
`))
	if err == nil || !strings.Contains(err.Error(), "cooldown_seconds") {
		t.Fatalf("decodeConfig() error = %v, want cooldown error", err)
	}

	_, err = decodeConfig([]byte(`enabled: true
rules:
  - name: bad
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
    cooldown_seconds: -1
`))
	if err == nil || !strings.Contains(err.Error(), "cooldown_seconds") {
		t.Fatalf("decodeConfig() error = %v, want rule cooldown error", err)
	}
}

func TestDecodeConfigExecutionTransform(t *testing.T) {
	cfg, err := decodeConfig([]byte(`enabled: true
execution_transform:
  enabled: false
  telemetry: true
  activation: tool_surface
  force_tools: never
  execution_envelope: "Execution mode is active. Custom execution envelope for OMP handoff loops."
  max_envelope_chars: 512
  source_formats: [anthropic]
  requested_model_patterns: ["claude-*"]
  selected_model_patterns: ["gpt-*"]
  exit_tool:
    name: ExitContinuationTool
rules:
  - name: claude_code
    source_formats: [claude]
    models: ["claude-*"]
    fallback_models: [gpt-5.5]
    execution_transform:
      enabled: true
      force_tools: when_available
fallback:
  cooldown_seconds: 60
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	settings := resolveExecutionTransform(cfg, cfg.Rules[0])
	if !settings.Enabled {
		t.Fatal("rule execution_transform.enabled did not override global false")
	}
	if settings.Activation != executionTransformActivationToolSurface {
		t.Fatalf("Activation = %q, want %q inherited from global", settings.Activation, executionTransformActivationToolSurface)
	}
	if settings.ForceTools != executionTransformForceToolsWhenAvailable {
		t.Fatalf("ForceTools = %q, want %q", settings.ForceTools, executionTransformForceToolsWhenAvailable)
	}
	if settings.MaxEnvelopeChars != 512 {
		t.Fatalf("MaxEnvelopeChars = %d, want 512 inherited from global", settings.MaxEnvelopeChars)
	}
	if settings.ExecutionEnvelope != "Execution mode is active. Custom execution envelope for OMP handoff loops." {
		t.Fatalf("ExecutionEnvelope = %q, want configured envelope", settings.ExecutionEnvelope)
	}
	if !stringInList("claude", settings.SourceFormats) {
		t.Fatalf("SourceFormats = %#v, want anthropic normalized to claude", settings.SourceFormats)
	}
}

func TestDecodeConfigRejectsInvalidExecutionTransform(t *testing.T) {
	_, err := decodeConfig([]byte(`enabled: true
execution_transform:
  activation: poetry
rules:
  - name: bad
    models: ["claude-*"]
    fallback_models: [gpt-5.5]
`))
	if err == nil || !strings.Contains(err.Error(), "execution_transform.activation") {
		t.Fatalf("decodeConfig() error = %v, want activation error", err)
	}

	_, err = decodeConfig([]byte(`enabled: true
execution_transform:
  activation: harness_signal
rules:
  - name: bad
    models: ["claude-*"]
    fallback_models: [gpt-5.5]
`))
	if err == nil || !strings.Contains(err.Error(), "execution_transform.activation") {
		t.Fatalf("decodeConfig() error = %v, want legacy activation error", err)
	}
	_, err = decodeConfig([]byte(`enabled: true
execution_transform:
  force_tools: sometimes
rules:
  - name: bad
    models: ["claude-*"]
    fallback_models: [gpt-5.5]
`))
	if err == nil || !strings.Contains(err.Error(), "execution_transform.force_tools") {
		t.Fatalf("decodeConfig() error = %v, want force_tools error", err)
	}

	_, err = decodeConfig([]byte(`enabled: true
execution_transform:
  exit_tool:
    name: "bad name"
rules:
  - name: bad
    models: ["claude-*"]
    fallback_models: [gpt-5.5]
`))
	if err == nil || !strings.Contains(err.Error(), "execution_transform.exit_tool.name") {
		t.Fatalf("decodeConfig() error = %v, want exit tool name error", err)
	}
}
