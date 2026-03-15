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

// WithTraceMeta attaches metadata to the trace itself, not to any
// specific entity or event. Useful for request-level context like
// request IDs, user IDs, or experiment variants.
//
//	ctx := stepmark.New(ctx, stepmark.WithTraceMeta(map[string]any{
//	    "request_id": reqID,
//	    "user_id":    userID,
//	}))
func WithTraceMeta(meta map[string]any) Option {
	return func(t *tracer) {
		t.meta = cloneMap(meta)
	}
}

// EntityFilter decides whether an entity should be traced.
// Return true to trace the entity, false to skip it.
// Unscoped events recorded via [Record] are never filtered.
type EntityFilter func(entityID string) bool

// WithEntityFilter sets a predicate that controls which entities are
// traced. When the filter returns false for an entityID, all [Track],
// [RecordEntity], [StepEntity], and [EnterEntity] calls for that entity
// become no-ops.
//
//	ctx := stepmark.New(ctx, stepmark.WithEntityFilter(func(id string) bool {
//	    return id == targetOrderID
//	}))
func WithEntityFilter(fn EntityFilter) Option {
	return func(t *tracer) {
		t.entityFilter = fn
	}
}

// TrackOption configures entity tracking via [Track].
type TrackOption func(*entityState)

// WithKind sets the entity's kind, allowing consumers to group or
// filter entities by type. The kind appears in the collected
// [EntityTrace.Kind] field.
//
//	stepmark.Track(ctx, "prod_1", meta, stepmark.WithKind("product"))
//	stepmark.Track(ctx, "order_9", meta, stepmark.WithKind("order"))
func WithKind(kind string) TrackOption {
	return func(es *entityState) {
		es.kind = kind
	}
}
