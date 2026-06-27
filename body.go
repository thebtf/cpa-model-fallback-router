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

func requestBody(exec pluginapi.ExecutorRequest) []byte {
	if len(exec.OriginalRequest) > 0 {
		return bytes.Clone(exec.OriginalRequest)
	}
	if len(exec.Payload) > 0 {
		return bytes.Clone(exec.Payload)
	}
	return nil
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
