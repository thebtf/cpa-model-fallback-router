package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestRunExecutionFallbackMarksPrimaryCooldownOnAuthUnavailable(t *testing.T) {
	cfg := configureFallbackTest(t, 60)
	calls := make([]string, 0, 2)
	executeHostModelAttempt = func(_ pluginapi.ExecutorRequest, _ string, model string, _ []byte) (pluginapi.HostModelExecutionResponse, error) {
		calls = append(calls, model)
		if model == "claude-sonnet-4-5" {
			return pluginapi.HostModelExecutionResponse{}, errors.New("auth_unavailable: no auth available")
		}
		return pluginapi.HostModelExecutionResponse{StatusCode: http.StatusOK, Body: []byte(`{"ok":true}`)}, nil
	}

	body, _, metadata, err := runExecutionFallback(testExecutorRequest(), "callback-1")
	if err != nil {
		t.Fatalf("runExecutionFallback() error = %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %s, want ok payload", body)
	}
	if !reflect.DeepEqual(calls, []string{"claude-sonnet-4-5", "gpt-5.4"}) {
		t.Fatalf("calls = %#v, want primary then fallback", calls)
	}
	if metadata["fallback_used"] != true {
		t.Fatalf("fallback_used = %#v, want true", metadata["fallback_used"])
	}
	key := fallbackCooldownKey("claude", cfg.Rules[0], "claude-sonnet-4-5")
	if _, ok := primaryCooldowns.active(key); !ok {
		t.Fatal("primary cooldown was not marked after auth unavailable failure")
	}
}

func TestRunExecutionFallbackSkipsPrimaryDuringCooldown(t *testing.T) {
	cfg := configureFallbackTest(t, 60)
	key := fallbackCooldownKey("claude", cfg.Rules[0], "claude-sonnet-4-5")
	primaryCooldowns.mark(key, time.Minute)
	calls := make([]string, 0, 1)
	executeHostModelAttempt = func(_ pluginapi.ExecutorRequest, _ string, model string, _ []byte) (pluginapi.HostModelExecutionResponse, error) {
		calls = append(calls, model)
		if model == "claude-sonnet-4-5" {
			t.Fatal("primary model was called while cooldown was active")
		}
		return pluginapi.HostModelExecutionResponse{StatusCode: http.StatusOK, Body: []byte(`{"ok":true}`)}, nil
	}

	_, _, metadata, err := runExecutionFallback(testExecutorRequest(), "callback-1")
	if err != nil {
		t.Fatalf("runExecutionFallback() error = %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"gpt-5.4"}) {
		t.Fatalf("calls = %#v, want direct fallback", calls)
	}
	if metadata["fallback_used"] != true {
		t.Fatalf("fallback_used = %#v, want true", metadata["fallback_used"])
	}
	if metadata["primary_cooldown_skipped"] != true {
		t.Fatalf("primary_cooldown_skipped = %#v, want true", metadata["primary_cooldown_skipped"])
	}
}

func TestRouteModelReturnsExplicitExecutorTarget(t *testing.T) {
	configureFallbackTest(t, 60)
	rawReq, errMarshal := json.Marshal(rpcModelRouteRequest{ModelRouteRequest: pluginapi.ModelRouteRequest{
		SourceFormat:   "claude",
		RequestedModel: "claude-haiku-4-5-20251001",
	}})
	if errMarshal != nil {
		t.Fatalf("marshal route request: %v", errMarshal)
	}

	rawResp, errRoute := routeModel(rawReq)
	if errRoute != nil {
		t.Fatalf("routeModel() error = %v", errRoute)
	}
	var env envelope
	if errDecode := json.Unmarshal(rawResp, &env); errDecode != nil {
		t.Fatalf("decode envelope: %v; body=%s", errDecode, rawResp)
	}
	if !env.OK {
		t.Fatalf("routeModel() envelope error = %#v", env.Error)
	}
	var resp pluginapi.ModelRouteResponse
	if errDecode := json.Unmarshal(env.Result, &resp); errDecode != nil {
		t.Fatalf("decode route response: %v; result=%s", errDecode, env.Result)
	}
	if !resp.Handled {
		t.Fatal("routeModel() Handled = false, want true")
	}
	if resp.TargetKind != pluginapi.ModelRouteTargetExecutor {
		t.Fatalf("TargetKind = %q, want %q", resp.TargetKind, pluginapi.ModelRouteTargetExecutor)
	}
	if resp.Target != pluginIdentifier {
		t.Fatalf("Target = %q, want %q", resp.Target, pluginIdentifier)
	}
}
func configureFallbackTest(t *testing.T, cooldownSeconds int) pluginConfig {
	t.Helper()
	originalExec := executeHostModelAttempt
	originalCooldowns := primaryCooldowns
	raw := []byte(`enabled: true
rules:
  - name: claude_quota
    source_formats:
      - claude
    models:
      - "claude-*"
    fallback_models:
      - gpt-5.4
fallback:
  cooldown_seconds: 60
`)
	cfg, err := decodeConfig(raw)
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	cfg.Fallback.CooldownSeconds = cooldownSeconds
	currentConfig.Store(cfg)
	primaryCooldowns = newPrimaryCooldownStore(func() time.Time {
		return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	})
	t.Cleanup(func() {
		executeHostModelAttempt = originalExec
		primaryCooldowns = originalCooldowns
		currentConfig.Store(defaultPluginConfig())
	})
	return cfg
}

func testExecutorRequest() pluginapi.ExecutorRequest {
	return pluginapi.ExecutorRequest{
		Model:           "claude-sonnet-4-5",
		SourceFormat:    "claude",
		OriginalRequest: []byte(`{"model":"claude-sonnet-4-5","messages":[]}`),
	}
}
