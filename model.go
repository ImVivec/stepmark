package stepmark

import "time"

// Event is a single recorded step in a trace.
type Event struct {
	Stage     string         `json:"stage"`
	Action    string         `json:"action"`
	Timestamp time.Time      `json:"timestamp"`
	Meta      map[string]any `json:"meta,omitempty"`
}

// EntityTrace holds the ordered events for a single tracked entity.
type EntityTrace struct {
	EntityID string         `json:"entity_id"`
	Meta     map[string]any `json:"meta,omitempty"`
	Events   []Event        `json:"events"`
}

// Trace is the complete collected output, containing per-entity
// traces and unscoped events not pinned to any entity.
type Trace struct {
	Entities map[string]EntityTrace `json:"entities,omitempty"`
	Events   []Event                `json:"events,omitempty"`
}
