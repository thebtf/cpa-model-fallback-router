package main

import (
	"errors"
	"testing"
)

func TestShouldFallbackStatusPolicy(t *testing.T) {
	settings := fallbackSettings{
		Enabled:            true,
		FallbackOnStatus:   []int{429, 503},
		NoFallbackOnStatus: []int{400, 404},
	}
	if !shouldFallback(429, nil, settings) {
		t.Fatal("shouldFallback(429) = false, want true")
	}
	if shouldFallback(400, nil, settings) {
		t.Fatal("shouldFallback(400) = true, want false")
	}
	if !shouldFallback(0, errors.New("connection reset by peer"), settings) {
		t.Fatal("shouldFallback(network error) = false, want true")
	}
	if !shouldFallback(0, errors.New("This request would exceed your account's rate limit. Please try again later."), settings) {
		t.Fatal("shouldFallback(rate limit text) = false, want true")
	}
	if !shouldFallback(0, errors.New("auth_unavailable: no auth available"), settings) {
		t.Fatal("shouldFallback(auth unavailable text) = false, want true")
	}
	if !shouldFallback(0, errors.New("auth_not_found: no auth available"), settings) {
		t.Fatal("shouldFallback(auth not found text) = false, want true")
	}
	if !shouldFallback(0, errors.New("model_cooldown: model is cooling down"), settings) {
		t.Fatal("shouldFallback(model cooldown text) = false, want true")
	}
	if !shouldFallback(0, errors.New("account disabled by operator"), settings) {
		t.Fatal("shouldFallback(disabled account text) = false, want true")
	}
	if !shouldFallback(0, errors.New("host_call_failed: unknown provider for model claude-haiku-4-5-20251001"), settings) {
		t.Fatal("shouldFallback(unknown provider text) = false, want true")
	}
	settings.Enabled = false
	if shouldFallback(429, nil, settings) {
		t.Fatal("disabled shouldFallback(429) = true, want false")
	}
}

func TestStatusFromError(t *testing.T) {
	if got := statusFromError(statusError{status: 503}); got != 503 {
		t.Fatalf("statusFromError(statusError) = %d, want 503", got)
	}
	if got := statusFromError(errors.New("model execution failed with status 429")); got != 429 {
		t.Fatalf("statusFromError(message) = %d, want 429", got)
	}
}
