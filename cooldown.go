package main

import (
	"strings"
	"sync"
	"time"
)

var primaryCooldowns = newPrimaryCooldownStore(time.Now)

type primaryCooldownStore struct {
	mu    sync.Mutex
	now   func() time.Time
	until map[string]time.Time
}

func newPrimaryCooldownStore(now func() time.Time) *primaryCooldownStore {
	if now == nil {
		now = time.Now
	}
	return &primaryCooldownStore{now: now, until: make(map[string]time.Time)}
}

func (s *primaryCooldownStore) active(key string) (time.Time, bool) {
	key = strings.TrimSpace(key)
	if s == nil || key == "" {
		return time.Time{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	until := s.until[key]
	if until.IsZero() || !until.After(s.now()) {
		delete(s.until, key)
		return time.Time{}, false
	}
	return until, true
}

func (s *primaryCooldownStore) mark(key string, duration time.Duration) {
	key = strings.TrimSpace(key)
	if s == nil || key == "" || duration <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.until[key] = s.now().Add(duration)
}

func (s *primaryCooldownStore) clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.until = make(map[string]time.Time)
}

func fallbackCooldownDuration(settings fallbackSettings) time.Duration {
	if settings.CooldownSeconds <= 0 {
		return 0
	}
	return time.Duration(settings.CooldownSeconds) * time.Second
}

func fallbackCooldownKey(sourceFormat string, rule fallbackRule, primaryModel string) string {
	source := normalizeProtocol(sourceFormat)
	ruleName := strings.ToLower(strings.TrimSpace(rule.Name))
	primary := strings.ToLower(strings.TrimSpace(primaryModel))
	if ruleName == "" || primary == "" {
		return ""
	}
	return source + "\x00" + ruleName + "\x00" + primary
}
