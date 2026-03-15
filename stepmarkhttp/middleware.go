package stepmarkhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/ImVivec/stepmark"
)

// TriggerFunc decides whether tracing should be enabled for a request.
// Return true to activate the tracer for this request.
type TriggerFunc func(r *http.Request) bool

// OnFinish is called after the handler completes with the collected
// trace. Use it for logging, exporting, or any post-request processing.
type OnFinish func(ctx context.Context, trace *stepmark.Trace)

// MiddlewareOption configures the behavior of [Middleware].
type MiddlewareOption func(*config)

type config struct {
	tracerOpts []stepmark.Option
	headerName string
	onFinish   OnFinish
}

// HeaderTrigger returns a [TriggerFunc] that activates tracing when the
// named request header is present and non-empty.
//
//	stepmarkhttp.HeaderTrigger("X-Stepmark")
func HeaderTrigger(name string) TriggerFunc {
	return func(r *http.Request) bool {
		return r.Header.Get(name) != ""
	}
}

// QueryTrigger returns a [TriggerFunc] that activates tracing when the
// named query parameter is present and non-empty.
//
//	stepmarkhttp.QueryTrigger("stepmark")
func QueryTrigger(param string) TriggerFunc {
	return func(r *http.Request) bool {
		return r.URL.Query().Get(param) != ""
	}
}

// WithResponseHeader writes the JSON-encoded trace to the named
// response header. The trace is captured just before the first byte
// of the response body is written, so it includes all events recorded
// during handler execution.
//
// For large traces consider using [WithOnFinish] instead, as HTTP
// headers have practical size limits (~8 KB in most servers).
func WithResponseHeader(name string) MiddlewareOption {
	return func(c *config) {
		c.headerName = name
	}
}

// WithOnFinish registers a callback that receives the complete trace
// after the handler returns. Unlike [WithResponseHeader], this always
// sees every event including any recorded during response serialization.
func WithOnFinish(fn OnFinish) MiddlewareOption {
	return func(c *config) {
		c.onFinish = fn
	}
}

// WithTracerOptions forwards [stepmark.Option] values to [stepmark.New]
// when the tracer is created.
//
//	stepmarkhttp.WithTracerOptions(stepmark.WithMaxEvents(500))
func WithTracerOptions(opts ...stepmark.Option) MiddlewareOption {
	return func(c *config) {
		c.tracerOpts = opts
	}
}

// Middleware returns a net/http middleware that conditionally enables
// Stepmark tracing. When trigger returns true for a request, a tracer
// is injected into the request context. All downstream handlers can
// then call [stepmark.Record], [stepmark.RecordEntity], etc.
//
//	mux.Handle("/api/", stepmarkhttp.Middleware(
//	    stepmarkhttp.HeaderTrigger("X-Stepmark"),
//	    stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
//	)(apiHandler))
func Middleware(trigger TriggerFunc, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !trigger(r) {
				next.ServeHTTP(w, r)
				return
			}

			ctx := stepmark.New(r.Context(), cfg.tracerOpts...)
			r = r.WithContext(ctx)

			if cfg.headerName != "" {
				dw := &deferredWriter{
					ResponseWriter: w,
					ctx:            ctx,
					headerName:     cfg.headerName,
				}
				next.ServeHTTP(dw, r)
				dw.inject() // covers handlers that return without writing
			} else {
				next.ServeHTTP(w, r)
			}

			if cfg.onFinish != nil {
				cfg.onFinish(ctx, stepmark.Collect(ctx))
			}
		})
	}
}

// deferredWriter intercepts Write/WriteHeader to inject the trace
// header before the response is flushed to the client.
type deferredWriter struct {
	http.ResponseWriter
	ctx        context.Context
	headerName string
	once       sync.Once
}

func (d *deferredWriter) inject() {
	d.once.Do(func() {
		trace := stepmark.Collect(d.ctx)
		if trace == nil {
			return
		}
		data, err := json.Marshal(trace)
		if err != nil {
			return
		}
		d.ResponseWriter.Header().Set(d.headerName, string(data))
	})
}

func (d *deferredWriter) WriteHeader(code int) {
	d.inject()
	d.ResponseWriter.WriteHeader(code)
}

func (d *deferredWriter) Write(b []byte) (int, error) {
	d.inject()
	return d.ResponseWriter.Write(b)
}

func (d *deferredWriter) Flush() {
	d.inject()
	if f, ok := d.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter, allowing
// [http.ResponseController] and other middleware to access it.
func (d *deferredWriter) Unwrap() http.ResponseWriter {
	return d.ResponseWriter
}
