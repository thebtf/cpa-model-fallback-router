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
