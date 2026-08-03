package main

import (
	"net/http"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestRequestBodyInfoUsesOriginalRequestProtocolForCrossModelFallback(t *testing.T) {
	exec := pluginapi.ExecutorRequest{
		Model:           "claude-opus-4-8",
		SourceFormat:    "anthropic",
		Format:          "openai-response",
		OriginalRequest: []byte(`{"model":"claude-opus-4-8","system":"Claude Code","messages":[]}`),
		Payload:         []byte(`{"model":"gpt-5.5","input":[]}`),
	}

	info := requestBodyInfo(exec)
	if got, want := string(info.Body), `{"model":"claude-opus-4-8","system":"Claude Code","messages":[]}`; got != want {
		t.Fatalf("Body = %s, want original request %s", got, want)
	}
	if info.EntryProtocol != "claude" {
		t.Fatalf("EntryProtocol = %q, want claude", info.EntryProtocol)
	}
	if info.ResponseProtocol != "claude" {
		t.Fatalf("ResponseProtocol = %q, want claude", info.ResponseProtocol)
	}
}

func TestRequestBodyInfoUsesPayloadFormatWhenOriginalRequestMissing(t *testing.T) {
	exec := pluginapi.ExecutorRequest{
		Model:        "gpt-5.5",
		SourceFormat: "claude",
		Format:       "openai-response",
		Payload:      []byte(`{"model":"gpt-5.5","input":[]}`),
	}

	info := requestBodyInfo(exec)
	if got, want := string(info.Body), `{"model":"gpt-5.5","input":[]}`; got != want {
		t.Fatalf("Body = %s, want payload %s", got, want)
	}
	if info.EntryProtocol != "openai-response" {
		t.Fatalf("EntryProtocol = %q, want openai-response", info.EntryProtocol)
	}
	if info.ResponseProtocol != "claude" {
		t.Fatalf("ResponseProtocol = %q, want original downstream claude", info.ResponseProtocol)
	}
}

func TestHostModelExecutionPayloadCarriesProtocolPair(t *testing.T) {
	exec := pluginapi.ExecutorRequest{
		Headers: http.Header{"X-Test": []string{"1"}},
		Alt:     "beta",
	}
	body := []byte(`{"model":"gpt-5.5","input":[]}`)

	got := hostModelExecutionPayload(exec, "gpt-5.5", "openai-response", "claude", true, body)
	if got.EntryProtocol != "openai-response" {
		t.Fatalf("EntryProtocol = %q, want openai-response", got.EntryProtocol)
	}
	if got.ExitProtocol != "claude" {
		t.Fatalf("ExitProtocol = %q, want claude", got.ExitProtocol)
	}
	if got.Model != "gpt-5.5" || !got.Stream || string(got.Body) != string(body) || got.Alt != "beta" {
		t.Fatalf("host payload = %#v", got)
	}
	if got.Headers.Get("X-Test") != "1" {
		t.Fatalf("Headers = %#v, want cloned X-Test", got.Headers)
	}
}
