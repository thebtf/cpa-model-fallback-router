package main

import (
	"testing"
	"time"
)

func TestPrimaryCooldownStoreMarksExpiresAndClears(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	store := newPrimaryCooldownStore(func() time.Time { return now })
	key := fallbackCooldownKey("anthropic", fallbackRule{Name: "claude_quota"}, "claude-sonnet-4-5")
	if key == "" {
		t.Fatal("fallbackCooldownKey returned empty key")
	}
	if _, ok := store.active(key); ok {
		t.Fatal("fresh cooldown is active")
	}
	store.mark(key, 30*time.Second)
	until, ok := store.active(key)
	if !ok {
		t.Fatal("cooldown is not active after mark")
	}
	if want := now.Add(30 * time.Second); !until.Equal(want) {
		t.Fatalf("cooldown until = %s, want %s", until, want)
	}
	now = now.Add(31 * time.Second)
	if _, ok := store.active(key); ok {
		t.Fatal("expired cooldown is still active")
	}
	store.mark(key, time.Minute)
	store.clear()
	if _, ok := store.active(key); ok {
		t.Fatal("cleared cooldown is still active")
	}
}

func TestFallbackCooldownDuration(t *testing.T) {
	if got := fallbackCooldownDuration(fallbackSettings{CooldownSeconds: 0}); got != 0 {
		t.Fatalf("duration for zero cooldown = %s, want 0", got)
	}
	if got := fallbackCooldownDuration(fallbackSettings{CooldownSeconds: 90}); got != 90*time.Second {
		t.Fatalf("duration = %s, want 90s", got)
	}
}
