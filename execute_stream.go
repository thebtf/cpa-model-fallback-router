package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type streamOrchestrationRunner func(context.Context, pluginapi.ExecutorRequest, string, string) error

type pluginStreamCloser func(string, string)

func executeStream(raw []byte) ([]byte, error) {
	var req rpcExecutorRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	return startExecutorStream(req, runExecutionFallbackStream, closePluginStream)
}

func startExecutorStream(req rpcExecutorRequest, runner streamOrchestrationRunner, closeStream pluginStreamCloser) ([]byte, error) {
	streamID := strings.TrimSpace(req.StreamID)
	if streamID == "" {
		return errorEnvelope("executor_error", "stream_id is required for executor.execute_stream"), nil
	}
	if runner == nil {
		return errorEnvelope("executor_error", "stream orchestration runner is unavailable"), nil
	}
	if closeStream == nil {
		closeStream = func(string, string) {}
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				closeStream(streamID, fmt.Sprintf("stream orchestration panic: %v", recovered))
			}
		}()
		errRun := runner(context.Background(), req.ExecutorRequest, req.HostCallbackID, streamID)
		if errRun != nil {
			closeStream(streamID, errRun.Error())
			return
		}
		closeStream(streamID, "")
	}()
	return okEnvelope(map[string]any{
		"headers": http.Header{"Content-Type": []string{"text/event-stream"}},
	})
}

func runExecutionFallbackStream(_ context.Context, exec pluginapi.ExecutorRequest, hostCallbackID, pluginStreamID string) error {
	cfg := loadedConfig()
	reqModel := strings.TrimSpace(exec.Model)
	rule, ok := matchingRule(cfg, executionSourceFormat(exec), reqModel)
	if !ok {
		return statusError{status: http.StatusBadGateway, message: "no fallback rule matched executor stream request"}
	}
	policy := fallbackPolicy(cfg, rule)
	primary := resolveModelToken(rule.PrimaryModel, reqModel)
	cooldownKey := fallbackCooldownKey(executionSourceFormat(exec), rule, primary)
	_, primarySkipped := primaryCooldowns.active(cooldownKey)
	plan := buildAttemptPlan(rule, reqModel, primarySkipped)
	attempts := plan.Attempts
	if len(attempts) == 0 {
		return statusError{status: http.StatusBadGateway, message: "fallback rule produced no stream model attempts"}
	}

	var lastErr error
	for index, model := range attempts {
		body := requestBodyForModel(requestBody(exec), model)
		status, emitted, errForward := forwardHostModelStream(exec, hostCallbackID, model, body, pluginStreamID)
		if errForward == nil && successStatus(responseStatus(status, nil)) {
			return nil
		}
		if errForward == nil {
			errForward = statusError{status: status, message: fmt.Sprintf("host model %s stream returned status %d", model, status)}
		}
		lastErr = errForward
		fallbackAllowed := shouldFallback(responseStatus(status, errForward), errForward, policy)
		if fallbackAllowed && strings.EqualFold(model, plan.Primary) {
			primaryCooldowns.mark(cooldownKey, fallbackCooldownDuration(policy))
		}
		if emitted || index == len(attempts)-1 || !fallbackAllowed {
			return errForward
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return statusError{status: http.StatusBadGateway, message: "fallback stream execution failed"}
}

func forwardHostModelStream(exec pluginapi.ExecutorRequest, hostCallbackID, model string, body []byte, pluginStreamID string) (int, bool, error) {
	resp, errStart := startHostModelStream(exec, hostCallbackID, model, body)
	if errStart != nil {
		return responseStatus(0, errStart), false, errStart
	}
	if resp.StatusCode >= 400 {
		_ = closeHostModelStream(resp.StreamID)
		return resp.StatusCode, false, statusError{status: resp.StatusCode, message: fmt.Sprintf("host model %s stream returned status %d", model, resp.StatusCode)}
	}
	if strings.TrimSpace(resp.StreamID) == "" {
		return 0, false, fmt.Errorf("host model stream: empty stream_id")
	}
	defer func() { _ = closeHostModelStream(resp.StreamID) }()

	emitted := false
	for {
		chunk, errRead := readHostModelStream(resp.StreamID)
		if errRead != nil {
			return responseStatus(0, errRead), emitted, errRead
		}
		if chunk.Error != "" {
			return responseStatus(0, fmt.Errorf("%s", chunk.Error)), emitted, statusError{status: statusFromError(fmt.Errorf("%s", chunk.Error)), message: chunk.Error}
		}
		if len(chunk.Payload) > 0 {
			emitted = true
			if errEmit := emitPluginStreamChunk(pluginStreamID, chunk.Payload); errEmit != nil {
				return 0, true, errEmit
			}
		}
		if chunk.Done {
			return http.StatusOK, emitted, nil
		}
	}
}
