package stepmark

import (
	"sync"
	"time"
)

type contextKey struct{}

type tracer struct {
	mu           sync.Mutex
	meta         map[string]any
	entities     map[string]*entityState
	events       []Event
	clock        func() time.Time
	entityFilter EntityFilter
	maxEvents    int
	count        int
}

type entityState struct {
	id     string
	kind   string
	meta   map[string]any
	events []Event
}

func newTracer(opts []Option) *tracer {
	t := &tracer{
		entities: make(map[string]*entityState),
		clock:    func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *tracer) track(entityID string, meta map[string]any, opts []TrackOption) {
	t.mu.Lock()
	defer t.mu.Unlock()

	es, exists := t.entities[entityID]
	if !exists {
		es = &entityState{
			id:     entityID,
			meta:   cloneMap(meta),
			events: make([]Event, 0, 4),
		}
		for _, opt := range opts {
			opt(es)
		}
		t.entities[entityID] = es
		return
	}
	for _, opt := range opts {
		opt(es)
	}
	if meta == nil {
		return
	}
	if es.meta == nil {
		es.meta = cloneMap(meta)
		return
	}
	for k, v := range meta {
		es.meta[k] = v
	}
}

func (t *tracer) recordEntity(entityID, stage, action string, meta map[string]any) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.maxEvents > 0 && t.count >= t.maxEvents {
		return
	}

	es, exists := t.entities[entityID]
	if !exists {
		es = &entityState{
			id:     entityID,
			events: make([]Event, 0, 4),
		}
		t.entities[entityID] = es
	}
	es.events = append(es.events, Event{
		Stage:     stage,
		Action:    action,
		Timestamp: t.clock(),
		Meta:      cloneMap(meta),
	})
	t.count++
}

func (t *tracer) record(stage, action string, meta map[string]any) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.maxEvents > 0 && t.count >= t.maxEvents {
		return
	}

	t.events = append(t.events, Event{
		Stage:     stage,
		Action:    action,
		Timestamp: t.clock(),
		Meta:      cloneMap(meta),
	})
	t.count++
}

func (t *tracer) collect() *Trace {
	t.mu.Lock()
	defer t.mu.Unlock()

	trace := &Trace{
		Meta: cloneMap(t.meta),
	}

	if len(t.entities) > 0 {
		trace.Entities = make(map[string]EntityTrace, len(t.entities))
		for id, es := range t.entities {
			events := make([]Event, len(es.events))
			for i, e := range es.events {
				events[i] = Event{
					Stage:     e.Stage,
					Action:    e.Action,
					Timestamp: e.Timestamp,
					Meta:      cloneMap(e.Meta),
				}
			}
			trace.Entities[id] = EntityTrace{
				EntityID: es.id,
				Kind:     es.kind,
				Meta:     cloneMap(es.meta),
				Events:   events,
			}
		}
	}

	if len(t.events) > 0 {
		trace.Events = make([]Event, len(t.events))
		for i, e := range t.events {
			trace.Events[i] = Event{
				Stage:     e.Stage,
				Action:    e.Action,
				Timestamp: e.Timestamp,
				Meta:      cloneMap(e.Meta),
			}
		}
	}

	return trace
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
