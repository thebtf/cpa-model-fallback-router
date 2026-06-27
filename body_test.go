package main

import (
	"bytes"
	"testing"
)

func TestRequestBodyForModelRewritesTopLevelModel(t *testing.T) {
	got := requestBodyForModel([]byte(`{"model":"claude-sonnet-4-5","messages":[]}`), "gpt-5.4")
	if !bytes.Contains(got, []byte(`"model":"gpt-5.4"`)) {
		t.Fatalf("rewritten body = %s, want gpt-5.4 model", got)
	}
}

func TestRequestBodyForModelLeavesInvalidJSON(t *testing.T) {
	input := []byte(`not-json`)
	got := requestBodyForModel(input, "gpt-5.4")
	if !bytes.Equal(got, input) {
		t.Fatalf("requestBodyForModel(invalid JSON) = %s, want original", got)
	}
}

func TestRequestBodyForModelLeavesBodyWithoutModel(t *testing.T) {
	input := []byte(`{"messages":[]}`)
	got := requestBodyForModel(input, "gpt-5.4")
	if !bytes.Equal(got, input) {
		t.Fatalf("requestBodyForModel(no model) = %s, want original", got)
	}
}
