package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestApplyExecutionTransformClaudeToolMode(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	rule := cfg.Rules[0]
	exec := executionTransformExecutorRequest()
	body := requestBodyForModel(exec.OriginalRequest, "gpt-5.5")

	got, metadata, exitToolAdded, err := applyExecutionTransform(cfg, exec, rule, "gpt-5.5", "claude", body)
	if err != nil {
		t.Fatalf("applyExecutionTransform() error = %v", err)
	}
	if bytes.Equal(got, body) {
		t.Fatal("applyExecutionTransform() left body unchanged, want mutation")
	}
	if !exitToolAdded {
		t.Fatal("applyExecutionTransform() exitToolAdded = false, want true")
	}
	var obj map[string]any
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode transformed body: %v; body=%s", err, got)
	}
	if !strings.Contains(asString(obj["system"]), "Execution mode is active. Custom execution envelope: perform a real state-changing tool call before prose.") {
		t.Fatalf("system = %#v, want configured execution envelope", obj["system"])
	}
	if !hasToolNamed(obj["tools"], "ExitContinuationTool") {
		t.Fatalf("tools = %#v, want ExitContinuationTool", obj["tools"])
	}
	choice, ok := obj["tool_choice"].(map[string]any)
	if !ok || choice["type"] != "any" {
		t.Fatalf("tool_choice = %#v, want Anthropic required-any", obj["tool_choice"])
	}
	transformMeta, ok := metadata["execution_transform"].(map[string]any)
	if !ok {
		t.Fatalf("metadata = %#v, want execution_transform", metadata)
	}
	if transformMeta["applied"] != true || transformMeta["forced_tools"] != true || transformMeta["exit_tool_added"] != true {
		t.Fatalf("execution_transform metadata = %#v", transformMeta)
	}
	if reason := asString(transformMeta["reason"]); reason != "activation:tool_surface" {
		t.Fatalf("execution_transform reason = %q, want structural tool-surface activation", reason)
	}
	if _, leaked := transformMeta["body"]; leaked {
		t.Fatalf("execution_transform metadata leaked body: %#v", transformMeta)
	}
}

func TestApplyExecutionTransformDryRunLeavesBody(t *testing.T) {
	cfg := executionTransformTestConfig(t, true)
	exec := executionTransformExecutorRequest()
	body := requestBodyForModel(exec.OriginalRequest, "gpt-5.5")

	got, metadata, _, err := applyExecutionTransform(cfg, exec, cfg.Rules[0], "gpt-5.5", "claude", body)
	if err != nil {
		t.Fatalf("applyExecutionTransform() error = %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("dry-run body mutated: %s", got)
	}
	transformMeta := metadata["execution_transform"].(map[string]any)
	if transformMeta["dry_run"] != true || transformMeta["applied"] != false {
		t.Fatalf("dry-run metadata = %#v", transformMeta)
	}
}

func TestApplyExecutionTransformBypassesWithoutToolSurface(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	exec := executionTransformExecutorRequest()
	exec.OriginalRequest = []byte(`{"model":"claude-sonnet-4-5","system":"Claude Code","messages":[{"role":"user","content":"review this Markdown and summarize it"}]}`)
	body := requestBodyForModel(exec.OriginalRequest, "gpt-5.5")

	got, metadata, _, err := applyExecutionTransform(cfg, exec, cfg.Rules[0], "gpt-5.5", "claude", body)
	if err != nil {
		t.Fatalf("applyExecutionTransform() error = %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("no-tool body mutated: %s", got)
	}
	transformMeta := metadata["execution_transform"].(map[string]any)
	if transformMeta["applied"] != false || transformMeta["bypassed_reason"] != "no_tools_available" {
		t.Fatalf("no-tool metadata = %#v", transformMeta)
	}
}

func TestApplyExecutionTransformToolSurfacePreservesReviewOnlyScope(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	exec := executionTransformExecutorRequest()
	exec.OriginalRequest = []byte(`{"model":"claude-opus-4-8","system":"You are Claude Code. <system-reminder>Mode: review only; do not execute production-code edits.</system-reminder>","tools":[{"name":"Read","input_schema":{"type":"object"}},{"name":"Write","input_schema":{"type":"object"}}],"messages":[{"role":"user","content":"Write the handoff in the coordination state, but do not edit production code."}]}`)
	body := requestBodyForModel(exec.OriginalRequest, "gpt-5.5")

	got, metadata, _, err := applyExecutionTransform(cfg, exec, cfg.Rules[0], "gpt-5.5", "claude", body)
	if err != nil {
		t.Fatalf("applyExecutionTransform() error = %v", err)
	}
	if bytes.Equal(got, body) {
		t.Fatal("review-only scoped tool-surface body was not transformed")
	}
	var obj map[string]any
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode transformed body: %v; body=%s", err, got)
	}
	if !strings.Contains(asString(obj["system"]), "Execution mode is active") {
		t.Fatalf("system = %#v, want execution envelope", obj["system"])
	}
	if !strings.Contains(asString(obj["system"]), "do not execute production-code edits") {
		t.Fatalf("system = %#v, want original scope constraint preserved", obj["system"])
	}
	if !hasToolNamed(obj["tools"], "ExitContinuationTool") {
		t.Fatalf("tools = %#v, want ExitContinuationTool", obj["tools"])
	}
	transformMeta, ok := metadata["execution_transform"].(map[string]any)
	if !ok || transformMeta["applied"] != true || asString(transformMeta["reason"]) != "activation:tool_surface" {
		t.Fatalf("metadata = %#v, want structural tool-surface execution_transform", metadata)
	}
}

func TestApplyExecutionTransformBypassesClaudePostToolContinuation(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	exec := executionTransformExecutorRequest()
	exec.OriginalRequest = []byte(`{"model":"claude-sonnet-4-5","system":"You are Claude Code.","tools":[{"name":"TaskUpdate","input_schema":{"type":"object","properties":{}}}],"messages":[{"role":"user","content":"Run the global rules update."},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"TaskUpdate","input":{"id":"task-1","status":"completed"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"global rules update complete"}]}]}`)
	body := requestBodyForModel(exec.OriginalRequest, "gpt-5.5")

	got, metadata, _, err := applyExecutionTransform(cfg, exec, cfg.Rules[0], "gpt-5.5", "claude", body)
	if err != nil {
		t.Fatalf("applyExecutionTransform() error = %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("post-tool continuation body mutated:\n got: %s\nwant: %s", got, body)
	}
	transformMeta := metadata["execution_transform"].(map[string]any)
	if transformMeta["applied"] != false || transformMeta["bypassed_reason"] != "post_tool_continuation" {
		t.Fatalf("post-tool continuation metadata = %#v", transformMeta)
	}
	if strings.Contains(string(got), "Execution mode is active") || strings.Contains(string(got), "ExitContinuationTool") || strings.Contains(string(got), "tool_choice") {
		t.Fatalf("post-tool continuation received execution pressure: %s", got)
	}
}

func TestApplyExecutionTransformBypassesOpenAIResponsesToolOutputContinuation(t *testing.T) {
	cfg, err := decodeConfig([]byte(`enabled: true
execution_transform:
  enabled: false
  telemetry: true
rules:
  - name: omp_gpt_responses_execution
    source_formats: [openai-response]
    models: ["gpt-5.5"]
    primary_model: "$requested"
    fallback_models: ["$requested"]
    execution_transform:
      enabled: true
      activation: tool_surface
      force_tools: when_available
      source_formats: [openai-response]
      requested_model_patterns: ["gpt-*"]
      selected_model_patterns: ["gpt-*"]
      execution_envelope: "Execution mode is active. Direct OMP Responses turn must use a tool before prose."
      add_audited_exit_tool: true
      exit_tool:
        name: ExitContinuationTool
fallback:
  cooldown_seconds: 60
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	exec := pluginapi.ExecutorRequest{
		Model:           "gpt-5.5",
		SourceFormat:    "openai-response",
		OriginalRequest: []byte(`{"model":"gpt-5.5","instructions":"You are OMP.","input":[{"role":"user","content":[{"type":"input_text","text":"Update the state."}]},{"type":"function_call","call_id":"call_1","name":"write","arguments":"{}"},{"type":"function_call_output","call_id":"call_1","output":"state updated"}],"tools":[{"type":"function","name":"write","parameters":{"type":"object","properties":{}}}],"stream":true}`),
	}
	body := requestBodyForModel(exec.OriginalRequest, "gpt-5.5")

	got, metadata, _, err := applyExecutionTransform(cfg, exec, cfg.Rules[0], "gpt-5.5", "openai-response", body)
	if err != nil {
		t.Fatalf("applyExecutionTransform() error = %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("Responses post-tool continuation body mutated:\n got: %s\nwant: %s", got, body)
	}
	transformMeta := metadata["execution_transform"].(map[string]any)
	if transformMeta["applied"] != false || transformMeta["bypassed_reason"] != "post_tool_continuation" {
		t.Fatalf("Responses post-tool continuation metadata = %#v", transformMeta)
	}
	if strings.Contains(string(got), "Direct OMP Responses turn must use a tool") || strings.Contains(string(got), "ExitContinuationTool") || strings.Contains(string(got), "tool_choice") {
		t.Fatalf("Responses post-tool continuation received execution pressure: %s", got)
	}
}
func TestRunExecutionFallbackTransformsOnlySelectedGPTAttempt(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	currentConfig.Store(cfg)
	originalExec := executeHostModelAttempt
	originalCooldowns := primaryCooldowns
	primaryCooldowns = newPrimaryCooldownStore(func() time.Time { return time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC) })
	t.Cleanup(func() {
		executeHostModelAttempt = originalExec
		primaryCooldowns = originalCooldowns
		currentConfig.Store(defaultPluginConfig())
	})

	bodies := make(map[string][]byte)
	executeHostModelAttempt = func(_ pluginapi.ExecutorRequest, _ string, model, _, _ string, body []byte) (pluginapi.HostModelExecutionResponse, error) {
		bodies[model] = append([]byte(nil), body...)
		if model == "claude-sonnet-4-5" {
			return pluginapi.HostModelExecutionResponse{}, errors.New("auth_unavailable: no auth available")
		}
		return pluginapi.HostModelExecutionResponse{StatusCode: http.StatusOK, Body: []byte(`{"ok":true}`)}, nil
	}

	_, _, metadata, err := runExecutionFallback(executionTransformExecutorRequest(), "callback-1")
	if err != nil {
		t.Fatalf("runExecutionFallback() error = %v", err)
	}
	if strings.Contains(string(bodies["claude-sonnet-4-5"]), "Execution mode is active") {
		t.Fatalf("primary body was transformed: %s", bodies["claude-sonnet-4-5"])
	}
	if !strings.Contains(string(bodies["gpt-5.5"]), "Execution mode is active") {
		t.Fatalf("fallback body was not transformed: %s", bodies["gpt-5.5"])
	}
	transformMeta, ok := metadata["execution_transform"].(map[string]any)
	if !ok || transformMeta["applied"] != true {
		t.Fatalf("metadata = %#v, want applied execution_transform", metadata)
	}
}

func TestRunExecutionFallbackDoesNotUnwrapPreexistingAuditedExitTool(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	currentConfig.Store(cfg)
	originalExec := executeHostModelAttempt
	originalCooldowns := primaryCooldowns
	primaryCooldowns = newPrimaryCooldownStore(func() time.Time { return time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC) })
	t.Cleanup(func() {
		executeHostModelAttempt = originalExec
		primaryCooldowns = originalCooldowns
		currentConfig.Store(defaultPluginConfig())
	})

	exec := executionTransformExecutorRequest()
	exec.OriginalRequest = []byte(`{"model":"claude-sonnet-4-5","system":"You are Claude Code.","tools":[{"name":"Read","input_schema":{"type":"object","properties":{}}},{"name":"ExitContinuationTool","input_schema":{"type":"object","properties":{}}}],"messages":[{"role":"user","content":"Continue from the saved state."}]}`)
	response := []byte(`{"content":[{"type":"tool_use","id":"toolu_exit","name":"ExitContinuationTool","input":{"status":"blocked","reason":"waiting for an external owner","user_response":"Waiting for the external owner."}}]}`)
	executeHostModelAttempt = func(_ pluginapi.ExecutorRequest, _ string, model, _, _ string, _ []byte) (pluginapi.HostModelExecutionResponse, error) {
		if model == "claude-sonnet-4-5" {
			return pluginapi.HostModelExecutionResponse{}, errors.New("auth_unavailable: no auth available")
		}
		return pluginapi.HostModelExecutionResponse{StatusCode: http.StatusOK, Body: response}, nil
	}

	body, _, metadata, err := runExecutionFallback(exec, "callback-1")
	if err != nil {
		t.Fatalf("runExecutionFallback() error = %v", err)
	}
	if !bytes.Equal(body, response) {
		t.Fatalf("pre-existing audited exit tool response was rewritten:\n got: %s\nwant: %s", body, response)
	}
	if _, unwrapped := metadata["audited_exit"]; unwrapped {
		t.Fatalf("metadata = %#v, want no audited_exit unwrap for a pre-existing tool", metadata)
	}
}

func TestExecutionTransformSolvesPassiveGPTFallbackEmulator(t *testing.T) {
	originalExec := executeHostModelAttempt
	originalCooldowns := primaryCooldowns
	t.Cleanup(func() {
		executeHostModelAttempt = originalExec
		primaryCooldowns = originalCooldowns
		currentConfig.Store(defaultPluginConfig())
	})

	baselineBody, baselineMetadata := runPassiveGPTEmulator(t, executionTransformDisabledTestConfig(t))
	if strings.Contains(string(baselineBody), `"tool_use"`) {
		t.Fatalf("baseline unexpectedly used a tool: %s", baselineBody)
	}
	if !strings.Contains(string(baselineBody), "I would inspect the saved session state") {
		t.Fatalf("baseline body = %s, want passive explanatory text", baselineBody)
	}
	if _, transformed := baselineMetadata["execution_transform"]; transformed {
		t.Fatalf("baseline metadata = %#v, want no execution_transform", baselineMetadata)
	}

	fixedBody, fixedMetadata := runPassiveGPTEmulator(t, executionTransformTestConfig(t, false))
	if !strings.Contains(string(fixedBody), `"type":"tool_use"`) || !strings.Contains(string(fixedBody), `"name":"Read"`) {
		t.Fatalf("transformed body = %s, want tool_use Read call", fixedBody)
	}
	transformMeta, ok := fixedMetadata["execution_transform"].(map[string]any)
	if !ok || transformMeta["applied"] != true || transformMeta["forced_tools"] != true {
		t.Fatalf("fixed metadata = %#v, want applied forced execution_transform", fixedMetadata)
	}
}

func TestApplyExecutionTransformOpenAIResponsesSameModelRule(t *testing.T) {
	cfg, err := decodeConfig([]byte(`enabled: true
execution_transform:
  enabled: false
  telemetry: true
rules:
  - name: omp_gpt_responses_execution
    source_formats: [openai-response]
    models: ["gpt-5.5"]
    primary_model: "$requested"
    fallback_models: ["$requested"]
    execution_transform:
      enabled: true
      activation: tool_surface
      force_tools: when_available
      source_formats: [openai-response]
      requested_model_patterns: ["gpt-*"]
      selected_model_patterns: ["gpt-*"]
      execution_envelope: "Execution mode is active. Direct OMP Responses turn must use a tool before prose."
      exit_tool:
        name: ExitContinuationTool
fallback:
  cooldown_seconds: 60
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	rule := cfg.Rules[0]
	exec := pluginapi.ExecutorRequest{
		Model:           "gpt-5.5",
		SourceFormat:    "openai-response",
		OriginalRequest: []byte(`{"model":"gpt-5.5","instructions":"You are OMP.","input":[{"role":"user","content":[{"type":"input_text","text":"PM state now: PATCH_DEVELOPER_LOOP_REQUIRED. Write the handoff."}]}],"tools":[{"type":"function","name":"read","parameters":{"type":"object","properties":{}}},{"type":"function","name":"write","parameters":{"type":"object","properties":{}}}],"stream":true}`),
	}

	body := requestBodyForModel(exec.OriginalRequest, "gpt-5.5")
	got, metadata, _, err := applyExecutionTransform(cfg, exec, rule, "gpt-5.5", "openai-response", body)
	if err != nil {
		t.Fatalf("applyExecutionTransform() error = %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode transformed body: %v; body=%s", err, got)
	}
	instructions := asString(obj["instructions"])
	if !strings.Contains(instructions, "Direct OMP Responses turn must use a tool before prose") {
		t.Fatalf("instructions = %q, want execution envelope", instructions)
	}
	if obj["tool_choice"] != "required" {
		t.Fatalf("tool_choice = %#v, want required", obj["tool_choice"])
	}
	if !hasToolNamed(obj["tools"], "ExitContinuationTool") {
		t.Fatalf("tools = %#v, want ExitContinuationTool", obj["tools"])
	}
	transformMeta, ok := metadata["execution_transform"].(map[string]any)
	if !ok || transformMeta["applied"] != true || transformMeta["requested_model"] != "gpt-5.5" || transformMeta["selected_model"] != "gpt-5.5" {
		t.Fatalf("metadata = %#v, want same-model applied execution_transform", metadata)
	}
}

func runPassiveGPTEmulator(t *testing.T, cfg pluginConfig) ([]byte, map[string]any) {
	t.Helper()
	currentConfig.Store(cfg)
	primaryCooldowns = newPrimaryCooldownStore(func() time.Time {
		return time.Date(2026, 6, 29, 13, 0, 0, 0, time.UTC)
	})
	executeHostModelAttempt = func(_ pluginapi.ExecutorRequest, _ string, model, _, _ string, body []byte) (pluginapi.HostModelExecutionResponse, error) {
		if model == "claude-sonnet-4-5" {
			return pluginapi.HostModelExecutionResponse{}, errors.New("auth_unavailable: no auth available")
		}
		bodyText := string(body)
		if strings.Contains(bodyText, "Execution mode is active") &&
			strings.Contains(bodyText, "ExitContinuationTool") &&
			strings.Contains(bodyText, `"tool_choice"`) {
			return pluginapi.HostModelExecutionResponse{StatusCode: http.StatusOK, Body: []byte(`{"content":[{"type":"tool_use","name":"Read","input":{"path":".agent/CONTINUITY.md"}}]}`)}, nil
		}
		return pluginapi.HostModelExecutionResponse{StatusCode: http.StatusOK, Body: []byte(`{"content":[{"type":"text","text":"I would inspect the saved session state first, then decide what to do next."}]}`)}, nil
	}

	body, _, metadata, err := runExecutionFallback(executionTransformExecutorRequest(), "callback-1")
	if err != nil {
		t.Fatalf("runExecutionFallback() error = %v", err)
	}
	return body, metadata
}

func executionTransformDisabledTestConfig(t *testing.T) pluginConfig {
	t.Helper()
	cfg, err := decodeConfig([]byte(`enabled: true
rules:
  - name: claude_code
    source_formats: [claude]
    models: ["claude-*"]
    fallback_models: [gpt-5.5]
fallback:
  cooldown_seconds: 60
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	return cfg
}
func executionTransformTestConfig(t *testing.T, dryRun bool) pluginConfig {
	t.Helper()
	dryRunYAML := "false"
	if dryRun {
		dryRunYAML = "true"
	}
	cfg, err := decodeConfig([]byte(`enabled: true
execution_transform:
  enabled: false
  telemetry: true
rules:
  - name: claude_code
    source_formats: [claude]
    models: ["claude-*"]
    fallback_models: [gpt-5.5]
    execution_transform:
      enabled: true
      dry_run: ` + dryRunYAML + `
      activation: tool_surface
      force_tools: when_available
      execution_envelope: "Execution mode is active. Custom execution envelope: perform a real state-changing tool call before prose."
      inject_execution_envelope: true
      add_audited_exit_tool: true
      selected_model_patterns: ["gpt-*"]
      requested_model_patterns: ["claude-*"]
      source_formats: [claude]
      exit_tool:
        name: ExitContinuationTool
fallback:
  cooldown_seconds: 60
`))
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	return cfg
}

func executionTransformExecutorRequest() pluginapi.ExecutorRequest {
	return pluginapi.ExecutorRequest{
		Model:           "claude-sonnet-4-5",
		SourceFormat:    "claude",
		OriginalRequest: []byte(`{"model":"claude-sonnet-4-5","system":"You are Claude Code.","tools":[{"name":"Read","input_schema":{"type":"object","properties":{}}}],"messages":[{"role":"user","content":"$nvmd-platform:session --load\nContinue from the saved session state."}]}`),
	}
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return value.(string)
}

func hasToolNamed(value any, name string) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if obj["name"] == name {
			return true
		}
		function, _ := obj["function"].(map[string]any)
		if function["name"] == name {
			return true
		}
	}
	return false
}
