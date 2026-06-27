package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var defaultFallbackOnStatus = []int{401, 403, 408, 409, 429, 500, 502, 503, 504}
var defaultNoFallbackOnStatus = []int{400, 404, 422}

const defaultFallbackCooldownSeconds = 60

type pluginConfig struct {
	Enabled  bool             `yaml:"enabled"`
	Rules    []fallbackRule   `yaml:"rules"`
	Fallback fallbackSettings `yaml:"fallback"`
}

type fallbackRule struct {
	Name               string   `yaml:"name"`
	SourceFormats      []string `yaml:"source_formats"`
	Models             []string `yaml:"models"`
	PrimaryModel       string   `yaml:"primary_model"`
	FallbackModels     []string `yaml:"fallback_models"`
	FallbackOnStatus   []int    `yaml:"fallback_on_status"`
	NoFallbackOnStatus []int    `yaml:"no_fallback_on_status"`
	CooldownSeconds    *int     `yaml:"cooldown_seconds"`
	Order              int      `yaml:"-"`
}

type fallbackSettings struct {
	Enabled            bool  `yaml:"enabled"`
	FallbackOnStatus   []int `yaml:"fallback_on_status"`
	NoFallbackOnStatus []int `yaml:"no_fallback_on_status"`
	CooldownSeconds    int   `yaml:"cooldown_seconds"`
}

func defaultPluginConfig() pluginConfig {
	return pluginConfig{
		Enabled: false,
		Fallback: fallbackSettings{
			Enabled:            true,
			FallbackOnStatus:   append([]int(nil), defaultFallbackOnStatus...),
			NoFallbackOnStatus: append([]int(nil), defaultNoFallbackOnStatus...),
			CooldownSeconds:    defaultFallbackCooldownSeconds,
		},
	}
}

func decodeConfig(raw []byte) (pluginConfig, error) {
	cfg := defaultPluginConfig()
	if strings.TrimSpace(string(raw)) != "" {
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return pluginConfig{}, fmt.Errorf("invalid %s config: %w", pluginIdentifier, err)
		}
	}
	normalizeConfig(&cfg)
	if err := validateConfig(cfg); err != nil {
		return pluginConfig{}, err
	}
	return cfg, nil
}

func normalizeConfig(cfg *pluginConfig) {
	if cfg == nil {
		return
	}
	cfg.Rules = append([]fallbackRule(nil), cfg.Rules...)
	for i := range cfg.Rules {
		rule := &cfg.Rules[i]
		rule.Name = strings.TrimSpace(rule.Name)
		rule.SourceFormats = normalizeStringList(rule.SourceFormats, true)
		for j := range rule.SourceFormats {
			rule.SourceFormats[j] = normalizeProtocol(rule.SourceFormats[j])
		}
		rule.Models = normalizeStringList(rule.Models, false)
		rule.PrimaryModel = strings.TrimSpace(rule.PrimaryModel)
		if rule.PrimaryModel == "" {
			rule.PrimaryModel = requestedModelToken
		}
		rule.FallbackModels = normalizeStringList(rule.FallbackModels, false)
		rule.Order = i
	}
	sort.SliceStable(cfg.Rules, func(i, j int) bool {
		return cfg.Rules[i].Order < cfg.Rules[j].Order
	})
}

func validateConfig(cfg pluginConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if len(cfg.Rules) == 0 {
		return fmt.Errorf("%s config requires at least one rule", pluginIdentifier)
	}
	if err := validateStatusCodes("fallback.fallback_on_status", cfg.Fallback.FallbackOnStatus); err != nil {
		return err
	}
	if err := validateStatusCodes("fallback.no_fallback_on_status", cfg.Fallback.NoFallbackOnStatus); err != nil {
		return err
	}
	if cfg.Fallback.CooldownSeconds < 0 {
		return fmt.Errorf("fallback.cooldown_seconds must be >= 0")
	}
	for i, rule := range cfg.Rules {
		prefix := fmt.Sprintf("%s rules[%d]", pluginIdentifier, i)
		if rule.Name == "" {
			return fmt.Errorf("%s requires name", prefix)
		}
		if len(rule.Models) == 0 {
			return fmt.Errorf("%s requires at least one model pattern", prefix)
		}
		if len(rule.FallbackModels) == 0 {
			return fmt.Errorf("%s requires at least one fallback_models entry", prefix)
		}
		if err := validateStatusCodes(prefix+".fallback_on_status", rule.FallbackOnStatus); err != nil {
			return err
		}
		if err := validateStatusCodes(prefix+".no_fallback_on_status", rule.NoFallbackOnStatus); err != nil {
			return err
		}
		if rule.CooldownSeconds != nil && *rule.CooldownSeconds < 0 {
			return fmt.Errorf("%s.cooldown_seconds must be >= 0", prefix)
		}
	}
	return nil
}

func validateStatusCodes(field string, codes []int) error {
	for _, code := range codes {
		if code < http.StatusContinue || code > 599 {
			return fmt.Errorf("%s contains invalid HTTP status %d", field, code)
		}
	}
	return nil
}

func normalizeStringList(input []string, lower bool) []string {
	out := make([]string, 0, len(input))
	for _, item := range input {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if lower {
			item = strings.ToLower(item)
		}
		out = append(out, item)
	}
	return out
}
