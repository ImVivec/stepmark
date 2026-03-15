package breadcrumb

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProductTraceJSONSerialization(t *testing.T) {
	productTrace := ProductTrace{
		ProductID: "test_product_123",
		ProductMeta: map[string]interface{}{
			"name":  "Test Product",
			"price": 99.99,
			"tags":  []string{"electronics", "gadgets"},
		},
		Traces: []TraceEvent{
			{
				Stage:     "search",
				Action:    "viewed",
				Timestamp: time.Date(2023, 12, 1, 10, 30, 0, 0, time.UTC),
				Meta: map[string]interface{}{
					"query": "smartphone",
					"page":  1,
				},
			},
		},
	}

	jsonData, err := json.Marshal(productTrace)
	if err != nil {
		t.Fatalf("Failed to marshal ProductTrace: %v", err)
	}

	var unmarshaledTrace ProductTrace
	err = json.Unmarshal(jsonData, &unmarshaledTrace)
	if err != nil {
		t.Fatalf("Failed to unmarshal ProductTrace: %v", err)
	}

	if unmarshaledTrace.ProductID != productTrace.ProductID {
		t.Errorf("ProductID mismatch: expected %s, got %s",
			productTrace.ProductID, unmarshaledTrace.ProductID)
	}

	if len(unmarshaledTrace.Traces) != len(productTrace.Traces) {
		t.Errorf("Traces length mismatch: expected %d, got %d",
			len(productTrace.Traces), len(unmarshaledTrace.Traces))
	}

	if len(unmarshaledTrace.Traces) > 0 {
		originalEvent := productTrace.Traces[0]
		unmarshaledEvent := unmarshaledTrace.Traces[0]

		if unmarshaledEvent.Stage != originalEvent.Stage {
			t.Errorf("Stage mismatch: expected %s, got %s",
				originalEvent.Stage, unmarshaledEvent.Stage)
		}

		if unmarshaledEvent.Action != originalEvent.Action {
			t.Errorf("Action mismatch: expected %s, got %s",
				originalEvent.Action, unmarshaledEvent.Action)
		}

		if !unmarshaledEvent.Timestamp.Equal(originalEvent.Timestamp) {
			t.Errorf("Timestamp mismatch: expected %v, got %v",
				originalEvent.Timestamp, unmarshaledEvent.Timestamp)
		}
	}
}

func TestTraceEventJSONSerialization(t *testing.T) {
	traceEvent := TraceEvent{
		Stage:     "checkout",
		Action:    "payment_completed",
		Timestamp: time.Date(2023, 12, 1, 15, 45, 30, 0, time.UTC),
		Meta: map[string]interface{}{
			"payment_method": "credit_card",
			"amount":         149.99,
			"currency":       "USD",
		},
	}

	jsonData, err := json.Marshal(traceEvent)
	if err != nil {
		t.Fatalf("Failed to marshal TraceEvent: %v", err)
	}

	var unmarshaledEvent TraceEvent
	err = json.Unmarshal(jsonData, &unmarshaledEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal TraceEvent: %v", err)
	}

	if unmarshaledEvent.Stage != traceEvent.Stage {
		t.Errorf("Stage mismatch: expected %s, got %s",
			traceEvent.Stage, unmarshaledEvent.Stage)
	}

	if unmarshaledEvent.Action != traceEvent.Action {
		t.Errorf("Action mismatch: expected %s, got %s",
			traceEvent.Action, unmarshaledEvent.Action)
	}

	if !unmarshaledEvent.Timestamp.Equal(traceEvent.Timestamp) {
		t.Errorf("Timestamp mismatch: expected %v, got %v",
			traceEvent.Timestamp, unmarshaledEvent.Timestamp)
	}

	if unmarshaledEvent.Meta["payment_method"] != "credit_card" {
		t.Errorf("Meta payment_method mismatch: expected credit_card, got %v",
			unmarshaledEvent.Meta["payment_method"])
	}

	if unmarshaledEvent.Meta["amount"] != 149.99 {
		t.Errorf("Meta amount mismatch: expected 149.99, got %v",
			unmarshaledEvent.Meta["amount"])
	}
}

func TestProductTraceWithNilMeta(t *testing.T) {
	productTrace := ProductTrace{
		ProductID:   "test_product",
		ProductMeta: nil,
		Traces: []TraceEvent{
			{
				Stage:     "test",
				Action:    "test_action",
				Timestamp: time.Now().UTC(),
				Meta:      nil,
			},
		},
	}

	jsonData, err := json.Marshal(productTrace)
	if err != nil {
		t.Fatalf("Failed to marshal ProductTrace with nil meta: %v", err)
	}

	var unmarshaledTrace ProductTrace
	err = json.Unmarshal(jsonData, &unmarshaledTrace)
	if err != nil {
		t.Fatalf("Failed to unmarshal ProductTrace with nil meta: %v", err)
	}

	if unmarshaledTrace.ProductID != productTrace.ProductID {
		t.Errorf("ProductID mismatch: expected %s, got %s",
			productTrace.ProductID, unmarshaledTrace.ProductID)
	}
}

func TestEmptyProductTrace(t *testing.T) {
	emptyTrace := ProductTrace{
		ProductID:   "empty_product",
		ProductMeta: nil,
		Traces:      []TraceEvent{},
	}

	jsonData, err := json.Marshal(emptyTrace)
	if err != nil {
		t.Fatalf("Failed to marshal empty ProductTrace: %v", err)
	}

	var unmarshaledTrace ProductTrace
	err = json.Unmarshal(jsonData, &unmarshaledTrace)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty ProductTrace: %v", err)
	}

	if unmarshaledTrace.ProductID != emptyTrace.ProductID {
		t.Errorf("ProductID mismatch: expected %s, got %s",
			emptyTrace.ProductID, unmarshaledTrace.ProductID)
	}

	if len(unmarshaledTrace.Traces) != 0 {
		t.Errorf("Expected empty productTraces, got %d productTraces", len(unmarshaledTrace.Traces))
	}
}

func TestBreadcrumbTracerKeyConstant(t *testing.T) {
	expectedKey := "__breadcrumb_tracer__"
	if breadcrumbTracerKey != expectedKey {
		t.Errorf("breadcrumbTracerKey constant mismatch: expected %s, got %s",
			expectedKey, breadcrumbTracerKey)
	}
}

func TestJSONTags(t *testing.T) {
	productTrace := ProductTrace{
		ProductID: "test",
		Traces:    []TraceEvent{},
	}

	jsonData, err := json.Marshal(productTrace)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonData)

	if !strings.Contains(jsonStr, "product_id") {
		t.Error("JSON should contain 'product_id' field")
	}

	if !strings.Contains(jsonStr, "traces") {
		t.Error("JSON should contain 'traces' field")
	}

	if strings.Contains(jsonStr, "product_meta") {
		t.Error("JSON should not contain 'product_meta' field when nil")
	}
}

func TestTraceEventJSONTags(t *testing.T) {
	traceEvent := TraceEvent{
		Stage:     "test",
		Action:    "test_action",
		Timestamp: time.Now().UTC(),
	}

	jsonData, err := json.Marshal(traceEvent)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonData)

	if !strings.Contains(jsonStr, "stage") {
		t.Error("JSON should contain 'stage' field")
	}

	if !strings.Contains(jsonStr, "action") {
		t.Error("JSON should contain 'action' field")
	}

	if !strings.Contains(jsonStr, "timestamp") {
		t.Error("JSON should contain 'timestamp' field")
	}

	if strings.Contains(jsonStr, "meta") {
		t.Error("JSON should not contain 'meta' field when nil")
	}
}

func TestBreadcrumbTracerGlobalTracesBasic(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces != nil {
		t.Error("GetGlobalTraces should return nil when no traces are recorded")
	}

	RecordGlobal(ctx, "search", "query_received", map[string]interface{}{
		"query":   "test search",
		"user_id": "12345",
	})

	globalTraces = GetGlobalTraces(ctx)
	if globalTraces == nil {
		t.Fatal("GetGlobalTraces should not return nil after recording a trace")
	}

	if len(globalTraces) != 1 {
		t.Fatalf("Expected 1 global trace, got %d", len(globalTraces))
	}

	trace := globalTraces[0]
	if trace.Stage != "search" {
		t.Errorf("Expected stage 'search', got '%s'", trace.Stage)
	}

	if trace.Action != "query_received" {
		t.Errorf("Expected action 'query_received', got '%s'", trace.Action)
	}

	if trace.Meta["query"] != "test search" {
		t.Errorf("Expected query 'test search', got '%v'", trace.Meta["query"])
	}

	if trace.Meta["user_id"] != "12345" {
		t.Errorf("Expected user_id '12345', got '%v'", trace.Meta["user_id"])
	}
}

func TestBreadcrumbTracerGlobalTracesMultiple(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	RecordGlobal(ctx, "search", "query_received", map[string]interface{}{
		"query": "first search",
	})

	RecordGlobal(ctx, "search", "filters_applied", map[string]interface{}{
		"filters": []string{"price", "category"},
	})

	RecordGlobal(ctx, "ranking", "algorithm_applied", map[string]interface{}{
		"algorithm": "ml_v2",
		"version":   "2.1",
	})

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces == nil {
		t.Fatal("GetGlobalTraces should not return nil")
	}

	if len(globalTraces) != 3 {
		t.Fatalf("Expected 3 global traces, got %d", len(globalTraces))
	}

	expectedActions := []string{"query_received", "filters_applied", "algorithm_applied"}
	for i, expectedAction := range expectedActions {
		if globalTraces[i].Action != expectedAction {
			t.Errorf("Expected action at index %d to be '%s', got '%s'",
				i, expectedAction, globalTraces[i].Action)
		}
	}

	for i := 1; i < len(globalTraces); i++ {
		if globalTraces[i].Timestamp.Before(globalTraces[i-1].Timestamp) {
			t.Error("Global traces should be in chronological order")
		}
	}
}

func TestBreadcrumbTracerGlobalTracesWithoutTracer(t *testing.T) {
	ctx := context.Background()

	RecordGlobal(ctx, "search", "query_received", map[string]interface{}{
		"query": "test search",
	})

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces != nil {
		t.Error("GetGlobalTraces should return nil for context without tracer")
	}
}

func TestBreadcrumbTracerGlobalTracesJSONSerialization(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	RecordGlobal(ctx, "search", "query_received", map[string]interface{}{
		"query":   "test search",
		"user_id": "12345",
		"filters": []string{"electronics", "books"},
	})

	RecordGlobal(ctx, "ranking", "algorithm_applied", map[string]interface{}{
		"algorithm": "ml_v2",
		"score":     0.95,
	})

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces == nil || len(globalTraces) != 2 {
		t.Fatal("Expected 2 global traces")
	}

	jsonData, err := json.Marshal(globalTraces)
	if err != nil {
		t.Fatalf("Failed to marshal global traces: %v", err)
	}

	var unmarshaledTraces []TraceEvent
	err = json.Unmarshal(jsonData, &unmarshaledTraces)
	if err != nil {
		t.Fatalf("Failed to unmarshal global traces: %v", err)
	}

	if len(unmarshaledTraces) != 2 {
		t.Fatalf("Expected 2 unmarshaled traces, got %d", len(unmarshaledTraces))
	}

	firstTrace := unmarshaledTraces[0]
	if firstTrace.Stage != "search" || firstTrace.Action != "query_received" {
		t.Error("First trace stage/action mismatch after JSON round-trip")
	}

	if firstTrace.Meta["query"] != "test search" {
		t.Error("First trace meta query mismatch after JSON round-trip")
	}

	secondTrace := unmarshaledTraces[1]
	if secondTrace.Stage != "ranking" || secondTrace.Action != "algorithm_applied" {
		t.Error("Second trace stage/action mismatch after JSON round-trip")
	}

	if secondTrace.Meta["algorithm"] != "ml_v2" {
		t.Error("Second trace meta algorithm mismatch after JSON round-trip")
	}
}

func TestBreadcrumbTracerGlobalTracesWithNilMeta(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	RecordGlobal(ctx, "search", "query_received", nil)

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces == nil || len(globalTraces) != 1 {
		t.Fatal("Expected 1 global trace")
	}

	trace := globalTraces[0]
	if trace.Stage != "search" || trace.Action != "query_received" {
		t.Error("Trace with nil meta should preserve stage and action")
	}

	if trace.Meta != nil {
		t.Error("Trace meta should be nil when recorded with nil meta")
	}

	jsonData, err := json.Marshal(globalTraces)
	if err != nil {
		t.Fatalf("Failed to marshal global traces with nil meta: %v", err)
	}

	var unmarshaledTraces []TraceEvent
	err = json.Unmarshal(jsonData, &unmarshaledTraces)
	if err != nil {
		t.Fatalf("Failed to unmarshal global traces with nil meta: %v", err)
	}

	if len(unmarshaledTraces) != 1 {
		t.Fatalf("Expected 1 unmarshaled trace, got %d", len(unmarshaledTraces))
	}
}

func TestBreadcrumbTracerGlobalTracesIndependentFromProductTraces(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	AddProduct(ctx, "product1", map[string]interface{}{"name": "Test Product"})
	RecordProduct(ctx, "product1", "ranking", "scored", map[string]interface{}{"score": 0.95})

	RecordGlobal(ctx, "search", "query_received", map[string]interface{}{"query": "test"})
	RecordGlobal(ctx, "search", "response_sent", map[string]interface{}{"duration_ms": 150})

	productTraces := GetProductTraces(ctx)
	globalTraces := GetGlobalTraces(ctx)

	if productTraces == nil || len(productTraces) != 1 {
		t.Error("Product traces should exist independently")
	}

	if _, exists := productTraces["product1"]; !exists {
		t.Error("Product1 trace should exist")
	}

	if globalTraces == nil || len(globalTraces) != 2 {
		t.Error("Global traces should exist independently")
	}

	if globalTraces[0].Action != "query_received" || globalTraces[1].Action != "response_sent" {
		t.Error("Global traces content should be independent from product traces")
	}
}

func TestBreadcrumbTracerGlobalTracesConcurrentAccess(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	const numGoroutines = 10
	const tracesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < tracesPerGoroutine; j++ {
				RecordGlobal(ctx, "concurrent", "test_action", map[string]interface{}{
					"routine_id": routineID,
					"trace_id":   j,
				})
			}
		}(i)
	}

	wg.Wait()

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces == nil {
		t.Fatal("Global traces should not be nil after concurrent writes")
	}

	expectedCount := numGoroutines * tracesPerGoroutine
	if len(globalTraces) != expectedCount {
		t.Errorf("Expected %d global traces, got %d", expectedCount, len(globalTraces))
	}

	for i, trace := range globalTraces {
		if trace.Stage != "concurrent" {
			t.Errorf("Trace %d: expected stage 'concurrent', got '%s'", i, trace.Stage)
		}
		if trace.Action != "test_action" {
			t.Errorf("Trace %d: expected action 'test_action', got '%s'", i, trace.Action)
		}
	}
}

func TestBreadcrumbTracerGlobalTracesDataIsolation(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	originalMeta := map[string]interface{}{
		"list":    []string{"item1", "item2"},
		"counter": 5,
	}

	RecordGlobal(ctx, "test", "data_isolation", originalMeta)

	originalMeta["counter"] = 10
	originalMeta["list"] = append(originalMeta["list"].([]string), "item3")

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces == nil || len(globalTraces) != 1 {
		t.Fatal("Expected 1 global trace")
	}

	trace := globalTraces[0]

	if trace.Meta["counter"] != 5 {
		t.Errorf("Expected counter to be 5, got %v", trace.Meta["counter"])
	}

	storedListInterface := trace.Meta["list"]
	var storedListLength int
	switch v := storedListInterface.(type) {
	case []string:
		storedListLength = len(v)
	case []interface{}:
		storedListLength = len(v)
	default:
		t.Fatalf("Unexpected type for stored list: %T", v)
	}

	if storedListLength != 2 {
		t.Errorf("Expected list length to be 2, got %d", storedListLength)
	}

	trace.Meta["counter"] = 20

	globalTraces2 := GetGlobalTraces(ctx)
	if globalTraces2[0].Meta["counter"] != 5 {
		t.Error("Returned trace meta should be isolated from modifications")
	}
}

func TestBreadcrumbParamsJSONSerialization(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	AddProduct(ctx, "product1", map[string]interface{}{"name": "Test"})
	RecordProduct(ctx, "product1", "search", "found", nil)
	RecordGlobal(ctx, "search", "started", map[string]interface{}{"query": "test"})

	params := GetBreadcrumbParams(ctx)
	if params == nil {
		t.Fatal("GetBreadcrumbParams should not return nil when tracing is enabled")
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal BreadcrumbParams: %v", err)
	}

	var unmarshaled BreadcrumbParams
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal BreadcrumbParams: %v", err)
	}

	if len(unmarshaled.ProductTraces) != 1 {
		t.Errorf("Expected 1 product trace, got %d", len(unmarshaled.ProductTraces))
	}
	if len(unmarshaled.GlobalTraces) != 1 {
		t.Errorf("Expected 1 global trace, got %d", len(unmarshaled.GlobalTraces))
	}
}

func TestGetBreadcrumbParamsWithoutTracer(t *testing.T) {
	ctx := context.Background()
	params := GetBreadcrumbParams(ctx)
	if params != nil {
		t.Error("GetBreadcrumbParams should return nil when tracing is not enabled")
	}
}
