package breadcrumb

import (
	"sync"
	"time"
)

const breadcrumbTracerKey = "__breadcrumb_tracer__"

type ProductTrace struct {
	ProductID   string                 `json:"product_id"`
	ProductMeta map[string]interface{} `json:"product_meta,omitempty"`
	Traces      []TraceEvent           `json:"traces"`
}

type TraceEvent struct {
	Stage     string                 `json:"stage"`
	Action    string                 `json:"action"`
	Timestamp time.Time              `json:"timestamp"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

// BreadcrumbParams holds the collected tracing data, ready for
// serialisation into an API response or any other consumer.
type BreadcrumbParams struct {
	ProductTraces map[string]ProductTrace `json:"productTraces"`
	GlobalTraces  []TraceEvent            `json:"globalTraces,omitempty"`
}

type breadcrumbTracer struct {
	mu            sync.RWMutex
	productTraces map[string]*ProductTrace
	globalTraces  []TraceEvent
}
