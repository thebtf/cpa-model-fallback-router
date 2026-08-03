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
	Enabled            bool                       `yaml:"enabled"`
	Rules              []fallbackRule             `yaml:"rules"`
	Fallback           fallbackSettings           `yaml:"fallback"`
	ExecutionTransform executionTransformSettings `yaml:"execution_transform"`
}

type fallbackRule struct {
	Name               string                      `yaml:"name"`
	SourceFormats      []string                    `yaml:"source_formats"`
	Models             []string                    `yaml:"models"`
	PrimaryModel       string                      `yaml:"primary_model"`
	FallbackModels     []string                    `yaml:"fallback_models"`
	FallbackOnStatus   []int                       `yaml:"fallback_on_status"`
	NoFallbackOnStatus []int                       `yaml:"no_fallback_on_status"`
	CooldownSeconds    *int                        `yaml:"cooldown_seconds"`
	ExecutionTransform *executionTransformSettings `yaml:"execution_transform"`
	Order              int                         `yaml:"-"`
}

type fallbackSettings struct {
	Enabled            bool  `yaml:"enabled"`
	FallbackOnStatus   []int `yaml:"fallback_on_status"`
	NoFallbackOnStatus []int `yaml:"no_fallback_on_status"`
	CooldownSeconds    int   `yaml:"cooldown_seconds"`
}

const (
	executionTransformForceToolsNever             = "never"
	executionTransformForceToolsWhenAvailable     = "when_available"
	executionTransformForceToolsAlwaysIfSupported = "always_if_supported"
	executionTransformActivationToolSurface       = "tool_surface"
	executionTransformActivationAlways            = "always"
	defaultExecutionEnvelopeChars                 = 1600
	defaultAuditedExitToolName                    = "ExitContinuationTool"
)

type executionTransformSettings struct {
	Enabled                 *bool                   `yaml:"enabled"`
	DryRun                  *bool                   `yaml:"dry_run"`
	Telemetry               *bool                   `yaml:"telemetry"`
	Activation              string                  `yaml:"activation"`
	ForceTools              string                  `yaml:"force_tools"`
	ExecutionEnvelope       string                  `yaml:"execution_envelope"`
	InjectExecutionEnvelope *bool                   `yaml:"inject_execution_envelope"`
	AddAuditedExitTool      *bool                   `yaml:"add_audited_exit_tool"`
	MaxEnvelopeChars        *int                    `yaml:"max_envelope_chars"`
	SourceFormats           []string                `yaml:"source_formats"`
	RequestedModelPatterns  []string                `yaml:"requested_model_patterns"`
	SelectedModelPatterns   []string                `yaml:"selected_model_patterns"`
	ExitTool                auditedExitToolSettings `yaml:"exit_tool"`
	Strict                  *bool                   `yaml:"strict"`
}

type auditedExitToolSettings struct {
	Name string `yaml:"name"`
}

type resolvedExecutionTransform struct {
	Enabled                 bool
	DryRun                  bool
	Telemetry               bool
	Activation              string
	ForceTools              string
	ExecutionEnvelope       string
	InjectExecutionEnvelope bool
	AddAuditedExitTool      bool
	MaxEnvelopeChars        int
	SourceFormats           []string
	RequestedModelPatterns  []string
	SelectedModelPatterns   []string
	ExitToolName            string
	Strict                  bool
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
		ExecutionTransform: defaultExecutionTransformSettings(),
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
	normalizeExecutionTransformSettings(&cfg.ExecutionTransform)
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
		if rule.ExecutionTransform != nil {
			normalizeExecutionTransformSettings(rule.ExecutionTransform)
		}
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
	if err := validateExecutionTransformSettings("execution_transform", cfg.ExecutionTransform); err != nil {
		return err
	}
	if err := validateResolvedExecutionTransform("execution_transform", resolveExecutionTransform(cfg, fallbackRule{})); err != nil {
		return err
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
		if rule.ExecutionTransform != nil {
			field := prefix + ".execution_transform"
			if err := validateExecutionTransformSettings(field, *rule.ExecutionTransform); err != nil {
				return err
			}
			if err := validateResolvedExecutionTransform(field, resolveExecutionTransform(cfg, rule)); err != nil {
				return err
			}
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

func defaultExecutionTransformSettings() executionTransformSettings {
	return executionTransformSettings{
		Enabled:                 boolPtr(false),
		DryRun:                  boolPtr(false),
		Telemetry:               boolPtr(true),
		Activation:              executionTransformActivationToolSurface,
		ForceTools:              executionTransformForceToolsWhenAvailable,
		ExecutionEnvelope:       defaultExecutionEnvelope,
		InjectExecutionEnvelope: boolPtr(true),
		AddAuditedExitTool:      boolPtr(true),
		MaxEnvelopeChars:        intPtr(defaultExecutionEnvelopeChars),
		SourceFormats:           []string{"claude"},
		RequestedModelPatterns:  []string{"claude-*"},
		SelectedModelPatterns:   []string{"gpt-*"},
		ExitTool:                auditedExitToolSettings{Name: defaultAuditedExitToolName},
		Strict:                  boolPtr(false),
	}
}

func boolPtr(value bool) *bool { return &value }

func intPtr(value int) *int { return &value }

func normalizeExecutionTransformSettings(settings *executionTransformSettings) {
	if settings == nil {
		return
	}
	settings.Activation = strings.TrimSpace(settings.Activation)
	settings.ForceTools = strings.TrimSpace(settings.ForceTools)
	settings.ExecutionEnvelope = strings.TrimSpace(settings.ExecutionEnvelope)
	settings.SourceFormats = normalizeStringList(settings.SourceFormats, true)
	for i := range settings.SourceFormats {
		settings.SourceFormats[i] = normalizeProtocol(settings.SourceFormats[i])
	}
	settings.RequestedModelPatterns = normalizeStringList(settings.RequestedModelPatterns, false)
	settings.SelectedModelPatterns = normalizeStringList(settings.SelectedModelPatterns, false)
	settings.ExitTool.Name = strings.TrimSpace(settings.ExitTool.Name)
}

func resolveExecutionTransform(cfg pluginConfig, rule fallbackRule) resolvedExecutionTransform {
	resolved := resolvedExecutionTransform{}
	resolved.apply(defaultExecutionTransformSettings())
	resolved.apply(cfg.ExecutionTransform)
	if rule.ExecutionTransform != nil {
		resolved.apply(*rule.ExecutionTransform)
	}
	return resolved
}

func (resolved *resolvedExecutionTransform) apply(settings executionTransformSettings) {
	if settings.Enabled != nil {
		resolved.Enabled = *settings.Enabled
	}
	if settings.DryRun != nil {
		resolved.DryRun = *settings.DryRun
	}
	if settings.Telemetry != nil {
		resolved.Telemetry = *settings.Telemetry
	}
	if strings.TrimSpace(settings.Activation) != "" {
		resolved.Activation = strings.TrimSpace(settings.Activation)
	}
	if strings.TrimSpace(settings.ForceTools) != "" {
		resolved.ForceTools = strings.TrimSpace(settings.ForceTools)
	}
	if strings.TrimSpace(settings.ExecutionEnvelope) != "" {
		resolved.ExecutionEnvelope = strings.TrimSpace(settings.ExecutionEnvelope)
	}
	if settings.InjectExecutionEnvelope != nil {
		resolved.InjectExecutionEnvelope = *settings.InjectExecutionEnvelope
	}
	if settings.AddAuditedExitTool != nil {
		resolved.AddAuditedExitTool = *settings.AddAuditedExitTool
	}
	if settings.MaxEnvelopeChars != nil {
		resolved.MaxEnvelopeChars = *settings.MaxEnvelopeChars
	}
	if len(settings.SourceFormats) > 0 {
		resolved.SourceFormats = append([]string(nil), settings.SourceFormats...)
	}
	if len(settings.RequestedModelPatterns) > 0 {
		resolved.RequestedModelPatterns = append([]string(nil), settings.RequestedModelPatterns...)
	}
	if len(settings.SelectedModelPatterns) > 0 {
		resolved.SelectedModelPatterns = append([]string(nil), settings.SelectedModelPatterns...)
	}
	if strings.TrimSpace(settings.ExitTool.Name) != "" {
		resolved.ExitToolName = strings.TrimSpace(settings.ExitTool.Name)
	}
	if settings.Strict != nil {
		resolved.Strict = *settings.Strict
	}
}

func validateExecutionTransformSettings(field string, settings executionTransformSettings) error {
	if settings.Activation != "" && !validExecutionTransformActivation(settings.Activation) {
		return fmt.Errorf("%s.activation must be one of tool_surface, always", field)
	}
	if settings.ForceTools != "" && !validExecutionTransformForceTools(settings.ForceTools) {
		return fmt.Errorf("%s.force_tools must be one of never, when_available, always_if_supported", field)
	}
	if settings.MaxEnvelopeChars != nil {
		if err := validateExecutionTransformMaxEnvelope(field+".max_envelope_chars", *settings.MaxEnvelopeChars); err != nil {
			return err
		}
	}
	if settings.ExitTool.Name != "" && !safeToolIdentifier(settings.ExitTool.Name) {
		return fmt.Errorf("%s.exit_tool.name must be a safe identifier", field)
	}
	return nil
}

func validateResolvedExecutionTransform(field string, settings resolvedExecutionTransform) error {
	if !validExecutionTransformActivation(settings.Activation) {
		return fmt.Errorf("%s.activation must be one of tool_surface, always", field)
	}
	if !validExecutionTransformForceTools(settings.ForceTools) {
		return fmt.Errorf("%s.force_tools must be one of never, when_available, always_if_supported", field)
	}
	if err := validateExecutionTransformMaxEnvelope(field+".max_envelope_chars", settings.MaxEnvelopeChars); err != nil {
		return err
	}
	if !safeToolIdentifier(settings.ExitToolName) {
		return fmt.Errorf("%s.exit_tool.name must be a safe identifier", field)
	}
	return nil
}

func validateExecutionTransformMaxEnvelope(field string, value int) error {
	if value == 0 {
		return nil
	}
	if value < 256 || value > 8000 {
		return fmt.Errorf("%s must be 0 or between 256 and 8000", field)
	}
	return nil
}

func validExecutionTransformForceTools(value string) bool {
	switch strings.TrimSpace(value) {
	case executionTransformForceToolsNever, executionTransformForceToolsWhenAvailable, executionTransformForceToolsAlwaysIfSupported:
		return true
	default:
		return false
	}
}

func validExecutionTransformActivation(value string) bool {
	switch strings.TrimSpace(value) {
	case executionTransformActivationToolSurface, executionTransformActivationAlways:
		return true
	default:
		return false
	}
}

func safeToolIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for i, r := range value {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
