package main

import "testing"

func TestMatchingRuleAndAttempts(t *testing.T) {
	cfg, err := decodeConfig([]byte(`enabled: true
rules:
  - name: claude_to_gpt
    source_formats:
      - anthropic
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
      - "$requested"
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	rule, ok := matchingRule(cfg, "anthropic", "claude-sonnet-4-5")
	if !ok {
		t.Fatal("matchingRule() did not match anthropic claude request")
	}
	attempts := buildAttempts(rule, "claude-sonnet-4-5")
	want := []string{"claude-sonnet-4-5", "gpt-5.4"}
	if len(attempts) != len(want) {
		t.Fatalf("attempts = %#v, want %#v", attempts, want)
	}
	for i := range want {
		if attempts[i] != want[i] {
			t.Fatalf("attempts = %#v, want %#v", attempts, want)
		}
	}
}

func TestMatchingRuleRejectsSourceMismatch(t *testing.T) {
	cfg, err := decodeConfig([]byte(`enabled: true
rules:
  - name: claude_only
    source_formats:
      - claude
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	if _, ok := matchingRule(cfg, "openai", "claude-sonnet-4-5"); ok {
		t.Fatal("matchingRule() matched unexpected source format")
	}
}

func TestWildcardMatch(t *testing.T) {
	cases := []struct {
		value   string
		pattern string
		want    bool
	}{
		{value: "claude-sonnet-4-5", pattern: "claude-*", want: true},
		{value: "prefix-middle-suffix", pattern: "prefix*suffix", want: true},
		{value: "prefix-middle-suffix", pattern: "middle*", want: false},
		{value: "gpt-5.4", pattern: "claude-*", want: false},
	}
	for _, tc := range cases {
		if got := wildcardMatch(tc.value, tc.pattern); got != tc.want {
			t.Fatalf("wildcardMatch(%q, %q) = %v, want %v", tc.value, tc.pattern, got, tc.want)
		}
	}
}
