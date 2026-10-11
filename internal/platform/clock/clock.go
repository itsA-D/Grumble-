// Package clock provides an injectable time source so logic stays testable.
package clock

import "time"

// Clock abstracts the current time. Inject it; never call time.Now directly in
// service or domain code.
type Clock interface {
	// Now returns the current time in UTC.
	Now() time.Time
	// Since returns the elapsed time since t.
	Since(t time.Time) time.Duration
}

// System is the real, UTC clock.
type System struct{}

// Now returns the current time in UTC.
func (System) Now() time.Time { return time.Now().UTC() }

// Since returns the elapsed time since t.
func (System) Since(t time.Time) time.Duration { return time.Since(t) }

// Fixed is a clock that always returns the same instant, for tests.
type Fixed struct {
	// T is the instant the clock reports.
	T time.Time
}

// Now returns the fixed instant.
func (f Fixed) Now() time.Time { return f.T }

// Since returns the time since t relative to the fixed instant.
func (f Fixed) Since(t time.Time) time.Duration { return f.T.Sub(t) }
