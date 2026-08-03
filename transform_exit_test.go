package main

import (
	"encoding/json"
	"testing"
)

func TestUnwrapClaudeAuditedExitResponse(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	settings := resolveExecutionTransform(cfg, cfg.Rules[0])
	body := []byte(`{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"ExitContinuationTool","input":{"status":"no_applicable_tool","reason":"No matching state action exists.","user_response":"I need a user decision before continuing."}}]}`)

	got, metadata, ok := unwrapAuditedExitResponse(settings, "claude", body)
	if !ok {
		t.Fatal("unwrapAuditedExitResponse() ok = false, want true")
	}
	var obj map[string]any
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode unwrapped body: %v; body=%s", err, got)
	}
	content := obj["content"].([]any)
	text := content[0].(map[string]any)
	if text["type"] != "text" || text["text"] != "I need a user decision before continuing." {
		t.Fatalf("content = %#v, want assistant text from user_response", content)
	}
	exitMeta := metadata["audited_exit"].(map[string]any)
	if exitMeta["unwrapped"] != true || exitMeta["status"] != "no_applicable_tool" {
		t.Fatalf("audited_exit metadata = %#v", exitMeta)
	}
}

func TestUnwrapOpenAIChatAuditedExitResponse(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	settings := resolveExecutionTransform(cfg, cfg.Rules[0])
	body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"ExitContinuationTool","arguments":"{\"status\":\"blocked\",\"reason\":\"Need approval.\",\"user_response\":\"Please approve the next action.\"}"}}]},"finish_reason":"tool_calls"}]}`)

	got, metadata, ok := unwrapAuditedExitResponse(settings, "openai", body)
	if !ok {
		t.Fatal("unwrapAuditedExitResponse() ok = false, want true")
	}
	var obj map[string]any
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("decode unwrapped body: %v; body=%s", err, got)
	}
	choice := obj["choices"].([]any)[0].(map[string]any)
	message := choice["message"].(map[string]any)
	if message["content"] != "Please approve the next action." {
		t.Fatalf("message = %#v, want content from user_response", message)
	}
	if _, exists := message["tool_calls"]; exists {
		t.Fatalf("message still has tool_calls after unwrap: %#v", message)
	}
	exitMeta := metadata["audited_exit"].(map[string]any)
	if exitMeta["unwrapped"] != true || exitMeta["status"] != "blocked" {
		t.Fatalf("audited_exit metadata = %#v", exitMeta)
	}
}

func TestUnwrapOpenAIChatInvalidAuditedExitReportsMetadata(t *testing.T) {
	cfg := executionTransformTestConfig(t, false)
	settings := resolveExecutionTransform(cfg, cfg.Rules[0])
	body := []byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"type":"function","function":{"name":"ExitContinuationTool","arguments":"{\"status\":\"blocked\",\"reason\":\"Need approval.\"}"}}]}}]}`)

	got, metadata, ok := unwrapAuditedExitResponse(settings, "openai", body)
	if ok || got != nil {
		t.Fatalf("unwrapAuditedExitResponse invalid = (%s, %v), want pass-through", got, ok)
	}
	exitMeta := metadata["audited_exit"].(map[string]any)
	if exitMeta["invalid"] != true || exitMeta["unwrapped"] != false {
		t.Fatalf("invalid audited_exit metadata = %#v", exitMeta)
	}
}

func TestUnwrapAuditedExitResponseDisabledByDefault(t *testing.T) {
	settings := resolveExecutionTransform(defaultPluginConfig(), fallbackRule{})
	body := []byte(`{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"ExitContinuationTool","input":{"status":"no_applicable_tool","reason":"No matching state action exists.","user_response":"I need a user decision before continuing."}}]}`)

	got, metadata, ok := unwrapAuditedExitResponse(settings, "claude", body)
	if ok || got != nil || metadata != nil {
		t.Fatalf("unwrapAuditedExitResponse disabled = (%s, %#v, %v), want untouched", got, metadata, ok)
	}
}
