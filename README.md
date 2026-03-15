# Stepmark

**On-Demand Business Logic Tracer for Go**

[![Go Reference](https://pkg.go.dev/badge/github.com/vivekpatidar/stepmark.svg)](https://pkg.go.dev/github.com/vivekpatidar/stepmark)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Standard logs are noisy. APM tools track latency, not logic. Stepmark traces the **decision-making journey** of specific entities — orders, users, search results — through your codebase. On demand, with zero overhead when off.

---

## The Problem

You have a complex backend where an order flows through validation, fraud detection, inventory checks, a pricing engine, and payment. Something goes wrong for a specific order. Now what?

- **Logs?** Grep through millions of lines across a dozen services.
- **APM?** It tells you the request took 200ms. It doesn't tell you *why* the order was rejected.
- **Debugger?** Good luck attaching one to production.

Stepmark gives you a structured audit trail of every decision point, but **only when you ask for it**. In normal production traffic, each call costs under 2 nanoseconds with zero memory allocations.

---

## Install

```bash
go get github.com/vivekpatidar/stepmark
```

Zero external dependencies. Only uses the Go standard library.

---

## Quick Start

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/vivekpatidar/stepmark"
)

func main() {
    ctx := stepmark.New(context.Background())

    stepmark.Track(ctx, "order_42", map[string]any{"customer": "alice"})
    stepmark.RecordEntity(ctx, "order_42", "validation", "passed", nil)
    stepmark.RecordEntity(ctx, "order_42", "fraud_check", "cleared", map[string]any{
        "score": 0.02,
        "model": "v3",
    })
    stepmark.RecordEntity(ctx, "order_42", "payment", "charged", map[string]any{
        "amount":   99.99,
        "currency": "USD",
    })

    trace := stepmark.Collect(ctx)
    data, _ := json.MarshalIndent(trace, "", "  ")
    fmt.Println(string(data))
}
```

**Output:**

```json
{
  "entities": {
    "order_42": {
      "entity_id": "order_42",
      "meta": { "customer": "alice" },
      "events": [
        { "stage": "validation",  "action": "passed",  "timestamp": "..." },
        { "stage": "fraud_check", "action": "cleared", "timestamp": "...", "meta": { "score": 0.02, "model": "v3" } },
        { "stage": "payment",     "action": "charged", "timestamp": "...", "meta": { "amount": 99.99, "currency": "USD" } }
      ]
    }
  }
}
```

---

## Zero Overhead Guarantee

When tracing is not enabled (the normal case in production), every Stepmark call compiles down to a nil-check on `context.Value()`. No allocations. No locks. No map lookups.

```
goos: darwin
goarch: arm64
cpu: Apple M4

BenchmarkEnabled_Disabled             692M      1.72 ns/op    0 B/op   0 allocs/op
BenchmarkRecord_Disabled              670M      1.78 ns/op    0 B/op   0 allocs/op
BenchmarkRecordEntity_Disabled        660M      1.86 ns/op    0 B/op   0 allocs/op
BenchmarkTrack_Disabled               680M      1.76 ns/op    0 B/op   0 allocs/op
```

Put `stepmark.Record(...)` calls throughout your hot paths. When nobody is tracing, each call costs **< 2 nanoseconds**.

When tracing *is* enabled (the rare debug case):

```
BenchmarkEnabled_Enabled              406M      2.96 ns/op    0 B/op   0 allocs/op
BenchmarkRecord_Enabled               5.6M      226  ns/op    704 B/op 2 allocs/op
BenchmarkRecordEntity_Enabled         4.9M      264  ns/op    690 B/op 3 allocs/op
```

---

## API

The entire public API is **6 functions**:

| Function | Purpose |
|---|---|
| `New(ctx, ...Option) context.Context` | Start tracing — injects a tracer into the context |
| `Enabled(ctx) bool` | Check if tracing is active (< 2ns fast path) |
| `Track(ctx, entityID, meta)` | Register an entity with optional metadata |
| `RecordEntity(ctx, entityID, stage, action, meta)` | Record an event for a specific entity |
| `Record(ctx, stage, action, meta)` | Record an unscoped event (not tied to an entity) |
| `Collect(ctx) *Trace` | Extract a deep-copied snapshot of all recorded data |

**Every function is a no-op when tracing is disabled.** You never need to guard with `if stepmark.Enabled(ctx)` for correctness — only as an optional optimization to avoid allocating a `map[string]any` literal on the hot path.

---

## HTTP Middleware

The `stepmarkhttp` subpackage provides ready-made middleware for `net/http` and compatible routers (Chi, gorilla/mux, etc.).

### Standard Library

```go
import "github.com/vivekpatidar/stepmark/stepmarkhttp"

mux := http.NewServeMux()
mux.Handle("/api/", stepmarkhttp.Middleware(
    stepmarkhttp.HeaderTrigger("X-Stepmark"),
    stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
)(apiHandler))
```

Send `X-Stepmark: true` with your request, and the response includes an `X-Stepmark-Trace` header containing the full JSON trace.

### With a Query Parameter Trigger

```go
// Enable tracing with ?stepmark=true
mw := stepmarkhttp.Middleware(
    stepmarkhttp.QueryTrigger("stepmark"),
    stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
)
```

### With a Callback

```go
mw := stepmarkhttp.Middleware(
    stepmarkhttp.HeaderTrigger("X-Stepmark"),
    stepmarkhttp.WithOnFinish(func(ctx context.Context, trace *stepmark.Trace) {
        data, _ := json.Marshal(trace)
        slog.InfoContext(ctx, "stepmark trace", "trace", string(data))
    }),
)
```

### With Both Header and Callback

```go
mw := stepmarkhttp.Middleware(
    stepmarkhttp.HeaderTrigger("X-Stepmark"),
    stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
    stepmarkhttp.WithOnFinish(func(ctx context.Context, trace *stepmark.Trace) {
        exportToDatadog(ctx, trace)
    }),
    stepmarkhttp.WithTracerOptions(stepmark.WithMaxEvents(500)),
)
```

---

## Framework Integration

### Chi

Chi uses the standard `func(http.Handler) http.Handler` middleware signature. It works directly:

```go
r := chi.NewRouter()
r.Use(stepmarkhttp.Middleware(
    stepmarkhttp.HeaderTrigger("X-Stepmark"),
    stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
))
```

### Gin

```go
func StepmarkMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.GetHeader("X-Stepmark") == "" {
            c.Next()
            return
        }
        ctx := stepmark.New(c.Request.Context())
        c.Request = c.Request.WithContext(ctx)
        c.Next()
        if trace := stepmark.Collect(ctx); trace != nil {
            data, _ := json.Marshal(trace)
            c.Header("X-Stepmark-Trace", string(data))
        }
    }
}
```

### Echo

```go
func StepmarkMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            if c.Request().Header.Get("X-Stepmark") == "" {
                return next(c)
            }
            ctx := stepmark.New(c.Request().Context())
            c.SetRequest(c.Request().WithContext(ctx))
            err := next(c)
            if trace := stepmark.Collect(ctx); trace != nil {
                data, _ := json.Marshal(trace)
                c.Response().Header().Set("X-Stepmark-Trace", string(data))
            }
            return err
        }
    }
}
```

### Fiber

```go
func StepmarkMiddleware() fiber.Handler {
    return func(c fiber.Ctx) error {
        if c.Get("X-Stepmark") == "" {
            return c.Next()
        }
        ctx := stepmark.New(c.Context())
        c.SetContext(ctx)
        err := c.Next()
        if trace := stepmark.Collect(ctx); trace != nil {
            data, _ := json.Marshal(trace)
            c.Set("X-Stepmark-Trace", string(data))
        }
        return err
    }
}
```

### gRPC (Unary Interceptor)

```go
func StepmarkInterceptor(
    ctx context.Context,
    req any,
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (any, error) {
    md, _ := metadata.FromIncomingContext(ctx)
    if len(md.Get("x-stepmark")) == 0 {
        return handler(ctx, req)
    }

    ctx = stepmark.New(ctx)
    resp, err := handler(ctx, req)

    if trace := stepmark.Collect(ctx); trace != nil {
        data, _ := json.Marshal(trace)
        grpc.SetTrailer(ctx, metadata.Pairs("x-stepmark-trace", string(data)))
    }
    return resp, err
}
```

---

## Real-World Example

A search API that traces how products are ranked, filtered, and scored:

```go
func SearchHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    query := r.URL.Query().Get("q")

    stepmark.Record(ctx, "search", "query_received", map[string]any{
        "query":   query,
        "user_id": getUserID(ctx),
    })

    products := catalog.Search(ctx, query)
    stepmark.Record(ctx, "search", "catalog_returned", map[string]any{
        "count": len(products),
    })

    for _, p := range products {
        stepmark.Track(ctx, p.ID, map[string]any{
            "name":     p.Name,
            "category": p.Category,
        })
    }

    ranked := ranking.Apply(ctx, products) // internally calls RecordEntity per product
    filtered := filters.Apply(ctx, ranked) // internally calls RecordEntity per product

    stepmark.Record(ctx, "search", "response_ready", map[string]any{
        "final_count": len(filtered),
    })

    json.NewEncoder(w).Encode(filtered)
}

// Inside ranking.Apply:
func (r *Ranker) Apply(ctx context.Context, products []Product) []Product {
    for i, p := range products {
        score := r.model.Score(p)
        stepmark.RecordEntity(ctx, p.ID, "ranking", "scored", map[string]any{
            "score":    score,
            "position": i,
            "model":    r.modelVersion,
        })
        products[i].Score = score
    }
    sort.Slice(products, func(i, j int) bool {
        return products[i].Score > products[j].Score
    })
    return products
}
```

With the middleware enabled, a single request with `X-Stepmark: true` returns a complete audit trail showing exactly why each product ended up where it did.

---

## Options

```go
// Cap total events to prevent runaway growth in pathological cases.
ctx := stepmark.New(ctx, stepmark.WithMaxEvents(1000))

// Inject a fixed clock for deterministic tests.
fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
ctx := stepmark.New(ctx, stepmark.WithClock(func() time.Time { return fixed }))
```

---

## Types

### Event

A single recorded step.

```go
type Event struct {
    Stage     string         `json:"stage"`
    Action    string         `json:"action"`
    Timestamp time.Time      `json:"timestamp"`
    Meta      map[string]any `json:"meta,omitempty"`
}
```

### EntityTrace

All events for one tracked entity.

```go
type EntityTrace struct {
    EntityID string         `json:"entity_id"`
    Meta     map[string]any `json:"meta,omitempty"`
    Events   []Event        `json:"events"`
}
```

### Trace

The complete output from `Collect`. Ready for `json.Marshal`.

```go
type Trace struct {
    Entities map[string]EntityTrace `json:"entities,omitempty"`
    Events   []Event                `json:"events,omitempty"`
}
```

---

## FAQ

**Q: Do I need to call `Enabled()` before `Record()`?**

No. Every function is a no-op when tracing is disabled. The only reason to check `Enabled()` is to avoid allocating a `map[string]any` on the hot path:

```go
// Always correct, always safe:
stepmark.Record(ctx, "stage", "action", map[string]any{"key": value})

// Slightly faster in the disabled case (avoids map allocation):
if stepmark.Enabled(ctx) {
    stepmark.Record(ctx, "stage", "action", map[string]any{"key": value})
}
```

**Q: Is it safe to use from multiple goroutines?**

Yes. The tracer uses a mutex internally. All functions are safe for concurrent use as long as they share the same context.

**Q: What happens if I call `Collect()` multiple times?**

Each call returns an independent deep copy. Calling `Collect()` does not consume or reset the tracer. Events recorded between calls appear in subsequent snapshots.

**Q: Will it slow down my production traffic?**

No. When tracing is not enabled, every call is a single `context.Value()` lookup that returns nil, followed by an early return. Benchmarked at 1.7–1.9 ns with zero allocations. This is comparable to a single pointer dereference.

**Q: How do I limit trace size?**

Use `WithMaxEvents(n)` when creating the tracer. Once the cap is reached, new events are silently dropped. Track metadata (`Track()` calls) is not counted toward the limit.

---

## Design Decisions

| Decision | Rationale |
|---|---|
| Context-only, no globals | Traces are scoped to a request. No shared mutable state, no cleanup needed. |
| `sync.Mutex` over `sync.RWMutex` | `Collect()` is called once per request; `Record()` is called many times. `RWMutex` adds atomic reader-count overhead on every `Record()` for a read-side benefit that's exercised once. |
| Shallow `cloneMap` | Deep copy requires reflection. Shallow copy isolates the map structure (callers can't add/remove keys) while keeping the common case fast. |
| No `slog`/`log` integration | Stepmark collects structured data. What you do with it — log it, return it in an API, send it to Kafka — is your choice. Compose `Collect()` with whatever output you need. |
| Separate `stepmarkhttp` package | Keeps the core library at zero dependencies. You only import the middleware if you need it. |

---

## License

MIT — see [LICENSE](LICENSE).
