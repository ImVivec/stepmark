package stepmark

import "context"

// Scope provides a convenient way to track multiple entities of the
// same kind without repeating [WithKind] on every call.
//
//	products := stepmark.NewScope(ctx, "product")
//	for _, p := range results {
//	    products.Track(p.ID, map[string]any{"name": p.Name})
//	    products.RecordEvent(p.ID, "ranking", "scored", map[string]any{"score": p.Score})
//	}
type Scope struct {
	ctx  context.Context
	kind string
}

// NewScope creates a [Scope] for tracking entities of the given kind.
// The scope uses the provided context for all operations.
func NewScope(ctx context.Context, kind string) Scope {
	return Scope{ctx: ctx, kind: kind}
}

// Track registers an entity within this scope, automatically setting
// its [EntityTrace.Kind]. Equivalent to calling the package-level
// [Track] with [WithKind].
func (s Scope) Track(entityID string, meta map[string]any) {
	Track(s.ctx, entityID, meta, WithKind(s.kind))
}

// RecordEvent appends an event to the named entity's trace.
// Equivalent to calling the package-level [RecordEntity].
func (s Scope) RecordEvent(entityID, stage, action string, meta map[string]any) {
	RecordEntity(s.ctx, entityID, stage, action, meta)
}

// Step records an entity event using the caller's function name as the
// stage. Equivalent to calling the package-level [StepEntity].
func (s Scope) Step(entityID, action string, meta map[string]any) {
	t, _ := s.ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return
	}
	if t.entityFilter != nil && !t.entityFilter(entityID) {
		return
	}
	t.recordEntity(entityID, resolveCallerName(2), action, meta)
}

// Enter records a function entry for a specific entity and returns a
// function that records the exit with duration. Equivalent to calling
// the package-level [EnterEntity].
func (s Scope) Enter(entityID string, meta map[string]any) func() {
	t, _ := s.ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return noop
	}
	if t.entityFilter != nil && !t.entityFilter(entityID) {
		return noop
	}
	stage := resolveCallerName(2)
	t.recordEntity(entityID, stage, "entered", meta)
	start := t.clock()
	return func() {
		elapsed := t.clock().Sub(start)
		t.recordEntity(entityID, stage, "exited", map[string]any{
			"duration_ms": float64(elapsed.Microseconds()) / 1000.0,
		})
	}
}
