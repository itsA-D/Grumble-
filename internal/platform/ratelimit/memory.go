package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Memory is a single-instance sliding-window limiter. It is correct for the
// dev mode and for single-replica deployments (RateLimit.Mode "memory");
// multi-replica deployments must switch to the Redis implementation.
type Memory struct {
	max    int
	window time.Duration
	now    func() time.Time

	mu   sync.Mutex
	hits map[string][]time.Time
}

// NewMemory builds a limiter keyed by an opaque string (typically a source
// hash or anon id). now is injectable for deterministic tests.
func NewMemory(cfg Config, now func() time.Time) *Memory {
	if now == nil {
		now = time.Now
	}
	return &Memory{
		max:    cfg.Max,
		window: cfg.Window,
		now:    now,
		hits:   make(map[string][]time.Time),
	}
}

// Allow records an allowance when the key is within budget, otherwise it
// reports false without recording. Expired history is pruned on access.
func (m *Memory) Allow(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	cutoff := now.Add(-m.window)
	history := m.hits[key]

	kept := history[:0]
	for _, at := range history {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= m.max {
		m.hits[key] = kept
		return false, nil
	}
	m.hits[key] = append(kept, now)

	// Opportunistically drop keys with no live history to bound memory.
	if len(m.hits) > 100_000 {
		for k, h := range m.hits {
			if len(h) == 0 || !h[len(h)-1].After(cutoff) {
				delete(m.hits, k)
			}
		}
	}
	return true, nil
}
