package stepmark

import "context"

// New returns a child context with an active tracer. All Record
// and Track calls on this context (and its children) accumulate
// events until Collect extracts the final snapshot.
func New(ctx context.Context, opts ...Option) context.Context {
	return context.WithValue(ctx, contextKey{}, newTracer(opts))
}

// Enabled reports whether ctx carries an active tracer.
// This compiles down to a single context-chain walk — no type
// assertion, no allocation, no lock.
func Enabled(ctx context.Context) bool {
	return ctx.Value(contextKey{}) != nil
}

// Track registers an entity for tracing with optional metadata.
// If the entity was already tracked, the new metadata is merged
// into the existing metadata. No-op when tracing is disabled.
func Track(ctx context.Context, entityID string, meta map[string]any) {
	t, _ := ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return
	}
	t.track(entityID, meta)
}

// RecordEntity appends an event to the named entity's trace.
// The entity is auto-created if it has not been tracked yet.
// No-op when tracing is disabled.
func RecordEntity(ctx context.Context, entityID, stage, action string, meta map[string]any) {
	t, _ := ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return
	}
	t.recordEntity(entityID, stage, action, meta)
}

// Record appends an unscoped event not tied to any entity.
// No-op when tracing is disabled.
func Record(ctx context.Context, stage, action string, meta map[string]any) {
	t, _ := ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return
	}
	t.record(stage, action, meta)
}

// Collect returns a deep-copied snapshot of all recorded data.
// Returns nil when tracing is disabled on ctx. Safe to call
// multiple times; each call returns an independent copy.
func Collect(ctx context.Context) *Trace {
	t, _ := ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return nil
	}
	return t.collect()
}
