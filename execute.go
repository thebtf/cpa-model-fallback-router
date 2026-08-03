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
		TargetKind: pluginapi.ModelRouteTargetExecutor,
		Target:     pluginIdentifier,
		Reason:     pluginIdentifier + ":matched",
	})
}

var executeHostModelAttempt = executeHostModel

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
	policy := fallbackPolicy(cfg, rule)
	primary := resolveModelToken(rule.PrimaryModel, reqModel)
	cooldownKey := fallbackCooldownKey(executionSourceFormat(exec), rule, primary)
	_, primarySkipped := primaryCooldowns.active(cooldownKey)
	plan := buildAttemptPlan(rule, reqModel, primarySkipped)
	attempts := plan.Attempts
	if len(attempts) == 0 {
		return nil, nil, nil, statusError{status: http.StatusBadGateway, message: "fallback rule produced no model attempts"}
	}

	var lastErr error
	bodyInfo := requestBodyInfo(exec)
	for index, model := range attempts {
		body := requestBodyForModel(bodyInfo.Body, model)
		body, transformMetadata, unwrapAuditedExit, errTransform := applyExecutionTransform(cfg, exec, rule, model, bodyInfo.EntryProtocol, body)
		if errTransform != nil {
			return nil, nil, transformMetadata, errTransform
		}
		resp, errExecute := executeHostModelAttempt(exec, hostCallbackID, model, bodyInfo.EntryProtocol, bodyInfo.ResponseProtocol, body)
		status := responseStatus(resp.StatusCode, errExecute)
		if errExecute == nil && successStatus(status) {
			metadata := attemptMetadata(rule, attempts, model, index, plan.PrimarySkipped)
			mergeMetadata(metadata, transformMetadata)
			payload := resp.Body
			if unwrapAuditedExit {
				unwrapped, exitMetadata, okUnwrap := unwrapAuditedExitResponse(resolveExecutionTransform(cfg, rule), bodyInfo.ResponseProtocol, resp.Body)
				mergeMetadata(metadata, exitMetadata)
				if okUnwrap {
					payload = unwrapped
				}
			}
			return payload, cloneHeader(resp.Headers), metadata, nil
		}
		if errExecute == nil {
			errExecute = hostModelStatusError(model, status, resp.Body)
		}
		lastErr = errExecute
		fallbackAllowed := shouldFallback(status, errExecute, policy)
		if fallbackAllowed && strings.EqualFold(model, plan.Primary) {
			primaryCooldowns.mark(cooldownKey, fallbackCooldownDuration(policy))
		}
		if index == len(attempts)-1 || !fallbackAllowed {
			return nil, nil, nil, errExecute
		}
	}
	if lastErr != nil {
		return nil, nil, nil, lastErr
	}
	return nil, nil, nil, statusError{status: http.StatusBadGateway, message: "fallback execution failed"}
}

func hostModelStatusError(model string, status int, body []byte) error {
	message := fmt.Sprintf("host model %s returned status %d", model, status)
	if summary := hostModelErrorSummary(body); summary != "" {
		message += ": " + summary
	}
	return statusError{status: status, message: message}
}

func hostModelErrorSummary(body []byte) string {
	summary := strings.TrimSpace(string(body))
	if summary == "" {
		return ""
	}
	if len(summary) > 512 {
		summary = summary[:512]
	}
	return summary
}

func attemptMetadata(rule fallbackRule, attempts []string, selected string, index int, primarySkipped bool) map[string]any {
	metadata := map[string]any{
		"fallback_rule":    rule.Name,
		"attempts":         append([]string(nil), attempts...),
		"selected_model":   selected,
		"fallback_used":    primarySkipped || index > 0,
		"selected_attempt": index,
	}
	if primarySkipped {
		metadata["primary_cooldown_skipped"] = true
	}
	return metadata
}

func mergeMetadata(dst map[string]any, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for key, value := range src {
		dst[key] = value
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
