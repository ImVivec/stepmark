package stepmark

import (
	"context"
	"runtime"
	"strings"
)

// noop is returned by Enter/EnterEntity when tracing is disabled
// so that `defer stepmark.Enter(ctx, nil)()` never allocates.
var noop = func() {}

// Step records an unscoped event, automatically using the calling
// function's name as the stage. This eliminates the need to manually
// pass stage strings.
//
//	func ProcessOrder(ctx context.Context) {
//	    stepmark.Step(ctx, "started", nil)       // stage = "ProcessOrder"
//	    stepmark.Step(ctx, "validated", nil)      // stage = "ProcessOrder"
//	}
//
// When tracing is disabled, this is a no-op (~2ns, 0 allocs).
// When enabled, [runtime.Caller] resolves the function name (~200ns
// overhead on top of the normal recording cost).
func Step(ctx context.Context, action string, meta map[string]any) {
	t, _ := ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return
	}
	t.record(resolveCallerName(2), action, meta)
}

// StepEntity records an entity event, automatically using the calling
// function's name as the stage. Respects [WithEntityFilter].
//
//	func ScoreProduct(ctx context.Context, productID string, score float64) {
//	    stepmark.StepEntity(ctx, productID, "scored", map[string]any{"score": score})
//	    // stage = "ScoreProduct"
//	}
func StepEntity(ctx context.Context, entityID, action string, meta map[string]any) {
	t, _ := ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return
	}
	if t.entityFilter != nil && !t.entityFilter(entityID) {
		return
	}
	t.recordEntity(entityID, resolveCallerName(2), action, meta)
}

// Enter records a function entry event and returns a function that
// records the exit with duration. Typically used with defer:
//
//	func ValidateOrder(ctx context.Context) error {
//	    defer stepmark.Enter(ctx, nil)()
//	    // stage = "ValidateOrder", action = "entered" ... then "exited"
//	}
//
// When tracing is disabled, returns a package-level no-op function
// (zero allocations). The defer itself costs ~50ns regardless.
func Enter(ctx context.Context, meta map[string]any) func() {
	t, _ := ctx.Value(contextKey{}).(*tracer)
	if t == nil {
		return noop
	}
	stage := resolveCallerName(2)
	t.record(stage, "entered", meta)
	start := t.clock()
	return func() {
		elapsed := t.clock().Sub(start)
		t.record(stage, "exited", map[string]any{
			"duration_ms": float64(elapsed.Microseconds()) / 1000.0,
		})
	}
}

// EnterEntity records a function entry for a specific entity and returns
// a function that records the exit with duration. Respects [WithEntityFilter].
//
//	func ChargePayment(ctx context.Context, orderID string) error {
//	    defer stepmark.EnterEntity(ctx, orderID, nil)()
//	    // Records entered/exited events on orderID with stage = "ChargePayment"
//	}
func EnterEntity(ctx context.Context, entityID string, meta map[string]any) func() {
	t, _ := ctx.Value(contextKey{}).(*tracer)
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

// resolveCallerName returns the short function name of the caller
// at the given stack depth. skip=2 means: resolveCallerName → public
// function (Step/Enter/etc.) → user's function.
func resolveCallerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	return cleanFuncName(runtime.FuncForPC(pc).Name())
}

// cleanFuncName extracts a readable function name from the fully
// qualified name returned by runtime.FuncForPC.
//
//	"github.com/user/pkg.Function"            → "Function"
//	"github.com/user/pkg.(*Type).Method"      → "Type.Method"
//	"github.com/user/pkg.Function.func1"      → "Function.func1"
func cleanFuncName(full string) string {
	if i := strings.LastIndexByte(full, '/'); i >= 0 {
		full = full[i+1:]
	}
	if i := strings.IndexByte(full, '.'); i >= 0 {
		full = full[i+1:]
	}
	full = strings.TrimPrefix(full, "(*")
	full = strings.Replace(full, ").", ".", 1)
	return full
}
