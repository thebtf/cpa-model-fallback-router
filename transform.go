package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const defaultExecutionEnvelope = "Execution mode is active for an agent harness turn routed to a non-native model. This is not a review or planning-only turn. Before answering, execute the first applicable state-advancing action with the available tools: read state, write/update state, call the requested tool, verify evidence, create/update a handoff, route the next owner, or cancel/reschedule. Do not answer with prose such as 'I will do it', 'next step', 'currently doing', 'continuing', or 'still monitoring' unless you have already performed the action in this turn. If no tool or state-changing action applies, call ExitContinuationTool with checked sources, attempted actions, and the blocker reason."

type executionTransformDecision struct {
	Candidate      bool
	Reason         string
	BypassedReason string
	ToolMode       bool
}

func applyExecutionTransform(cfg pluginConfig, exec pluginapi.ExecutorRequest, rule fallbackRule, selectedModel, entryProtocol string, body []byte) ([]byte, map[string]any, bool, error) {
	settings := resolveExecutionTransform(cfg, rule)
	if !settings.Enabled {
		return bytes.Clone(body), nil, false, nil
	}
	entryProtocol = normalizeProtocol(entryProtocol)
	decision := detectExecutionTransform(settings, exec, selectedModel, entryProtocol, body)
	metadata := executionTransformMetadata(settings, exec, selectedModel, entryProtocol, decision, false, false)
	if !decision.Candidate {
		return bytes.Clone(body), metadata, false, nil
	}
	if settings.DryRun {
		metadata = executionTransformMetadata(settings, exec, selectedModel, entryProtocol, decision, false, false)
		return bytes.Clone(body), metadata, false, nil
	}

	mutated, exitToolAdded, err := transformExecutionBody(settings, entryProtocol, decision, body)
	if err != nil {
		if settings.Strict {
			return nil, metadata, false, err
		}
		decision.Candidate = false
		decision.BypassedReason = "transform_error:" + err.Error()
		return bytes.Clone(body), executionTransformMetadata(settings, exec, selectedModel, entryProtocol, decision, false, false), false, nil
	}
	return mutated, executionTransformMetadata(settings, exec, selectedModel, entryProtocol, decision, true, exitToolAdded), exitToolAdded, nil
}

func detectExecutionTransform(settings resolvedExecutionTransform, exec pluginapi.ExecutorRequest, selectedModel, entryProtocol string, body []byte) executionTransformDecision {
	source := normalizeProtocol(entryProtocol)
	if len(settings.SourceFormats) > 0 && !stringInList(source, settings.SourceFormats) {
		return executionTransformDecision{BypassedReason: "source_format_not_allowed"}
	}
	if !matchesAnyPattern(exec.Model, settings.RequestedModelPatterns) {
		return executionTransformDecision{BypassedReason: "requested_model_not_allowed"}
	}
	if !matchesAnyPattern(selectedModel, settings.SelectedModelPatterns) {
		return executionTransformDecision{BypassedReason: "selected_model_not_allowed"}
	}

	obj, err := decodeJSONObject(body)
	if err != nil {
		return executionTransformDecision{BypassedReason: "invalid_json"}
	}
	if !recognizableRequestShapeForProtocol(source, obj) {
		return executionTransformDecision{BypassedReason: "unknown_protocol_shape"}
	}
	if requestIsPostToolContinuation(source, obj) {
		return executionTransformDecision{BypassedReason: "post_tool_continuation"}
	}
	toolsAvailable := requestHasTools(obj)
	if !toolsAvailable && settings.ForceTools == executionTransformForceToolsWhenAvailable {
		return executionTransformDecision{BypassedReason: "no_tools_available"}
	}

	reason := ""
	switch settings.Activation {
	case executionTransformActivationToolSurface:
		if !toolsAvailable {
			return executionTransformDecision{BypassedReason: "no_tool_surface"}
		}
		reason = "activation:tool_surface"
	case executionTransformActivationAlways:
		reason = "activation:always"
	default:
		return executionTransformDecision{BypassedReason: "invalid_activation"}
	}

	toolMode := settings.ForceTools != executionTransformForceToolsNever && (toolsAvailable || settings.ForceTools == executionTransformForceToolsAlwaysIfSupported)
	return executionTransformDecision{Candidate: true, Reason: reason, ToolMode: toolMode}
}

func transformExecutionBody(settings resolvedExecutionTransform, protocol string, decision executionTransformDecision, body []byte) ([]byte, bool, error) {
	obj, err := decodeJSONObject(body)
	if err != nil {
		return nil, false, err
	}
	envelope := executionEnvelope(settings)
	switch protocol {
	case "claude":
		return transformClaudeExecutionBody(settings, decision, obj, envelope)
	case "openai":
		return transformOpenAIChatExecutionBody(settings, decision, obj, envelope)
	case "openai-response":
		return transformOpenAIResponsesExecutionBody(settings, decision, obj, envelope)
	default:
		return nil, false, fmt.Errorf("unknown protocol %q", protocol)
	}
}

func transformClaudeExecutionBody(settings resolvedExecutionTransform, decision executionTransformDecision, obj map[string]any, envelope string) ([]byte, bool, error) {
	if settings.InjectExecutionEnvelope {
		appendClaudeSystemEnvelope(obj, envelope)
	}
	exitToolAdded := false
	if settings.AddAuditedExitTool {
		tools, ok := obj["tools"].([]any)
		if ok || settings.ForceTools == executionTransformForceToolsAlwaysIfSupported {
			if !ok {
				tools = []any{}
			}
			if !toolListHasName(tools, settings.ExitToolName) {
				tools = append(tools, claudeAuditedExitTool(settings.ExitToolName))
				exitToolAdded = true
			}
			obj["tools"] = tools
		}
	}
	if decision.ToolMode && requestHasTools(obj) {
		obj["tool_choice"] = map[string]any{"type": "any"}
	}
	raw, err := json.Marshal(obj)
	return raw, exitToolAdded, err
}

func transformOpenAIChatExecutionBody(settings resolvedExecutionTransform, decision executionTransformDecision, obj map[string]any, envelope string) ([]byte, bool, error) {
	if settings.InjectExecutionEnvelope {
		messages, _ := obj["messages"].([]any)
		if !messagesContainText(messages, envelope) {
			messages = append([]any{map[string]any{"role": "system", "content": envelope}}, messages...)
		}
		obj["messages"] = messages
	}
	exitToolAdded := false
	if settings.AddAuditedExitTool {
		tools, ok := obj["tools"].([]any)
		if ok || settings.ForceTools == executionTransformForceToolsAlwaysIfSupported {
			if !ok {
				tools = []any{}
			}
			if !toolListHasName(tools, settings.ExitToolName) {
				tools = append(tools, openAIChatAuditedExitTool(settings.ExitToolName))
				exitToolAdded = true
			}
			obj["tools"] = tools
		}
	}
	if decision.ToolMode && requestHasTools(obj) {
		obj["tool_choice"] = "required"
	}
	raw, err := json.Marshal(obj)
	return raw, exitToolAdded, err
}

func transformOpenAIResponsesExecutionBody(settings resolvedExecutionTransform, decision executionTransformDecision, obj map[string]any, envelope string) ([]byte, bool, error) {
	if settings.InjectExecutionEnvelope {
		if instructions, ok := obj["instructions"].(string); ok && strings.TrimSpace(instructions) != "" {
			if !strings.Contains(instructions, envelope) {
				obj["instructions"] = instructions + "\n\n" + envelope
			}
		} else {
			obj["instructions"] = envelope
		}
	}
	exitToolAdded := false
	if settings.AddAuditedExitTool {
		tools, ok := obj["tools"].([]any)
		if ok || settings.ForceTools == executionTransformForceToolsAlwaysIfSupported {
			if !ok {
				tools = []any{}
			}
			if !toolListHasName(tools, settings.ExitToolName) {
				tools = append(tools, openAIResponsesAuditedExitTool(settings.ExitToolName))
				exitToolAdded = true
			}
			obj["tools"] = tools
		}
	}
	if decision.ToolMode && requestHasTools(obj) {
		obj["tool_choice"] = "required"
	}
	raw, err := json.Marshal(obj)
	return raw, exitToolAdded, err
}

func appendClaudeSystemEnvelope(obj map[string]any, envelope string) {
	switch system := obj["system"].(type) {
	case string:
		if strings.Contains(system, envelope) {
			return
		}
		if strings.TrimSpace(system) == "" {
			obj["system"] = envelope
			return
		}
		obj["system"] = system + "\n\n" + envelope
	case []any:
		if arrayContainsText(system, envelope) {
			return
		}
		obj["system"] = append(system, map[string]any{"type": "text", "text": envelope})
	default:
		obj["system"] = envelope
	}
}

func executionTransformMetadata(settings resolvedExecutionTransform, exec pluginapi.ExecutorRequest, selectedModel, entryProtocol string, decision executionTransformDecision, applied bool, exitToolAdded bool) map[string]any {
	if !settings.Telemetry {
		return nil
	}
	entry := map[string]any{
		"candidate":       decision.Candidate,
		"applied":         applied,
		"dry_run":         settings.DryRun,
		"activation":      settings.Activation,
		"forced_tools":    applied && decision.ToolMode,
		"exit_tool_added": exitToolAdded,
		"requested_model": strings.TrimSpace(exec.Model),
		"selected_model":  strings.TrimSpace(selectedModel),
		"source_format":   normalizeProtocol(entryProtocol),
	}
	if decision.Reason != "" {
		entry["reason"] = decision.Reason
	}
	if decision.BypassedReason != "" {
		entry["bypassed_reason"] = decision.BypassedReason
	}
	return map[string]any{"execution_transform": entry}
}

func executionEnvelope(settings resolvedExecutionTransform) string {
	envelope := strings.TrimSpace(settings.ExecutionEnvelope)
	if envelope == "" {
		envelope = defaultExecutionEnvelope
	}
	if settings.ExitToolName != defaultAuditedExitToolName && settings.ExitToolName != "" {
		envelope = strings.ReplaceAll(envelope, defaultAuditedExitToolName, settings.ExitToolName)
	}
	if settings.MaxEnvelopeChars > 0 && len(envelope) > settings.MaxEnvelopeChars {
		return envelope[:settings.MaxEnvelopeChars]
	}
	return envelope
}

func decodeJSONObject(body []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("empty body")
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("body is not an object")
	}
	return obj, nil
}

func recognizableRequestShapeForProtocol(protocol string, obj map[string]any) bool {
	switch normalizeProtocol(protocol) {
	case "claude", "openai":
		_, hasMessages := obj["messages"]
		return hasMessages
	case "openai-response":
		_, hasInput := obj["input"]
		_, hasInstructions := obj["instructions"]
		return hasInput || hasInstructions
	default:
		_, hasMessages := obj["messages"]
		_, hasInput := obj["input"]
		_, hasSystem := obj["system"]
		_, hasInstructions := obj["instructions"]
		return hasMessages || hasInput || hasSystem || hasInstructions
	}
}

func requestHasTools(obj map[string]any) bool {
	tools, ok := obj["tools"].([]any)
	return ok && len(tools) > 0
}

func requestIsPostToolContinuation(protocol string, obj map[string]any) bool {
	switch normalizeProtocol(protocol) {
	case "claude":
		return claudePostToolContinuation(obj)
	case "openai":
		return openAIChatPostToolContinuation(obj)
	case "openai-response":
		return openAIResponsesPostToolContinuation(obj)
	default:
		return claudePostToolContinuation(obj) || openAIChatPostToolContinuation(obj) || openAIResponsesPostToolContinuation(obj)
	}
}

func claudePostToolContinuation(obj map[string]any) bool {
	messages, ok := obj["messages"].([]any)
	if !ok || len(messages) == 0 {
		return false
	}
	last, ok := messages[len(messages)-1].(map[string]any)
	if !ok {
		return false
	}
	if role, ok := last["role"].(string); ok && !strings.EqualFold(role, "user") {
		return false
	}
	return valueHasBlockType(last["content"], "tool_result")
}

func openAIChatPostToolContinuation(obj map[string]any) bool {
	messages, ok := obj["messages"].([]any)
	if !ok || len(messages) == 0 {
		return false
	}
	last, ok := messages[len(messages)-1].(map[string]any)
	if !ok {
		return false
	}
	role, _ := last["role"].(string)
	if strings.EqualFold(role, "tool") {
		return true
	}
	if _, hasToolCallID := last["tool_call_id"]; hasToolCallID {
		return true
	}
	return valueHasBlockType(last["content"], "tool_result", "function_call_output", "computer_call_output")
}

func openAIResponsesPostToolContinuation(obj map[string]any) bool {
	input, ok := obj["input"].([]any)
	if !ok || len(input) == 0 {
		return false
	}
	return valueHasBlockType(input[len(input)-1], "function_call_output", "tool_result", "computer_call_output", "mcp_call_output")
}

func valueHasBlockType(value any, blockTypes ...string) bool {
	wanted := make(map[string]struct{}, len(blockTypes))
	for _, blockType := range blockTypes {
		blockType = strings.ToLower(strings.TrimSpace(blockType))
		if blockType != "" {
			wanted[blockType] = struct{}{}
		}
	}
	return valueHasAnyBlockType(value, wanted)
}

func valueHasAnyBlockType(value any, wanted map[string]struct{}) bool {
	switch item := value.(type) {
	case []any:
		for _, child := range item {
			if valueHasAnyBlockType(child, wanted) {
				return true
			}
		}
	case map[string]any:
		if blockType, ok := item["type"].(string); ok {
			if _, found := wanted[strings.ToLower(strings.TrimSpace(blockType))]; found {
				return true
			}
		}
		if content, ok := item["content"]; ok {
			return valueHasAnyBlockType(content, wanted)
		}
	}
	return false
}

func extractRequestText(value any) string {
	var parts []string
	collectText(value, &parts)
	return strings.Join(parts, "\n")
}

func collectText(value any, parts *[]string) {
	switch item := value.(type) {
	case string:
		*parts = append(*parts, item)
	case []any:
		for _, child := range item {
			collectText(child, parts)
		}
	case map[string]any:
		for _, child := range item {
			collectText(child, parts)
		}
	}
}

func arrayContainsText(items []any, text string) bool {
	for _, item := range items {
		if strings.Contains(extractRequestText(item), text) {
			return true
		}
	}
	return false
}

func messagesContainText(items []any, text string) bool {
	return arrayContainsText(items, text)
}

func toolListHasName(tools []any, name string) bool {
	for _, tool := range tools {
		if strings.EqualFold(toolName(tool), name) {
			return true
		}
	}
	return false
}

func toolName(tool any) string {
	obj, ok := tool.(map[string]any)
	if !ok {
		return ""
	}
	if name, ok := obj["name"].(string); ok {
		return name
	}
	if function, ok := obj["function"].(map[string]any); ok {
		if name, ok := function["name"].(string); ok {
			return name
		}
	}
	return ""
}

func claudeAuditedExitTool(name string) map[string]any {
	return map[string]any{
		"name":         name,
		"description":  "Audited structured exit for a Claude Code execution turn when no state-advancing tool call applies.",
		"input_schema": auditedExitToolSchema(),
	}
}

func openAIChatAuditedExitTool(name string) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        name,
			"description": "Audited structured exit for a Claude Code execution turn when no state-advancing tool call applies.",
			"parameters":  auditedExitToolSchema(),
		},
	}
}

func openAIResponsesAuditedExitTool(name string) map[string]any {
	return map[string]any{
		"type":        "function",
		"name":        name,
		"description": "Audited structured exit for a Claude Code execution turn when no state-advancing tool call applies.",
		"parameters":  auditedExitToolSchema(),
	}
}

func auditedExitToolSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"status", "reason", "user_response"},
		"properties": map[string]any{
			"status": map[string]any{
				"type": "string",
				"enum": []any{"blocked", "not_operational_continue", "no_applicable_tool", "done_verified", "needs_user_decision", "reference_turn"},
			},
			"state_sources_checked": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"actions_attempted":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"reason":                map[string]any{"type": "string"},
			"next_owner": map[string]any{
				"type": "string",
				"enum": []any{"user", "pm", "developer", "same-session", "unknown"},
			},
			"reschedule_or_cancel": map[string]any{"type": "string"},
			"user_response":        map[string]any{"type": "string"},
		},
	}
}
