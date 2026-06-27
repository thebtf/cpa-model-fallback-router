package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func routeModel(raw []byte) ([]byte, error) {
	var req rpcModelRouteRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	cfg := loadedConfig()
	if _, ok := matchingRule(cfg, req.SourceFormat, req.RequestedModel); !ok {
		return okEnvelope(pluginapi.ModelRouteResponse{Handled: false})
	}
	return okEnvelope(pluginapi.ModelRouteResponse{
		Handled:    true,
		TargetKind: pluginapi.ModelRouteTargetSelf,
		Reason:     pluginIdentifier + ":matched",
	})
}

func execute(raw []byte) ([]byte, error) {
	var req rpcExecutorRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	payload, headers, metadata, errRun := runExecutionFallback(req.ExecutorRequest, req.HostCallbackID)
	if errRun != nil {
		return errorEnvelopeWithStatus("executor_error", errRun.Error(), statusOrDefault(statusFromError(errRun))), nil
	}
	return okEnvelope(pluginapi.ExecutorResponse{Payload: payload, Headers: headers, Metadata: metadata})
}

func runExecutionFallback(exec pluginapi.ExecutorRequest, hostCallbackID string) ([]byte, http.Header, map[string]any, error) {
	cfg := loadedConfig()
	reqModel := strings.TrimSpace(exec.Model)
	rule, ok := matchingRule(cfg, executionSourceFormat(exec), reqModel)
	if !ok {
		return nil, nil, nil, statusError{status: http.StatusBadGateway, message: "no fallback rule matched executor request"}
	}
	attempts := buildAttempts(rule, reqModel)
	if len(attempts) == 0 {
		return nil, nil, nil, statusError{status: http.StatusBadGateway, message: "fallback rule produced no model attempts"}
	}
	policy := fallbackPolicy(cfg, rule)

	var lastErr error
	for index, model := range attempts {
		body := requestBodyForModel(requestBody(exec), model)
		resp, errExecute := executeHostModel(exec, hostCallbackID, model, body)
		status := responseStatus(resp.StatusCode, errExecute)
		if errExecute == nil && successStatus(status) {
			return resp.Body, cloneHeader(resp.Headers), attemptMetadata(rule, attempts, model, index), nil
		}
		if errExecute == nil {
			errExecute = statusError{status: status, message: fmt.Sprintf("host model %s returned status %d", model, status)}
		}
		lastErr = errExecute
		if index == len(attempts)-1 || !shouldFallback(status, errExecute, policy) {
			return nil, nil, nil, errExecute
		}
	}
	if lastErr != nil {
		return nil, nil, nil, lastErr
	}
	return nil, nil, nil, statusError{status: http.StatusBadGateway, message: "fallback execution failed"}
}

func attemptMetadata(rule fallbackRule, attempts []string, selected string, index int) map[string]any {
	return map[string]any{
		"fallback_rule":    rule.Name,
		"attempts":         append([]string(nil), attempts...),
		"selected_model":   selected,
		"fallback_used":    index > 0,
		"selected_attempt": index,
	}
}

func executionSourceFormat(exec pluginapi.ExecutorRequest) string {
	return firstNonEmpty(exec.SourceFormat, exec.Format)
}

func hostProtocol(exec pluginapi.ExecutorRequest) string {
	protocol := normalizeProtocol(executionSourceFormat(exec))
	if protocol == "" {
		return "openai"
	}
	return protocol
}

func responseStatus(status int, err error) int {
	if status > 0 {
		return status
	}
	if errStatus := statusFromError(err); errStatus > 0 {
		return errStatus
	}
	if err == nil {
		return http.StatusOK
	}
	return 0
}
