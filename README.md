# Stepmark

**On-Demand Business Logic Tracer for Go**

[![Go Reference](https://pkg.go.dev/badge/github.com/ImVivec/stepmark.svg)](https://pkg.go.dev/github.com/ImVivec/stepmark)
[![CI](https://github.com/ImVivec/stepmark/actions/workflows/ci.yml/badge.svg)](https://github.com/ImVivec/stepmark/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Standard logs are noisy. APM tools track latency, not logic. Stepmark traces the **decision-making journey** of specific entities — orders, users, search results — through your codebase. On demand.

---

## The Problem

You have a complex backend where an order flows through validation, fraud detection, inventory checks, a pricing engine, and payment. Something goes wrong for a specific order. Now what?

- **Logs?** Grep through millions of lines across a dozen services.
- **APM?** It tells you the request took 200ms. It doesn't tell you *why* the order was rejected.
- **Debugger?** Good luck attaching one to production.

Stepmark gives you a structured audit trail of every decision point, but **only when you ask for it**. When tracing is not enabled, every call is a no-op.

---

## Install

```bash
go get github.com/ImVivec/stepmark
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

    "github.com/ImVivec/stepmark"
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

---

## API

**Core** — the complete tracing lifecycle:

| Function | Purpose |
|---|---|
| `New(ctx, ...Option) context.Context` | Start tracing — injects a tracer into the context |
| `Enabled(ctx) bool` | Check if tracing is active |
| `Track(ctx, entityID, meta, ...TrackOption)` | Register an entity with optional metadata and kind |
| `RecordEntity(ctx, entityID, stage, action, meta)` | Record an event for a specific entity |
| `Record(ctx, stage, action, meta)` | Record an unscoped event (not tied to an entity) |
| `Collect(ctx) *Trace` | Extract a deep-copied snapshot of all recorded data |

**Auto-Instrumentation** — the function name *is* the stage:

| Function | Purpose |
|---|---|
| `Step(ctx, action, meta)` | Record an event using the caller's function name as stage |
| `StepEntity(ctx, entityID, action, meta)` | Record an entity event using the caller's function name |
| `Enter(ctx, meta) func()` | Record function entry/exit with duration (use with `defer`) |
| `EnterEntity(ctx, entityID, meta) func()` | Record entity function entry/exit with duration |

**Helpers** — ergonomics for common patterns:

| Function / Type | Purpose |
|---|---|
| `NewScope(ctx, kind) Scope` | Create a scoped recorder that auto-sets entity kind |
| `Scope.Track(entityID, meta)` | Track an entity within the scope |
| `Scope.RecordEvent(entityID, stage, action, meta)` | Record an event within the scope |
| `Scope.Step(entityID, action, meta)` | Auto-instrumented entity event (caller's name as stage) |
| `Scope.Enter(entityID, meta) func()` | Auto-instrumented entity entry/exit with duration |

**Options:**

| Option | Purpose |
|---|---|
| `WithMaxEvents(n)` | Cap total events to prevent unbounded growth |
| `WithClock(fn)` | Custom time source for deterministic tests |
| `WithTraceMeta(meta)` | Attach request-level metadata to the trace |
| `WithEntityFilter(fn)` | Only trace entities matching a predicate |
| `WithKind(kind)` | Classify an entity by type (used with `Track`) |

**Every function is a no-op when tracing is disabled.** You never need to guard with `if stepmark.Enabled(ctx)` for correctness.

---

## HTTP Middleware

The `stepmarkhttp` subpackage provides ready-made middleware for `net/http` and compatible routers (Chi, gorilla/mux, etc.).

### Standard Library

```go
import "github.com/ImVivec/stepmark/stepmarkhttp"

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

## Auto-Instrumentation

Manually specifying stage names as strings is tedious and error-prone. The auto-instrumentation API uses `runtime.Caller` to derive the stage from the calling function's name — automatically.

### `Step` / `StepEntity` — the function name is the stage

```go
func ValidateOrder(ctx context.Context, order Order) error {
    // stage is automatically set to "ValidateOrder"
    stepmark.Step(ctx, "started", nil)

    if order.Total > 10000 {
        stepmark.Step(ctx, "flagged_high_value", map[string]any{"total": order.Total})
    }
    stepmark.Step(ctx, "passed", nil)
    return nil
}

func ScoreProduct(ctx context.Context, productID string, score float64) {
    // stage is automatically set to "ScoreProduct", scoped to productID
    stepmark.StepEntity(ctx, productID, "scored", map[string]any{"score": score})
}
```

### `Enter` / `EnterEntity` — function boundary tracing with duration

Wrap an entire function with a single `defer` line. Stepmark records "entered" at the top and "exited" (with `duration_ms`) when the function returns:

```go
func ChargePayment(ctx context.Context, orderID string) error {
    defer stepmark.EnterEntity(ctx, orderID, nil)()
    // Records:
    //   { stage: "ChargePayment", action: "entered" }
    //   ... your logic runs ...
    //   { stage: "ChargePayment", action: "exited", meta: { duration_ms: 12.5 } }

    return processPayment(ctx, orderID)
}

func ProcessRequest(ctx context.Context) {
    defer stepmark.Enter(ctx, map[string]any{"path": "/checkout"})()
    // Unscoped enter/exit events with stage = "ProcessRequest"
}
```

### When to use what

| Situation | Use |
|---|---|
| You want a named stage like `"validation"` | `Record` / `RecordEntity` |
| The function name *is* the stage | `Step` / `StepEntity` |
| You want enter/exit with duration | `Enter` / `EnterEntity` |
| You're tracking many entities of the same kind | `Scope` + `RecordEvent` |

---

## Conditional Entity Recording

When you only care about specific entities — a particular order, the top-10 search results, a flagged user — use `WithEntityFilter` to skip the rest:

```go
ctx := stepmark.New(ctx,
    stepmark.WithEntityFilter(func(entityID string) bool {
        return entityID == targetOrderID
    }),
)

// Only events for targetOrderID are recorded.
// All other entity calls are no-ops.
stepmark.RecordEntity(ctx, targetOrderID, "validation", "passed", nil)  // ✓ recorded
stepmark.RecordEntity(ctx, "other_order", "validation", "passed", nil)  // ✗ skipped
stepmark.Record(ctx, "request", "processed", nil)                       // ✓ always recorded
```

The filter applies to all entity-scoped calls: `Track`, `RecordEntity`, `StepEntity`, `EnterEntity`. Unscoped calls (`Record`, `Step`, `Enter`) are never filtered.

```go
// Filter by a set of interesting entities:
interestingIDs := map[string]bool{"order_42": true, "order_99": true}
ctx := stepmark.New(ctx, stepmark.WithEntityFilter(func(id string) bool {
    return interestingIDs[id]
}))

// Filter by kind (using naming convention):
ctx := stepmark.New(ctx, stepmark.WithEntityFilter(func(id string) bool {
    return strings.HasPrefix(id, "order_")
}))
```

---

## Scoped Entity Tracking

When a request touches multiple entity types — products, orders, users — the `Scope` helper groups them by kind without repeating `WithKind` on every call:

```go
func SearchHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    products := stepmark.NewScope(ctx, "product")
    for _, p := range catalog.Search(ctx, query) {
        products.Track(p.ID, map[string]any{"name": p.Name, "category": p.Category})
        products.RecordEvent(p.ID, "ranking", "scored", map[string]any{"score": p.Score})
    }

    users := stepmark.NewScope(ctx, "user")
    users.Track(userID, map[string]any{"tier": "premium"})
    users.RecordEvent(userID, "personalization", "applied", nil)

    stepmark.Record(ctx, "search", "response_ready", map[string]any{"count": len(results)})
}
```

The collected trace groups entities by kind:

```json
{
  "entities": {
    "prod_1": { "entity_id": "prod_1", "kind": "product", "events": [...] },
    "prod_2": { "entity_id": "prod_2", "kind": "product", "events": [...] },
    "u_42":   { "entity_id": "u_42",   "kind": "user",    "events": [...] }
  },
  "events": [
    { "stage": "search", "action": "response_ready", "meta": { "count": 2 } }
  ]
}
```

You can also set kind directly without a scope:

```go
stepmark.Track(ctx, "order_99", meta, stepmark.WithKind("order"))
```

---

## Trace Metadata

Attach request-level context to the trace itself — not to any specific entity or event:

```go
ctx := stepmark.New(ctx, stepmark.WithTraceMeta(map[string]any{
    "request_id":  reqID,
    "user_id":     userID,
    "ab_variant":  "checkout_v2",
}))
```

This appears at the top level of the collected trace:

```json
{
  "meta": { "request_id": "req_abc", "user_id": "u_42", "ab_variant": "checkout_v2" },
  "entities": { ... },
  "events": [ ... ]
}
```

---

## Use Cases

### E-Commerce: Search Ranking Pipeline

Track why each product ended up at its position — across search, ranking, filtering, and personalization:

```go
products := stepmark.NewScope(ctx, "product")
for _, p := range catalogResults {
    products.Track(p.ID, map[string]any{"name": p.Name})
}

// Inside the ranker:
products.RecordEvent(p.ID, "ranking", "ml_scored", map[string]any{
    "score": 0.92, "model": "v3", "features": featureCount,
})

// Inside the filter:
products.RecordEvent(p.ID, "filter", "excluded", map[string]any{
    "reason": "out_of_stock",
})
```

### Order Processing: Audit Trail

Trace every decision an order passes through — from validation to fulfillment:

```go
stepmark.Track(ctx, orderID, map[string]any{"total": 249.99}, stepmark.WithKind("order"))
stepmark.RecordEntity(ctx, orderID, "validation", "passed", nil)
stepmark.RecordEntity(ctx, orderID, "fraud_check", "flagged", map[string]any{
    "score": 0.78, "model": "fraud_v2", "action": "manual_review",
})
stepmark.RecordEntity(ctx, orderID, "inventory", "reserved", map[string]any{
    "warehouse": "us-east-1", "items": 3,
})
```

### ML Inference: Decision Explainability

Trace why a model made a specific prediction — features, thresholds, fallback logic:

```go
stepmark.Track(ctx, predictionID, nil, stepmark.WithKind("prediction"))
stepmark.RecordEntity(ctx, predictionID, "features", "extracted", map[string]any{
    "count": 128, "source": "feature_store_v2",
})
stepmark.RecordEntity(ctx, predictionID, "model", "scored", map[string]any{
    "model": "xgboost_v4", "confidence": 0.91, "latency_ms": 12,
})
stepmark.RecordEntity(ctx, predictionID, "threshold", "passed", map[string]any{
    "min_confidence": 0.85, "action": "auto_approve",
})
```

### Content Moderation Pipeline

Trace why content was approved, flagged, or rejected — across multiple rules and models:

```go
stepmark.Track(ctx, contentID, map[string]any{"type": "comment"}, stepmark.WithKind("content"))
stepmark.RecordEntity(ctx, contentID, "toxicity", "scored", map[string]any{
    "score": 0.12, "model": "perspective_v2",
})
stepmark.RecordEntity(ctx, contentID, "spam", "cleared", map[string]any{
    "score": 0.03, "threshold": 0.5,
})
stepmark.RecordEntity(ctx, contentID, "pii", "detected", map[string]any{
    "fields": []string{"email", "phone"}, "action": "redact",
})
stepmark.RecordEntity(ctx, contentID, "moderation", "approved", map[string]any{
    "auto": true, "policy": "standard_v3",
})
```

### Multi-Service Debugging

Combine Stepmark with trace metadata to correlate across services:

```go
// Service A: API Gateway
ctx := stepmark.New(ctx, stepmark.WithTraceMeta(map[string]any{
    "trace_id":   traceID,
    "service":    "api-gateway",
    "request_id": reqID,
}))
stepmark.Record(ctx, "routing", "backend_selected", map[string]any{
    "backend": "search-v2", "reason": "canary_10pct",
})

// Collect and pass downstream via header, log, or message queue.
// Service B can create its own trace with the same trace_id
// for later correlation.
```

---

## Options

```go
// Cap total events to prevent runaway growth in pathological cases.
ctx := stepmark.New(ctx, stepmark.WithMaxEvents(1000))

// Inject a fixed clock for deterministic tests.
fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
ctx := stepmark.New(ctx, stepmark.WithClock(func() time.Time { return fixed }))

// Attach request-level context.
ctx := stepmark.New(ctx, stepmark.WithTraceMeta(map[string]any{
    "request_id": reqID,
    "user_id":    userID,
}))

// Only trace specific entities.
ctx := stepmark.New(ctx, stepmark.WithEntityFilter(func(id string) bool {
    return id == targetOrderID
}))
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

All events for one tracked entity. `Kind` groups entities by type.

```go
type EntityTrace struct {
    EntityID string         `json:"entity_id"`
    Kind     string         `json:"kind,omitempty"`
    Meta     map[string]any `json:"meta,omitempty"`
    Events   []Event        `json:"events"`
}
```

### Trace

The complete output from `Collect`. Ready for `json.Marshal`.

```go
type Trace struct {
    Meta     map[string]any         `json:"meta,omitempty"`
    Entities map[string]EntityTrace `json:"entities,omitempty"`
    Events   []Event                `json:"events,omitempty"`
}
```

---

## FAQ

**Q: Do I need to call `Enabled()` before `Record()`?**

No. Every function is a no-op when tracing is disabled. Just call `Record` directly.

**Q: Is it safe to use from multiple goroutines?**

Yes. The tracer uses a mutex internally. All functions are safe for concurrent use as long as they share the same context.

**Q: What happens if I call `Collect()` multiple times?**

Each call returns an independent deep copy. Calling `Collect()` does not consume or reset the tracer. Events recorded between calls appear in subsequent snapshots.

**Q: How do I limit trace size?**

Use `WithMaxEvents(n)` when creating the tracer. Once the cap is reached, new events are silently dropped. Track metadata (`Track()` calls) is not counted toward the limit.

**Q: When should I use `Step` vs `Record`?**

Use `Record` when you want a specific, human-readable stage name like `"validation"` or `"fraud_check"`. Use `Step` when the function name itself is the stage — it saves you from typing the same string you'd copy from the function declaration. Use `Enter`/`EnterEntity` when you want automatic enter/exit boundary events with duration.

---

## Design Decisions

| Decision | Rationale |
|---|---|
| Context-only, no globals | Traces are scoped to a request. No shared mutable state, no cleanup needed. |
| `sync.Mutex` over `sync.RWMutex` | `Collect()` is called once per request; `Record()` is called many times. A write-biased lock is the right fit. |
| Shallow `cloneMap` | Deep copy requires reflection. Shallow copy isolates the map structure (callers can't add/remove keys) which is sufficient for the common case. |
| `runtime.Caller` gated by nil-check | `Step`/`Enter` only call `runtime.Caller` when tracing is enabled. When disabled, the nil-check exits before touching the runtime. |
| Entity filter before lock | `WithEntityFilter` is checked before acquiring the mutex. The filter is set once at creation and never mutated, making the unsynchronized read safe. |
| No `slog`/`log` integration | Stepmark collects structured data. What you do with it — log it, return it in an API, send it to Kafka — is your choice. Compose `Collect()` with whatever output you need. |
| Separate `stepmarkhttp` package | Keeps the core library at zero dependencies. You only import the middleware if you need it. |

---

## Benchmarks

When tracing is disabled, every call is a nil-check on `context.Value()` — no allocations, no locks.

```
goos: darwin / goarch: arm64 / cpu: Apple M4

Disabled path (normal production traffic):
  BenchmarkRecord_Disabled              1.78 ns/op    0 B/op   0 allocs/op
  BenchmarkRecordEntity_Disabled        1.86 ns/op    0 B/op   0 allocs/op
  BenchmarkStep_Disabled                1.96 ns/op    0 B/op   0 allocs/op
  BenchmarkEnter_Disabled               1.98 ns/op    0 B/op   0 allocs/op

Enabled path (active tracing):
  BenchmarkRecord_Enabled               234  ns/op    682 B/op 2 allocs/op
  BenchmarkRecordEntity_Enabled         242  ns/op    724 B/op 3 allocs/op
  BenchmarkStep_Enabled                 327  ns/op    596 B/op 2 allocs/op
```

Run `make bench` to reproduce.

---

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
git clone https://github.com/ImVivec/stepmark.git
cd stepmark
make check   # fmt + vet + tests
```

---

## License

MIT — see [LICENSE](LICENSE).
