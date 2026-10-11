// Package ratelimit guards shared endpoints against abuse. The Limiter
// interface is declared here (an adapter) and implemented in memory or against
// Redis; only the composition root chooses the implementation.
package ratelimit

import (
	"context"
	"time"
)

// Limiter reports whether a key may proceed within its window.
type Limiter interface {
	// Allow returns true when the key is within its budget. A false result is
	// not an error: callers map it to the domain's ErrRateLimited.
	Allow(ctx context.Context, key string) (bool, error)
}

// Config bounds one key within a sliding window.
type Config struct {
	// Window is how far back history is considered.
	Window time.Duration
	// Max is how many allowances fit in one window.
	Max int
}
