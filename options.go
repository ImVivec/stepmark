package stepmark

import "time"

// Option configures a tracer created by [New].
type Option func(*tracer)

// WithMaxEvents sets a hard cap on the total number of recorded
// events across all entities and unscoped events. Once the cap is
// reached, subsequent Record and RecordEntity calls are silently
// dropped. Zero (the default) means no limit.
func WithMaxEvents(n int) Option {
	return func(t *tracer) {
		t.maxEvents = n
	}
}

// WithClock overrides the time source used for event timestamps.
// Useful for deterministic tests. The provided function must be
// safe for concurrent use.
func WithClock(fn func() time.Time) Option {
	return func(t *tracer) {
		t.clock = fn
	}
}
