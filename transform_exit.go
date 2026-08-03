package main

import (
	"encoding/json"
	"strings"
)

type auditedExitArguments struct {
	Status              string   `json:"status"`
	StateSourcesChecked []string `json:"state_sources_checked"`
	ActionsAttempted    []string `json:"actions_attempted"`
	Reason              string   `json:"reason"`
	NextOwner           string   `json:"next_owner"`
	RescheduleOrCancel  string   `json:"reschedule_or_cancel"`
	UserResponse        string   `json:"user_response"`
}

func unwrapAuditedExitResponse(settings resolvedExecutionTransform, protocol string, body []byte) ([]byte, map[string]any, bool) {
	if !settings.Enabled || strings.TrimSpace(settings.ExitToolName) == "" || len(body) == 0 {
		return nil, nil, false
	}
	switch normalizeProtocol(protocol) {
	case "claude":
		return unwrapClaudeAuditedExitResponse(settings, body)
	case "openai":
		return unwrapOpenAIChatAuditedExitResponse(settings, body)
	default:
		return nil, nil, false
	}
}

func unwrapClaudeAuditedExitResponse(settings resolvedExecutionTransform, body []byte) ([]byte, map[string]any, bool) {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, nil, false
	}
	content, ok := obj["content"].([]any)
	if !ok || len(content) != 1 {
		return nil, nil, false
	}
	block, ok := content[0].(map[string]any)
	if !ok || block["type"] != "tool_use" || !strings.EqualFold(asMapString(block, "name"), settings.ExitToolName) {
		return nil, nil, false
	}
	args := auditedExitArguments{}
	input, ok := block["input"].(map[string]any)
	if !ok {
		return nil, nil, false
	}
	rawInput, err := json.Marshal(input)
	if err != nil || json.Unmarshal(rawInput, &args) != nil || !validAuditedExitArguments(args) {
		return nil, auditedExitInvalidMetadata(args.Status), false
	}
	obj["content"] = []any{map[string]any{"type": "text", "text": args.UserResponse}}
	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, nil, false
	}
	return raw, auditedExitMetadata(args), true
}

func unwrapOpenAIChatAuditedExitResponse(settings resolvedExecutionTransform, body []byte) ([]byte, map[string]any, bool) {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, nil, false
	}
	choices, ok := obj["choices"].([]any)
	if !ok || len(choices) != 1 {
		return nil, nil, false
	}
	choice, ok := choices[0].(map[string]any)
	if !ok {
		return nil, nil, false
	}
	message, ok := choice["message"].(map[string]any)
	if !ok {
		return nil, nil, false
	}
	toolCalls, ok := message["tool_calls"].([]any)
	if !ok || len(toolCalls) != 1 {
		return nil, nil, false
	}
	call, ok := toolCalls[0].(map[string]any)
	if !ok || call["type"] != "function" {
		return nil, nil, false
	}
	function, ok := call["function"].(map[string]any)
	if !ok || !strings.EqualFold(asMapString(function, "name"), settings.ExitToolName) {
		return nil, nil, false
	}
	args := auditedExitArguments{}
	if err := json.Unmarshal([]byte(asMapString(function, "arguments")), &args); err != nil || !validAuditedExitArguments(args) {
		return nil, auditedExitInvalidMetadata(args.Status), false
	}
	message["content"] = args.UserResponse
	delete(message, "tool_calls")
	choice["finish_reason"] = "stop"
	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, nil, false
	}
	return raw, auditedExitMetadata(args), true
}

func validAuditedExitArguments(args auditedExitArguments) bool {
	if strings.TrimSpace(args.Status) == "" || strings.TrimSpace(args.Reason) == "" || strings.TrimSpace(args.UserResponse) == "" {
		return false
	}
	if args.Status == "done_verified" && len(args.StateSourcesChecked) == 0 && len(args.ActionsAttempted) == 0 {
		return false
	}
	return true
}

func auditedExitMetadata(args auditedExitArguments) map[string]any {
	entry := map[string]any{
		"unwrapped": true,
		"status":    strings.TrimSpace(args.Status),
	}
	if len(args.StateSourcesChecked) > 0 {
		entry["state_sources_checked"] = append([]string(nil), args.StateSourcesChecked...)
	}
	if len(args.ActionsAttempted) > 0 {
		entry["actions_attempted"] = append([]string(nil), args.ActionsAttempted...)
	}
	if strings.TrimSpace(args.NextOwner) != "" {
		entry["next_owner"] = strings.TrimSpace(args.NextOwner)
	}
	return map[string]any{"audited_exit": entry}
}

func auditedExitInvalidMetadata(status string) map[string]any {
	return map[string]any{"audited_exit": map[string]any{"unwrapped": false, "invalid": true, "status": strings.TrimSpace(status)}}
}

func asMapString(obj map[string]any, key string) string {
	value, _ := obj[key].(string)
	return value
}
