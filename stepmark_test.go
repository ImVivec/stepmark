package stepmark

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Context lifecycle
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	ctx := context.Background()
	if Enabled(ctx) {
		t.Fatal("bare context must not be enabled")
	}

	traced := New(ctx)
	if !Enabled(traced) {
		t.Fatal("context from New must be enabled")
	}
	if ctx == traced {
		t.Fatal("New must return a new context")
	}
}

func TestEnabledFalse(t *testing.T) {
	if Enabled(context.Background()) {
		t.Fatal("bare Background must not be enabled")
	}
	if Enabled(context.TODO()) {
		t.Fatal("bare TODO must not be enabled")
	}
}

// ---------------------------------------------------------------------------
// Track
// ---------------------------------------------------------------------------

func TestTrack(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "item_1", map[string]any{"name": "Widget", "price": 9.99})

	trace := Collect(ctx)
	et, ok := trace.Entities["item_1"]
	if !ok {
		t.Fatal("entity item_1 not found")
	}
	if et.EntityID != "item_1" {
		t.Fatalf("expected entity_id item_1, got %s", et.EntityID)
	}
	if et.Meta["name"] != "Widget" {
		t.Fatalf("expected meta name Widget, got %v", et.Meta["name"])
	}
	if len(et.Events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(et.Events))
	}
}

func TestTrackMerge(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", map[string]any{"a": 1})
	Track(ctx, "e1", map[string]any{"b": 2})

	trace := Collect(ctx)
	et := trace.Entities["e1"]
	if et.Meta["a"] != 1 {
		t.Error("original key should be preserved")
	}
	if et.Meta["b"] != 2 {
		t.Error("new key should be merged")
	}
}

func TestTrackOverwrite(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", map[string]any{"v": 1})
	Track(ctx, "e1", map[string]any{"v": 2})

	trace := Collect(ctx)
	if trace.Entities["e1"].Meta["v"] != 2 {
		t.Error("re-tracked key should overwrite")
	}
}

func TestTrackNilMeta(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", nil)

	trace := Collect(ctx)
	if trace.Entities["e1"].Meta != nil {
		t.Error("nil meta should stay nil")
	}
}

func TestTrackNilThenMeta(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", nil)
	Track(ctx, "e1", map[string]any{"k": "v"})

	trace := Collect(ctx)
	if trace.Entities["e1"].Meta["k"] != "v" {
		t.Error("meta should be set when tracking nil then non-nil")
	}
}

func TestTrackMetaThenNil(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", map[string]any{"k": "v"})
	Track(ctx, "e1", nil)

	trace := Collect(ctx)
	if trace.Entities["e1"].Meta["k"] != "v" {
		t.Error("tracking with nil meta should not clear existing meta")
	}
}

func TestTrackDisabled(t *testing.T) {
	Track(context.Background(), "e1", map[string]any{"k": "v"})
}

// ---------------------------------------------------------------------------
// RecordEntity
// ---------------------------------------------------------------------------

func TestRecordEntity(t *testing.T) {
	ctx := New(context.Background())
	meta := map[string]any{"filter": "category"}

	before := time.Now().UTC()
	RecordEntity(ctx, "item_1", "search", "filter_applied", meta)
	after := time.Now().UTC()

	trace := Collect(ctx)
	et := trace.Entities["item_1"]
	if et.EntityID != "item_1" {
		t.Fatalf("expected item_1, got %s", et.EntityID)
	}
	if len(et.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(et.Events))
	}

	ev := et.Events[0]
	if ev.Stage != "search" || ev.Action != "filter_applied" {
		t.Fatalf("stage/action mismatch: %s/%s", ev.Stage, ev.Action)
	}
	if ev.Meta["filter"] != "category" {
		t.Fatalf("meta mismatch: %v", ev.Meta)
	}
	if ev.Timestamp.Before(before) || ev.Timestamp.After(after) {
		t.Fatal("timestamp out of range")
	}
}

func TestRecordEntityAutoCreates(t *testing.T) {
	ctx := New(context.Background())
	RecordEntity(ctx, "auto", "stage", "action", nil)

	trace := Collect(ctx)
	et, ok := trace.Entities["auto"]
	if !ok {
		t.Fatal("RecordEntity should auto-create the entity")
	}
	if et.Meta != nil {
		t.Error("auto-created entity should have nil meta")
	}
	if len(et.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(et.Events))
	}
}

func TestRecordEntityMultiple(t *testing.T) {
	ctx := New(context.Background())
	id := "order_1"

	RecordEntity(ctx, id, "validation", "started", nil)
	RecordEntity(ctx, id, "validation", "passed", nil)
	RecordEntity(ctx, id, "payment", "charged", map[string]any{"amount": 42})

	trace := Collect(ctx)
	events := trace.Entities[id].Events
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	expected := []struct{ stage, action string }{
		{"validation", "started"},
		{"validation", "passed"},
		{"payment", "charged"},
	}
	for i, want := range expected {
		if events[i].Stage != want.stage || events[i].Action != want.action {
			t.Errorf("event %d: want %s/%s, got %s/%s",
				i, want.stage, want.action, events[i].Stage, events[i].Action)
		}
	}
}

func TestRecordEntityDisabled(t *testing.T) {
	RecordEntity(context.Background(), "e1", "s", "a", nil)
}

// ---------------------------------------------------------------------------
// Record (unscoped)
// ---------------------------------------------------------------------------

func TestRecord(t *testing.T) {
	ctx := New(context.Background())
	Record(ctx, "search", "query_received", map[string]any{"q": "milk"})

	trace := Collect(ctx)
	if len(trace.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(trace.Events))
	}
	ev := trace.Events[0]
	if ev.Stage != "search" || ev.Action != "query_received" {
		t.Fatalf("stage/action mismatch: %s/%s", ev.Stage, ev.Action)
	}
	if ev.Meta["q"] != "milk" {
		t.Fatalf("meta mismatch: %v", ev.Meta)
	}
}

func TestRecordMultiple(t *testing.T) {
	ctx := New(context.Background())
	Record(ctx, "search", "started", nil)
	Record(ctx, "search", "filtered", map[string]any{"filters": 3})
	Record(ctx, "ranking", "applied", nil)

	trace := Collect(ctx)
	if len(trace.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(trace.Events))
	}
	for i := 1; i < len(trace.Events); i++ {
		if trace.Events[i].Timestamp.Before(trace.Events[i-1].Timestamp) {
			t.Error("events should be in chronological order")
		}
	}
}

func TestRecordNilMeta(t *testing.T) {
	ctx := New(context.Background())
	Record(ctx, "s", "a", nil)

	trace := Collect(ctx)
	if trace.Events[0].Meta != nil {
		t.Error("nil meta should remain nil")
	}
}

func TestRecordDisabled(t *testing.T) {
	Record(context.Background(), "s", "a", map[string]any{"k": "v"})
}

// ---------------------------------------------------------------------------
// Collect
// ---------------------------------------------------------------------------

func TestCollectDisabled(t *testing.T) {
	if Collect(context.Background()) != nil {
		t.Fatal("Collect on bare context must return nil")
	}
}

func TestCollectEmpty(t *testing.T) {
	ctx := New(context.Background())
	trace := Collect(ctx)
	if trace == nil {
		t.Fatal("Collect on enabled context must not return nil")
	}
	if trace.Entities != nil {
		t.Error("empty entities should be nil")
	}
	if trace.Events != nil {
		t.Error("empty events should be nil")
	}
}

func TestCollectDeepCopy(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", map[string]any{"orig": "val"})
	RecordEntity(ctx, "e1", "s", "a", map[string]any{"trace": "data"})

	t1 := Collect(ctx)
	t2 := Collect(ctx)

	// Entity maps are independent.
	t1.Entities["e1_injected"] = EntityTrace{EntityID: "injected"}
	if _, ok := t2.Entities["e1_injected"]; ok {
		t.Error("mutation of t1 entities map must not affect t2")
	}

	// Entity meta maps are independent copies.
	et1 := t1.Entities["e1"]
	et2 := t2.Entities["e1"]
	et1.Meta["mutated"] = true
	if et2.Meta["mutated"] != nil {
		t.Error("mutation of t1 entity meta must not affect t2")
	}

	// Event meta maps are independent copies.
	et1.Events[0].Meta["injected"] = true
	if et2.Events[0].Meta["injected"] != nil {
		t.Error("mutation of t1 event meta must not affect t2")
	}

	// Original values preserved.
	if et2.Meta["orig"] != "val" {
		t.Error("original meta should be preserved")
	}
	if et2.Events[0].Meta["trace"] != "data" {
		t.Error("original event meta should be preserved")
	}
}

func TestCollectEntitiesAndEvents(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "order_1", map[string]any{"customer": "alice"})
	RecordEntity(ctx, "order_1", "validation", "passed", nil)
	Record(ctx, "search", "started", map[string]any{"q": "test"})

	trace := Collect(ctx)
	if len(trace.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(trace.Entities))
	}
	if len(trace.Events) != 1 {
		t.Fatalf("expected 1 unscoped event, got %d", len(trace.Events))
	}
	if trace.Entities["order_1"].Meta["customer"] != "alice" {
		t.Error("entity meta mismatch")
	}
	if trace.Events[0].Meta["q"] != "test" {
		t.Error("event meta mismatch")
	}
}

func TestCollectDataIsolation(t *testing.T) {
	ctx := New(context.Background())

	orig := map[string]any{"counter": 5, "list": []string{"a", "b"}}
	Record(ctx, "test", "isolation", orig)

	orig["counter"] = 99
	orig["new_key"] = "added"

	trace := Collect(ctx)
	ev := trace.Events[0]
	if ev.Meta["counter"] != 5 {
		t.Errorf("counter should be 5, got %v", ev.Meta["counter"])
	}
	if _, ok := ev.Meta["new_key"]; ok {
		t.Error("new_key should not exist in stored meta")
	}

	ev.Meta["counter"] = 999
	trace2 := Collect(ctx)
	if trace2.Events[0].Meta["counter"] != 5 {
		t.Error("mutating collected output must not affect internal state")
	}
}

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

func TestWithMaxEvents(t *testing.T) {
	ctx := New(context.Background(), WithMaxEvents(3))
	Record(ctx, "s", "a1", nil)
	Record(ctx, "s", "a2", nil)
	RecordEntity(ctx, "e1", "s", "a3", nil)
	Record(ctx, "s", "a4_dropped", nil)
	RecordEntity(ctx, "e1", "s", "a5_dropped", nil)

	trace := Collect(ctx)
	entityEvents := 0
	for _, et := range trace.Entities {
		entityEvents += len(et.Events)
	}
	total := len(trace.Events) + entityEvents
	if total != 3 {
		t.Fatalf("expected 3 total events, got %d", total)
	}
}

func TestWithMaxEventsZeroMeansUnlimited(t *testing.T) {
	ctx := New(context.Background(), WithMaxEvents(0))
	for i := 0; i < 1000; i++ {
		Record(ctx, "s", fmt.Sprintf("a%d", i), nil)
	}

	trace := Collect(ctx)
	if len(trace.Events) != 1000 {
		t.Fatalf("expected 1000 events with maxEvents=0, got %d", len(trace.Events))
	}
}

func TestWithClock(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := New(context.Background(), WithClock(func() time.Time { return fixed }))

	Record(ctx, "s", "a", nil)
	RecordEntity(ctx, "e1", "s", "a", nil)

	trace := Collect(ctx)
	if !trace.Events[0].Timestamp.Equal(fixed) {
		t.Error("unscoped event should use custom clock")
	}
	if !trace.Entities["e1"].Events[0].Timestamp.Equal(fixed) {
		t.Error("entity event should use custom clock")
	}
}

// ---------------------------------------------------------------------------
// Kind / Scope
// ---------------------------------------------------------------------------

func TestTrackWithKind(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "p_1", map[string]any{"name": "Widget"}, WithKind("product"))
	Track(ctx, "o_1", map[string]any{"total": 99.99}, WithKind("order"))
	RecordEntity(ctx, "p_1", "ranking", "scored", nil)
	RecordEntity(ctx, "o_1", "validation", "passed", nil)

	trace := Collect(ctx)
	if trace.Entities["p_1"].Kind != "product" {
		t.Errorf("expected kind 'product', got '%s'", trace.Entities["p_1"].Kind)
	}
	if trace.Entities["o_1"].Kind != "order" {
		t.Errorf("expected kind 'order', got '%s'", trace.Entities["o_1"].Kind)
	}
}

func TestTrackWithoutKind(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", nil)

	trace := Collect(ctx)
	if trace.Entities["e1"].Kind != "" {
		t.Error("kind should be empty when WithKind is not used")
	}
}

func TestTrackKindOnRetrack(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", nil)
	Track(ctx, "e1", nil, WithKind("order"))

	trace := Collect(ctx)
	if trace.Entities["e1"].Kind != "order" {
		t.Error("kind should be settable on re-track")
	}
}

func TestKindJSON(t *testing.T) {
	et := EntityTrace{EntityID: "p_1", Kind: "product", Events: []Event{}}
	data, _ := json.Marshal(et)
	if !strings.Contains(string(data), `"kind":"product"`) {
		t.Error("kind should appear in JSON")
	}

	et2 := EntityTrace{EntityID: "p_2", Events: []Event{}}
	data, _ = json.Marshal(et2)
	if strings.Contains(string(data), "kind") {
		t.Error("empty kind should be omitted from JSON")
	}
}

func TestScope(t *testing.T) {
	ctx := New(context.Background())
	products := NewScope(ctx, "product")
	products.Track("p_1", map[string]any{"name": "Widget"})
	products.Track("p_2", map[string]any{"name": "Gadget"})
	products.RecordEvent("p_1", "ranking", "scored", map[string]any{"score": 0.9})
	products.RecordEvent("p_2", "ranking", "scored", map[string]any{"score": 0.7})

	trace := Collect(ctx)
	if len(trace.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(trace.Entities))
	}
	if trace.Entities["p_1"].Kind != "product" {
		t.Error("p_1 should have kind 'product'")
	}
	if trace.Entities["p_2"].Kind != "product" {
		t.Error("p_2 should have kind 'product'")
	}
	if len(trace.Entities["p_1"].Events) != 1 {
		t.Error("expected 1 event for p_1")
	}
}

func TestMultipleScopes(t *testing.T) {
	ctx := New(context.Background())

	products := NewScope(ctx, "product")
	orders := NewScope(ctx, "order")

	products.Track("p_1", map[string]any{"name": "Widget"})
	products.RecordEvent("p_1", "search", "found", nil)

	orders.Track("o_1", map[string]any{"total": 42})
	orders.RecordEvent("o_1", "validation", "passed", nil)

	Record(ctx, "request", "processed", nil)

	trace := Collect(ctx)
	if len(trace.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(trace.Entities))
	}
	if trace.Entities["p_1"].Kind != "product" {
		t.Error("p_1 should be product")
	}
	if trace.Entities["o_1"].Kind != "order" {
		t.Error("o_1 should be order")
	}
	if len(trace.Events) != 1 {
		t.Error("expected 1 unscoped event")
	}
}

func TestScopeDisabled(t *testing.T) {
	products := NewScope(context.Background(), "product")
	products.Track("p_1", nil)
	products.RecordEvent("p_1", "s", "a", nil)
}

func TestScopeStep(t *testing.T) {
	ctx := New(context.Background())
	products := NewScope(ctx, "product")
	products.Track("p_1", nil)
	products.Step("p_1", "scored", map[string]any{"score": 0.95})

	trace := Collect(ctx)
	et := trace.Entities["p_1"]
	if len(et.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(et.Events))
	}
	if et.Events[0].Stage != "TestScopeStep" {
		t.Errorf("expected stage 'TestScopeStep', got '%s'", et.Events[0].Stage)
	}
	if et.Events[0].Action != "scored" {
		t.Errorf("expected action 'scored', got '%s'", et.Events[0].Action)
	}
}

func TestScopeStepDisabled(t *testing.T) {
	products := NewScope(context.Background(), "product")
	products.Step("p_1", "noop", nil)
}

func TestScopeStepFiltered(t *testing.T) {
	ctx := New(context.Background(), WithEntityFilter(func(id string) bool {
		return id == "p_1"
	}))
	products := NewScope(ctx, "product")
	products.Step("p_1", "ok", nil)
	products.Step("p_2", "blocked", nil)

	trace := Collect(ctx)
	if len(trace.Entities) != 1 {
		t.Fatalf("filter should apply to Scope.Step; got %d entities", len(trace.Entities))
	}
}

func TestScopeEnter(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	callNum := 0
	ctx := New(context.Background(), WithClock(func() time.Time {
		callNum++
		return fixed.Add(time.Duration(callNum) * 50 * time.Millisecond)
	}))

	products := NewScope(ctx, "product")
	products.Track("p_1", nil)
	exit := products.Enter("p_1", nil)
	exit()

	trace := Collect(ctx)
	et := trace.Entities["p_1"]
	if len(et.Events) != 2 {
		t.Fatalf("expected 2 events (enter+exit), got %d", len(et.Events))
	}
	if et.Events[0].Stage != "TestScopeEnter" {
		t.Errorf("expected stage 'TestScopeEnter', got '%s'", et.Events[0].Stage)
	}
	if et.Events[0].Action != "entered" {
		t.Error("first event should be 'entered'")
	}
	if et.Events[1].Action != "exited" {
		t.Error("second event should be 'exited'")
	}
	dur, ok := et.Events[1].Meta["duration_ms"].(float64)
	if !ok || dur <= 0 {
		t.Errorf("exit event should have positive duration_ms, got %v", et.Events[1].Meta["duration_ms"])
	}
}

func TestScopeEnterDisabled(t *testing.T) {
	products := NewScope(context.Background(), "product")
	exit := products.Enter("p_1", nil)
	exit()
}

func TestScopeEnterFiltered(t *testing.T) {
	ctx := New(context.Background(), WithEntityFilter(func(id string) bool {
		return id == "p_1"
	}))
	products := NewScope(ctx, "product")
	exitWanted := products.Enter("p_1", nil)
	exitWanted()
	exitBlocked := products.Enter("p_2", nil)
	exitBlocked()

	trace := Collect(ctx)
	if len(trace.Entities) != 1 {
		t.Fatalf("filter should apply to Scope.Enter; got %d entities", len(trace.Entities))
	}
}

// ---------------------------------------------------------------------------
// Trace metadata
// ---------------------------------------------------------------------------

func TestWithTraceMeta(t *testing.T) {
	meta := map[string]any{"request_id": "req_123", "user_id": "u_456"}
	ctx := New(context.Background(), WithTraceMeta(meta))

	trace := Collect(ctx)
	if trace.Meta["request_id"] != "req_123" {
		t.Error("expected request_id in trace meta")
	}
	if trace.Meta["user_id"] != "u_456" {
		t.Error("expected user_id in trace meta")
	}
}

func TestWithTraceMetaIsolation(t *testing.T) {
	orig := map[string]any{"k": "v"}
	ctx := New(context.Background(), WithTraceMeta(orig))

	orig["k"] = "mutated"

	trace := Collect(ctx)
	if trace.Meta["k"] != "v" {
		t.Error("trace meta should be cloned from input")
	}
}

func TestWithTraceMetaDeepCopy(t *testing.T) {
	ctx := New(context.Background(), WithTraceMeta(map[string]any{"k": "v"}))
	t1 := Collect(ctx)
	t1.Meta["injected"] = true
	t2 := Collect(ctx)
	if t2.Meta["injected"] != nil {
		t.Error("mutating collected trace meta must not affect internal state")
	}
}

func TestWithTraceMetaNil(t *testing.T) {
	ctx := New(context.Background())
	trace := Collect(ctx)
	if trace.Meta != nil {
		t.Error("trace meta should be nil when WithTraceMeta is not used")
	}
}

func TestWithTraceMetaJSON(t *testing.T) {
	ctx := New(context.Background(), WithTraceMeta(map[string]any{"req": "123"}))
	Record(ctx, "s", "a", nil)

	trace := Collect(ctx)
	data, err := json.Marshal(trace)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"meta"`) {
		t.Error("trace meta should appear in JSON")
	}
	if !strings.Contains(s, `"req"`) {
		t.Error("trace meta key should appear in JSON")
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestConcurrentRecordEntity(t *testing.T) {
	ctx := New(context.Background())
	const goroutines = 50
	const eventsPerG = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			entityID := fmt.Sprintf("entity_%d", id)
			for j := 0; j < eventsPerG; j++ {
				RecordEntity(ctx, entityID, "stage", "action", map[string]any{"j": j})
			}
		}(i)
	}
	wg.Wait()

	trace := Collect(ctx)
	if len(trace.Entities) != goroutines {
		t.Fatalf("expected %d entities, got %d", goroutines, len(trace.Entities))
	}
	for id, et := range trace.Entities {
		if len(et.Events) != eventsPerG {
			t.Errorf("entity %s: expected %d events, got %d", id, eventsPerG, len(et.Events))
		}
	}
}

func TestConcurrentRecord(t *testing.T) {
	ctx := New(context.Background())
	const goroutines = 50
	const eventsPerG = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerG; j++ {
				Record(ctx, "stage", "action", map[string]any{"id": id, "j": j})
			}
		}(i)
	}
	wg.Wait()

	trace := Collect(ctx)
	expected := goroutines * eventsPerG
	if len(trace.Events) != expected {
		t.Fatalf("expected %d events, got %d", expected, len(trace.Events))
	}
}

func TestConcurrentMixed(t *testing.T) {
	ctx := New(context.Background())
	const goroutines = 100
	const opsPerG = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			entityID := fmt.Sprintf("entity_%d", id)
			for j := 0; j < opsPerG; j++ {
				meta := map[string]any{"id": id, "op": j}
				Track(ctx, entityID, meta)
				RecordEntity(ctx, entityID, "s", "a", meta)
				Record(ctx, "s", "a", meta)
				Collect(ctx)
			}
		}(i)
	}
	wg.Wait()

	trace := Collect(ctx)
	if trace == nil {
		t.Fatal("trace should not be nil")
	}
	if len(trace.Entities) != goroutines {
		t.Errorf("expected %d entities, got %d", goroutines, len(trace.Entities))
	}
	expectedEvents := goroutines * opsPerG
	if len(trace.Events) != expectedEvents {
		t.Errorf("expected %d unscoped events, got %d", expectedEvents, len(trace.Events))
	}
}

// ---------------------------------------------------------------------------
// JSON serialization
// ---------------------------------------------------------------------------

func TestEventJSON(t *testing.T) {
	ev := Event{
		Stage:     "checkout",
		Action:    "paid",
		Timestamp: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Meta:      map[string]any{"amount": 42.5, "currency": "USD"},
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}

	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Stage != ev.Stage || got.Action != ev.Action {
		t.Error("stage/action mismatch after round-trip")
	}
	if !got.Timestamp.Equal(ev.Timestamp) {
		t.Error("timestamp mismatch after round-trip")
	}
	if got.Meta["currency"] != "USD" {
		t.Error("meta mismatch after round-trip")
	}
}

func TestEntityTraceJSON(t *testing.T) {
	et := EntityTrace{
		EntityID: "order_42",
		Meta:     map[string]any{"customer": "bob"},
		Events: []Event{
			{Stage: "validation", Action: "passed", Timestamp: time.Now().UTC()},
		},
	}

	data, err := json.Marshal(et)
	if err != nil {
		t.Fatal(err)
	}

	var got EntityTrace
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.EntityID != et.EntityID {
		t.Error("entity_id mismatch")
	}
	if len(got.Events) != 1 {
		t.Error("events count mismatch")
	}
	if got.Meta["customer"] != "bob" {
		t.Error("meta mismatch")
	}
}

func TestTraceJSON(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", map[string]any{"k": "v"})
	RecordEntity(ctx, "e1", "s", "a", nil)
	Record(ctx, "global_s", "global_a", map[string]any{"q": "test"})

	trace := Collect(ctx)
	data, err := json.Marshal(trace)
	if err != nil {
		t.Fatal(err)
	}

	var got Trace
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Entities) != 1 {
		t.Errorf("expected 1 entity, got %d", len(got.Entities))
	}
	if len(got.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(got.Events))
	}
}

func TestJSONOmitEmpty(t *testing.T) {
	ev := Event{Stage: "s", Action: "a", Timestamp: time.Now().UTC()}
	data, _ := json.Marshal(ev)
	if strings.Contains(string(data), "meta") {
		t.Error("nil meta should be omitted from JSON")
	}

	et := EntityTrace{EntityID: "e1", Events: []Event{}}
	data, _ = json.Marshal(et)
	if strings.Contains(string(data), `"meta"`) {
		t.Error("nil entity meta should be omitted from JSON")
	}

	tr := Trace{}
	data, _ = json.Marshal(tr)
	s := string(data)
	if strings.Contains(s, `"meta"`) {
		t.Error("nil trace meta should be omitted")
	}
	if strings.Contains(s, "entities") {
		t.Error("nil entities should be omitted")
	}
	if strings.Contains(s, "events") {
		t.Error("nil events should be omitted")
	}
}

// ---------------------------------------------------------------------------
// Entity filter
// ---------------------------------------------------------------------------

func TestWithEntityFilter(t *testing.T) {
	ctx := New(context.Background(), WithEntityFilter(func(id string) bool {
		return id == "wanted"
	}))

	Track(ctx, "wanted", map[string]any{"ok": true})
	Track(ctx, "blocked", map[string]any{"nope": true})
	RecordEntity(ctx, "wanted", "s", "a", nil)
	RecordEntity(ctx, "blocked", "s", "a", nil)
	Record(ctx, "unscoped", "always", nil)

	trace := Collect(ctx)
	if len(trace.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(trace.Entities))
	}
	if _, ok := trace.Entities["wanted"]; !ok {
		t.Error("wanted entity should exist")
	}
	if _, ok := trace.Entities["blocked"]; ok {
		t.Error("blocked entity should not exist")
	}
	if len(trace.Events) != 1 {
		t.Error("unscoped events must not be filtered")
	}
}

func TestWithEntityFilterNil(t *testing.T) {
	ctx := New(context.Background())
	Track(ctx, "e1", nil)
	RecordEntity(ctx, "e2", "s", "a", nil)

	trace := Collect(ctx)
	if len(trace.Entities) != 2 {
		t.Fatalf("nil filter should allow all entities; got %d", len(trace.Entities))
	}
}

func TestWithEntityFilterAcceptAll(t *testing.T) {
	ctx := New(context.Background(), WithEntityFilter(func(string) bool { return true }))
	Track(ctx, "a", nil)
	Track(ctx, "b", nil)

	trace := Collect(ctx)
	if len(trace.Entities) != 2 {
		t.Fatalf("accept-all filter should allow all entities; got %d", len(trace.Entities))
	}
}

func TestWithEntityFilterRejectAll(t *testing.T) {
	ctx := New(context.Background(), WithEntityFilter(func(string) bool { return false }))
	Track(ctx, "a", nil)
	RecordEntity(ctx, "b", "s", "a", nil)

	trace := Collect(ctx)
	if len(trace.Entities) != 0 {
		t.Fatalf("reject-all filter should block all entities; got %d", len(trace.Entities))
	}
}

func TestWithEntityFilterScope(t *testing.T) {
	ctx := New(context.Background(), WithEntityFilter(func(id string) bool {
		return id == "p_1"
	}))
	products := NewScope(ctx, "product")
	products.Track("p_1", nil)
	products.Track("p_2", nil)
	products.RecordEvent("p_1", "s", "a", nil)
	products.RecordEvent("p_2", "s", "a", nil)

	trace := Collect(ctx)
	if len(trace.Entities) != 1 {
		t.Fatalf("filter should apply through Scope; got %d entities", len(trace.Entities))
	}
}

// ---------------------------------------------------------------------------
// Auto-instrumentation
// ---------------------------------------------------------------------------

func TestStep(t *testing.T) {
	ctx := New(context.Background())
	Step(ctx, "started", nil)

	trace := Collect(ctx)
	if len(trace.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(trace.Events))
	}
	if trace.Events[0].Stage != "TestStep" {
		t.Errorf("expected stage 'TestStep', got '%s'", trace.Events[0].Stage)
	}
	if trace.Events[0].Action != "started" {
		t.Errorf("expected action 'started', got '%s'", trace.Events[0].Action)
	}
}

func TestStepDisabled(t *testing.T) {
	Step(context.Background(), "noop", nil)
}

func TestStepEntity(t *testing.T) {
	ctx := New(context.Background())
	StepEntity(ctx, "order_1", "scored", map[string]any{"score": 0.9})

	trace := Collect(ctx)
	et := trace.Entities["order_1"]
	if len(et.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(et.Events))
	}
	if et.Events[0].Stage != "TestStepEntity" {
		t.Errorf("expected stage 'TestStepEntity', got '%s'", et.Events[0].Stage)
	}
	if et.Events[0].Action != "scored" {
		t.Errorf("expected action 'scored', got '%s'", et.Events[0].Action)
	}
}

func TestStepEntityDisabled(t *testing.T) {
	StepEntity(context.Background(), "e1", "noop", nil)
}

func TestStepEntityFiltered(t *testing.T) {
	ctx := New(context.Background(), WithEntityFilter(func(id string) bool {
		return id == "wanted"
	}))
	StepEntity(ctx, "wanted", "ok", nil)
	StepEntity(ctx, "blocked", "nope", nil)

	trace := Collect(ctx)
	if len(trace.Entities) != 1 {
		t.Fatalf("filter should apply to StepEntity; got %d entities", len(trace.Entities))
	}
}

func TestEnter(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	callNum := 0
	ctx := New(context.Background(), WithClock(func() time.Time {
		callNum++
		return fixed.Add(time.Duration(callNum) * 100 * time.Millisecond)
	}))

	exit := Enter(ctx, map[string]any{"request": true})
	exit()

	trace := Collect(ctx)
	if len(trace.Events) != 2 {
		t.Fatalf("expected 2 events (enter+exit), got %d", len(trace.Events))
	}
	if trace.Events[0].Stage != "TestEnter" {
		t.Errorf("expected stage 'TestEnter', got '%s'", trace.Events[0].Stage)
	}
	if trace.Events[0].Action != "entered" {
		t.Errorf("expected action 'entered', got '%s'", trace.Events[0].Action)
	}
	if trace.Events[1].Action != "exited" {
		t.Errorf("expected action 'exited', got '%s'", trace.Events[1].Action)
	}
	dur, ok := trace.Events[1].Meta["duration_ms"].(float64)
	if !ok || dur <= 0 {
		t.Errorf("exit event should have positive duration_ms, got %v", trace.Events[1].Meta["duration_ms"])
	}
}

func TestEnterDisabled(t *testing.T) {
	exit := Enter(context.Background(), nil)
	exit()
}

func TestEnterEntity(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	callNum := 0
	ctx := New(context.Background(), WithClock(func() time.Time {
		callNum++
		return fixed.Add(time.Duration(callNum) * 50 * time.Millisecond)
	}))

	exit := EnterEntity(ctx, "order_1", nil)
	exit()

	trace := Collect(ctx)
	et := trace.Entities["order_1"]
	if len(et.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(et.Events))
	}
	if et.Events[0].Stage != "TestEnterEntity" {
		t.Errorf("expected stage 'TestEnterEntity', got '%s'", et.Events[0].Stage)
	}
	if et.Events[0].Action != "entered" {
		t.Error("first event should be 'entered'")
	}
	if et.Events[1].Action != "exited" {
		t.Error("second event should be 'exited'")
	}
	dur, ok := et.Events[1].Meta["duration_ms"].(float64)
	if !ok || dur <= 0 {
		t.Errorf("exit event should have positive duration_ms, got %v", et.Events[1].Meta["duration_ms"])
	}
}

func TestEnterEntityDisabled(t *testing.T) {
	exit := EnterEntity(context.Background(), "e1", nil)
	exit()
}

func TestEnterEntityFiltered(t *testing.T) {
	ctx := New(context.Background(), WithEntityFilter(func(id string) bool {
		return id == "wanted"
	}))

	exitWanted := EnterEntity(ctx, "wanted", nil)
	exitWanted()
	exitBlocked := EnterEntity(ctx, "blocked", nil)
	exitBlocked()

	trace := Collect(ctx)
	if len(trace.Entities) != 1 {
		t.Fatalf("filter should apply to EnterEntity; got %d entities", len(trace.Entities))
	}
}

func TestCleanFuncName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"github.com/user/pkg.Function", "Function"},
		{"github.com/user/pkg.(*Type).Method", "Type.Method"},
		{"github.com/user/pkg.Function.func1", "Function.func1"},
		{"main.main", "main"},
		{"pkg.init", "init"},
	}
	for _, tc := range tests {
		got := cleanFuncName(tc.input)
		if got != tc.want {
			t.Errorf("cleanFuncName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func TestCloneMap(t *testing.T) {
	if cloneMap(nil) != nil {
		t.Error("cloneMap(nil) should return nil")
	}

	empty := make(map[string]any)
	c := cloneMap(empty)
	if c == nil || len(c) != 0 {
		t.Error("cloneMap of empty map should return empty non-nil map")
	}

	orig := map[string]any{"a": 1, "b": "two", "c": true}
	clone := cloneMap(orig)

	orig["d"] = "new"
	if _, ok := clone["d"]; ok {
		t.Error("modifying original must not affect clone")
	}

	clone["e"] = "also_new"
	if _, ok := orig["e"]; ok {
		t.Error("modifying clone must not affect original")
	}
}
