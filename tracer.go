package breadcrumb

import (
	"context"
	"time"
)

func newBreadcrumbTracer() *breadcrumbTracer {
	return &breadcrumbTracer{
		productTraces: make(map[string]*ProductTrace),
		globalTraces:  make([]TraceEvent, 0),
	}
}

func getBreadcrumbTracer(ctx context.Context) (*breadcrumbTracer, bool) {
	s, ok := ctx.Value(breadcrumbTracerKey).(*breadcrumbTracer)
	return s, ok
}

// WithBreadcrumbTracer returns a child context that carries a new breadcrumb tracer.
func WithBreadcrumbTracer(ctx context.Context) context.Context {
	return context.WithValue(ctx, breadcrumbTracerKey, newBreadcrumbTracer())
}

// IsBreadcrumbTracingEnabled reports whether the context carries a breadcrumb tracer.
func IsBreadcrumbTracingEnabled(ctx context.Context) bool {
	if _, ok := ctx.Value(breadcrumbTracerKey).(*breadcrumbTracer); ok {
		return true
	}
	return false
}

// AddProduct registers (or re-initialises) a product in the tracer with optional metadata.
func AddProduct(ctx context.Context, productID string, meta map[string]interface{}) {
	tracer, ok := getBreadcrumbTracer(ctx)
	if !ok {
		return
	}
	tracer.mu.Lock()
	defer tracer.mu.Unlock()
	pt, exists := tracer.productTraces[productID]
	if !exists {
		tracer.productTraces[productID] = &ProductTrace{ProductID: productID, ProductMeta: copyMap(meta), Traces: make([]TraceEvent, 0)}
		return
	}
	metaCopy := copyMap(meta)
	metaCopy["re-initialized"] = true
	if pt.ProductMeta == nil {
		pt.ProductMeta = metaCopy
		return
	}
	for k, v := range metaCopy {
		pt.ProductMeta[k] = v
	}
}

// RecordProduct appends a trace event to the given product.
func RecordProduct(ctx context.Context, productID, stage, action string, meta map[string]interface{}) {
	tracer, ok := getBreadcrumbTracer(ctx)
	if !ok {
		return
	}
	tracer.mu.Lock()
	defer tracer.mu.Unlock()

	pt, exists := tracer.productTraces[productID]
	if !exists {
		pt = &ProductTrace{ProductID: productID, Traces: make([]TraceEvent, 0, 4)}
		tracer.productTraces[productID] = pt
	}
	pt.Traces = append(pt.Traces, TraceEvent{Timestamp: time.Now().UTC(), Stage: stage, Action: action, Meta: copyMap(meta)})
}

// RecordGlobal appends a trace event to the global (non-product-specific) trace log.
func RecordGlobal(ctx context.Context, stage, action string, meta map[string]interface{}) {
	tracer, ok := getBreadcrumbTracer(ctx)
	if !ok {
		return
	}
	tracer.mu.Lock()
	defer tracer.mu.Unlock()

	tracer.globalTraces = append(tracer.globalTraces, TraceEvent{
		Timestamp: time.Now().UTC(),
		Stage:     stage,
		Action:    action,
		Meta:      copyMap(meta),
	})
}

// GetGlobalTraces returns a deep-copied slice of all global trace events.
func GetGlobalTraces(ctx context.Context) []TraceEvent {
	tracer, ok := getBreadcrumbTracer(ctx)
	if !ok {
		return nil
	}
	tracer.mu.RLock()
	defer tracer.mu.RUnlock()

	if len(tracer.globalTraces) == 0 {
		return nil
	}

	out := make([]TraceEvent, len(tracer.globalTraces))
	for i, trace := range tracer.globalTraces {
		out[i] = TraceEvent{
			Stage:     trace.Stage,
			Action:    trace.Action,
			Timestamp: trace.Timestamp,
			Meta:      copyMap(trace.Meta),
		}
	}
	return out
}

// GetProductTraces returns a deep-copied map of all per-product traces.
func GetProductTraces(ctx context.Context) map[string]ProductTrace {
	tracer, ok := getBreadcrumbTracer(ctx)
	if !ok {
		return nil
	}
	tracer.mu.RLock()
	defer tracer.mu.RUnlock()
	out := make(map[string]ProductTrace, len(tracer.productTraces))
	for _, v := range tracer.productTraces {
		trCopy := make([]TraceEvent, len(v.Traces))
		for i, trace := range v.Traces {
			trCopy[i] = TraceEvent{
				Stage:     trace.Stage,
				Action:    trace.Action,
				Timestamp: trace.Timestamp,
				Meta:      copyMap(trace.Meta),
			}
		}
		var pm map[string]interface{}
		if v.ProductMeta != nil {
			pm = copyMap(v.ProductMeta)
		}
		out[v.ProductID] = ProductTrace{ProductID: v.ProductID, ProductMeta: pm, Traces: trCopy}
	}
	return out
}

// GetBreadcrumbParams is a convenience function that collects all traces
// into a single BreadcrumbParams value, ready for serialisation.
// Returns nil when tracing is not enabled on the context.
func GetBreadcrumbParams(ctx context.Context) *BreadcrumbParams {
	if !IsBreadcrumbTracingEnabled(ctx) {
		return nil
	}
	return &BreadcrumbParams{
		ProductTraces: GetProductTraces(ctx),
		GlobalTraces:  GetGlobalTraces(ctx),
	}
}

func copyMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
