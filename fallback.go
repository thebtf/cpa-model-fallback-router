package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var statusPattern = regexp.MustCompile(`(?i)(?:status|http status|status_code)[^0-9]*(\d{3})`)

type statusError struct {
	status  int
	message string
}

func (e statusError) Error() string {
	if strings.TrimSpace(e.message) != "" {
		return e.message
	}
	if e.status > 0 {
		return fmt.Sprintf("model execution failed with status %d", e.status)
	}
	return "model execution failed"
}

func (e statusError) StatusCode() int { return e.status }

func fallbackPolicy(cfg pluginConfig, rule fallbackRule) fallbackSettings {
	out := cfg.Fallback
	if len(rule.FallbackOnStatus) > 0 {
		out.FallbackOnStatus = append([]int(nil), rule.FallbackOnStatus...)
	}
	if len(rule.NoFallbackOnStatus) > 0 {
		out.NoFallbackOnStatus = append([]int(nil), rule.NoFallbackOnStatus...)
	}
	return out
}

func shouldFallback(status int, err error, settings fallbackSettings) bool {
	if !settings.Enabled {
		return false
	}
	if statusInList(status, settings.NoFallbackOnStatus) {
		return false
	}
	if statusInList(status, settings.FallbackOnStatus) {
		return true
	}
	return status == 0 && (isNetworkError(err) || isRateLimitError(err))
}

func statusInList(status int, list []int) bool {
	if status <= 0 {
		return false
	}
	for _, item := range list {
		if item == status {
			return true
		}
	}
	return false
}

func statusFromError(err error) int {
	if err == nil {
		return 0
	}
	var carrier interface{ StatusCode() int }
	if errors.As(err, &carrier) && carrier.StatusCode() > 0 {
		return carrier.StatusCode()
	}
	match := statusPattern.FindStringSubmatch(err.Error())
	if len(match) == 2 {
		code, errParse := strconv.Atoi(match[1])
		if errParse == nil && code >= 100 && code <= 599 {
			return code
		}
	}
	return 0
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, token := range []string{
		"timeout",
		"timed out",
		"connection reset",
		"connection refused",
		"connection aborted",
		"broken pipe",
		"no such host",
		"dns",
		"temporary failure",
		"network is unreachable",
		"eof",
	} {
		if strings.Contains(message, token) {
			return true
		}
	}
	return false
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, token := range []string{
		"rate limit",
		"ratelimit",
		"too many requests",
		"quota",
		"exceed your account",
	} {
		if strings.Contains(message, token) {
			return true
		}
	}
	return false
}
func successStatus(status int) bool {
	return status >= 200 && status < 300
}

func statusOrDefault(status int) int {
	if status > 0 {
		return status
	}
	return 502
}
