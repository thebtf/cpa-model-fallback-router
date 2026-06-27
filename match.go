package main

import (
	"strings"
)

const requestedModelToken = "$requested"

func matchingRule(cfg pluginConfig, sourceFormat, requestedModel string) (fallbackRule, bool) {
	if !cfg.Enabled {
		return fallbackRule{}, false
	}
	source := normalizeProtocol(sourceFormat)
	requested := strings.TrimSpace(requestedModel)
	if requested == "" {
		return fallbackRule{}, false
	}
	for _, rule := range cfg.Rules {
		if len(rule.SourceFormats) > 0 && !stringInList(source, rule.SourceFormats) {
			continue
		}
		if !matchesAnyPattern(requested, rule.Models) {
			continue
		}
		return rule, true
	}
	return fallbackRule{}, false
}

type attemptPlan struct {
	Attempts       []string
	Primary        string
	PrimarySkipped bool
}

func buildAttempts(rule fallbackRule, requestedModel string) []string {
	return buildAttemptPlan(rule, requestedModel, false).Attempts
}

func buildAttemptPlan(rule fallbackRule, requestedModel string, skipPrimary bool) attemptPlan {
	requested := strings.TrimSpace(requestedModel)
	primary := resolveModelToken(rule.PrimaryModel, requested)
	out := make([]string, 0, 1+len(rule.FallbackModels))
	if primary != "" && !skipPrimary {
		out = append(out, primary)
	}
	for _, model := range rule.FallbackModels {
		resolved := resolveModelToken(model, requested)
		if resolved == "" {
			continue
		}
		if skipPrimary && strings.EqualFold(resolved, primary) {
			continue
		}
		out = append(out, resolved)
	}
	return attemptPlan{Attempts: dedupeStrings(out), Primary: primary, PrimarySkipped: skipPrimary && primary != ""}
}

func resolveModelToken(model, requested string) string {
	model = strings.TrimSpace(model)
	if strings.EqualFold(model, requestedModelToken) {
		return strings.TrimSpace(requested)
	}
	return model
}

func normalizeProtocol(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "anthropic":
		return "claude"
	case "responses", "openai-responses", "openai_responses":
		return "openai-response"
	case "chat-completions", "chat_completions", "openai-chat-completions", "openai_chat_completions":
		return "openai"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func matchesAnyPattern(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if wildcardMatch(value, pattern) {
			return true
		}
	}
	return false
}

func wildcardMatch(value, pattern string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return value == pattern
	}
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(value[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
			return false
		}
		pos += idx + len(part)
	}
	last := parts[len(parts)-1]
	if last != "" && !strings.HasSuffix(pattern, "*") && !strings.HasSuffix(value, last) {
		return false
	}
	return true
}

func stringInList(value string, list []string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}

func dedupeStrings(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	out := make([]string, 0, len(input))
	for _, item := range input {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}
