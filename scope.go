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
//
// A Scope is a thin, zero-allocation wrapper. It holds a reference to
// the context and the kind string — no additional state is created.
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
