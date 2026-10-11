package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryAllowsWithinBudget(t *testing.T) {
	m := NewMemory(Config{Window: time.Hour, Max: 2}, func() time.Time {
		return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	})
	ctx := context.Background()

	if ok, _ := m.Allow(ctx, "k"); !ok {
		t.Fatal("first call should be allowed")
	}
	if ok, _ := m.Allow(ctx, "k"); !ok {
		t.Fatal("second call should be allowed")
	}
	if ok, _ := m.Allow(ctx, "k"); ok {
		t.Fatal("third call must be rejected at the budget of 2")
	}
	if ok, _ := m.Allow(ctx, "other"); !ok {
		t.Fatal("a different key must have its own budget")
	}
}

func TestMemoryWindowRefills(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	m := NewMemory(Config{Window: 10 * time.Minute, Max: 1}, func() time.Time { return now })

	if ok, _ := m.Allow(context.Background(), "k"); !ok {
		t.Fatal("first call should be allowed")
	}
	if ok, _ := m.Allow(context.Background(), "k"); ok {
		t.Fatal("call within the window must be rejected")
	}
	now = now.Add(10 * time.Minute)
	if ok, _ := m.Allow(context.Background(), "k"); !ok {
		t.Fatal("after the window slides the key must be allowed again")
	}
}

func TestMemoryIsSliding(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	m := NewMemory(Config{Window: 10 * time.Minute, Max: 2}, func() time.Time { return now })

	if ok, _ := m.Allow(context.Background(), "k"); !ok { // at T
		t.Fatal("first call should be allowed")
	}
	now = now.Add(9 * time.Minute)
	if ok, _ := m.Allow(context.Background(), "k"); !ok { // T+9m
		t.Fatal("second call should be allowed")
	}
	if ok, _ := m.Allow(context.Background(), "k"); ok {
		t.Fatal("budget exhausted: two hits within the window")
	}
	now = now.Add(2 * time.Minute) // T+11m: the T hit has left the window
	if ok, _ := m.Allow(context.Background(), "k"); !ok {
		t.Fatal("sliding window must have released the oldest hit")
	}
}
