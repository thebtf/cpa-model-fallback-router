package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func requestBodyForModel(body []byte, model string) []byte {
	model = strings.TrimSpace(model)
	if len(bytes.TrimSpace(body)) == 0 || model == "" {
		return bytes.Clone(body)
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return bytes.Clone(body)
	}
	if _, ok := obj["model"]; !ok {
		return bytes.Clone(body)
	}
	obj["model"] = model
	next, err := json.Marshal(obj)
	if err != nil {
		return bytes.Clone(body)
	}
	return next
}

type executionRequestBody struct {
	Body             []byte
	EntryProtocol    string
	ResponseProtocol string
}

func requestBody(exec pluginapi.ExecutorRequest) []byte {
	return requestBodyInfo(exec).Body
}

func requestBodyInfo(exec pluginapi.ExecutorRequest) executionRequestBody {
	sourceProtocol := normalizeProtocol(executionSourceFormat(exec))
	formatProtocol := normalizeProtocol(exec.Format)
	if len(exec.OriginalRequest) > 0 {
		entry := firstNonEmpty(sourceProtocol, inferRequestProtocol(exec.OriginalRequest), formatProtocol, "openai")
		return executionRequestBody{
			Body:             bytes.Clone(exec.OriginalRequest),
			EntryProtocol:    entry,
			ResponseProtocol: firstNonEmpty(sourceProtocol, entry),
		}
	}
	if len(exec.Payload) > 0 {
		entry := firstNonEmpty(formatProtocol, inferRequestProtocol(exec.Payload), sourceProtocol, "openai")
		return executionRequestBody{
			Body:             bytes.Clone(exec.Payload),
			EntryProtocol:    entry,
			ResponseProtocol: firstNonEmpty(sourceProtocol, entry),
		}
	}
	entry := firstNonEmpty(sourceProtocol, formatProtocol, "openai")
	return executionRequestBody{
		EntryProtocol:    entry,
		ResponseProtocol: firstNonEmpty(sourceProtocol, entry),
	}
}

func inferRequestProtocol(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}
	if _, ok := obj["input"]; ok {
		return "openai-response"
	}
	if _, ok := obj["instructions"]; ok {
		return "openai-response"
	}
	if _, ok := obj["system"]; ok {
		return "claude"
	}
	if tools, ok := obj["tools"].([]any); ok {
		for _, tool := range tools {
			toolObj, ok := tool.(map[string]any)
			if !ok {
				continue
			}
			if _, ok := toolObj["input_schema"]; ok {
				return "claude"
			}
			if toolObj["type"] == "function" {
				return "openai"
			}
		}
	}
	if _, ok := obj["messages"]; ok {
		return "openai"
	}
	return ""
}

func cloneHeader(headers http.Header) http.Header {
	if headers == nil {
		return nil
	}
	cloned := make(http.Header, len(headers))
	for key, values := range headers {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	cloned := make(url.Values, len(values))
	for key, items := range values {
		cloned[key] = append([]string(nil), items...)
	}
	return cloned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
