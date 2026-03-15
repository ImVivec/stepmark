package breadcrumb

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewBreadcrumbTracer(t *testing.T) {
	tracer := newBreadcrumbTracer()

	if tracer == nil {
		t.Fatal("newBreadcrumbTracer() returned nil")
	}

	if tracer.productTraces == nil {
		t.Error("tracer.productTraces is nil")
	}

	if len(tracer.productTraces) != 0 {
		t.Error("tracer.productTraces should be empty initially")
	}
}

func TestWithBreadcrumbTracer(t *testing.T) {
	ctx := context.Background()

	if IsBreadcrumbTracingEnabled(ctx) {
		t.Error("tracing should not be enabled in empty context")
	}

	ctxWithTracer := WithBreadcrumbTracer(ctx)
	if !IsBreadcrumbTracingEnabled(ctxWithTracer) {
		t.Error("tracing should be enabled after WithBreadcrumbTracer")
	}

	if ctx == ctxWithTracer {
		t.Error("WithBreadcrumbTracer should return a different context")
	}
}

func TestGetBreadcrumbTracer(t *testing.T) {
	ctx := context.Background()

	tracer, ok := getBreadcrumbTracer(ctx)
	if ok {
		t.Error("getBreadcrumbTracer should return false for context without tracer")
	}
	if tracer != nil {
		t.Error("tracer should be nil when not present")
	}

	ctxWithTracer := WithBreadcrumbTracer(ctx)
	tracer, ok = getBreadcrumbTracer(ctxWithTracer)
	if !ok {
		t.Error("getBreadcrumbTracer should return true for context with tracer")
	}
	if tracer == nil {
		t.Error("tracer should not be nil when present")
	}
}

func TestIsBreadcrumbTracingEnabled(t *testing.T) {
	ctx := context.Background()

	if IsBreadcrumbTracingEnabled(ctx) {
		t.Error("tracing should be disabled in empty context")
	}

	ctxWithTracer := WithBreadcrumbTracer(ctx)
	if !IsBreadcrumbTracingEnabled(ctxWithTracer) {
		t.Error("tracing should be enabled after adding tracer")
	}
}

func TestAddProduct(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())
	productID := "product_123"
	meta := map[string]interface{}{
		"name":  "Test Product",
		"price": 99.99,
	}

	AddProduct(ctx, productID, meta)
	traces := GetProductTraces(ctx)

	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	productTrace, exists := traces[productID]
	if !exists {
		t.Fatalf("expected trace for product %s to exist", productID)
	}

	if productTrace.ProductID != productID {
		t.Errorf("expected product ID %s, got %s", productID, productTrace.ProductID)
	}

	if !reflect.DeepEqual(productTrace.ProductMeta, meta) {
		t.Errorf("expected meta %v, got %v", meta, productTrace.ProductMeta)
	}

	if len(productTrace.Traces) != 0 {
		t.Error("productTraces should be empty for new product")
	}
}

func TestAddProductReinitialization(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())
	productID := "product_123"

	meta1 := map[string]interface{}{"version": 1}
	AddProduct(ctx, productID, meta1)

	meta2 := map[string]interface{}{"version": 2}
	AddProduct(ctx, productID, meta2)

	traces := GetProductTraces(ctx)
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	productTrace, exists := traces[productID]
	if !exists {
		t.Fatalf("expected trace for product %s to exist", productID)
	}

	productMeta := productTrace.ProductMeta
	if productMeta["version"] != 2 {
		t.Errorf("expected version 2, got %v", productMeta["version"])
	}

	if productMeta["re-initialized"] != true {
		t.Error("expected re-initialized flag to be true")
	}
}

func TestAddProductWithNilMeta(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())
	productID := "product_123"

	AddProduct(ctx, productID, nil)
	traces := GetProductTraces(ctx)

	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	productTrace, exists := traces[productID]
	if !exists {
		t.Fatalf("expected trace for product %s to exist", productID)
	}

	if productTrace.ProductMeta != nil {
		t.Error("expected nil product meta")
	}
}

func TestAddProductWithoutTracer(t *testing.T) {
	ctx := context.Background()
	productID := "product_123"
	meta := map[string]interface{}{"test": "value"}

	AddProduct(ctx, productID, meta)

	traces := GetProductTraces(ctx)
	if traces != nil {
		t.Error("expected nil productTraces for context without tracer")
	}
}

func TestRecordProduct(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())
	productID := "product_123"
	stage := "search"
	action := "filter_applied"
	meta := map[string]interface{}{"filter": "category"}

	before := time.Now().UTC()
	RecordProduct(ctx, productID, stage, action, meta)
	after := time.Now().UTC()

	traces := GetProductTraces(ctx)
	if len(traces) != 1 {
		t.Fatalf("expected 1 product trace, got %d", len(traces))
	}

	productTrace, exists := traces[productID]
	if !exists {
		t.Fatalf("expected trace for product %s to exist", productID)
	}

	if productTrace.ProductID != productID {
		t.Errorf("expected product ID %s, got %s", productID, productTrace.ProductID)
	}

	if len(productTrace.Traces) != 1 {
		t.Fatalf("expected 1 trace event, got %d", len(productTrace.Traces))
	}

	traceEvent := productTrace.Traces[0]
	if traceEvent.Stage != stage {
		t.Errorf("expected stage %s, got %s", stage, traceEvent.Stage)
	}

	if traceEvent.Action != action {
		t.Errorf("expected action %s, got %s", action, traceEvent.Action)
	}

	if !reflect.DeepEqual(traceEvent.Meta, meta) {
		t.Errorf("expected meta %v, got %v", meta, traceEvent.Meta)
	}

	if traceEvent.Timestamp.Before(before) || traceEvent.Timestamp.After(after) {
		t.Error("timestamp should be between before and after times")
	}
}

func TestRecordMultipleEvents(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())
	productID := "product_123"

	RecordProduct(ctx, productID, "search", "query_entered", map[string]interface{}{"query": "milk"})
	RecordProduct(ctx, productID, "search", "filter_applied", map[string]interface{}{"filter": "brand"})
	RecordProduct(ctx, productID, "cart", "item_added", nil)

	traces := GetProductTraces(ctx)
	if len(traces) != 1 {
		t.Fatalf("expected 1 product trace, got %d", len(traces))
	}

	productTrace, exists := traces[productID]
	if !exists {
		t.Fatalf("expected trace for product %s to exist", productID)
	}

	if len(productTrace.Traces) != 3 {
		t.Fatalf("expected 3 trace events, got %d", len(productTrace.Traces))
	}

	expectedStages := []string{"search", "search", "cart"}
	expectedActions := []string{"query_entered", "filter_applied", "item_added"}

	for i, traceEvent := range productTrace.Traces {
		if traceEvent.Stage != expectedStages[i] {
			t.Errorf("event %d: expected stage %s, got %s", i, expectedStages[i], traceEvent.Stage)
		}
		if traceEvent.Action != expectedActions[i] {
			t.Errorf("event %d: expected action %s, got %s", i, expectedActions[i], traceEvent.Action)
		}
	}
}

func TestRecordWithoutTracer(t *testing.T) {
	ctx := context.Background()
	productID := "product_123"

	RecordProduct(ctx, productID, "test", "test", nil)

	traces := GetProductTraces(ctx)
	if traces != nil {
		t.Error("expected nil productTraces for context without tracer")
	}
}

func TestGetProductTraces(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	AddProduct(ctx, "product_1", map[string]interface{}{"name": "Product 1"})
	RecordProduct(ctx, "product_1", "search", "viewed", nil)

	AddProduct(ctx, "product_2", map[string]interface{}{"name": "Product 2"})
	RecordProduct(ctx, "product_2", "cart", "added", nil)

	traces := GetProductTraces(ctx)
	if len(traces) != 2 {
		t.Fatalf("expected 2 productTraces, got %d", len(traces))
	}

	product1Trace, exists := traces["product_1"]
	if !exists {
		t.Error("expected trace for product_1 to exist")
	}
	if product1Trace.ProductID != "product_1" {
		t.Errorf("expected product_1, got %s", product1Trace.ProductID)
	}
	if product1Trace.ProductMeta["name"] != "Product 1" {
		t.Errorf("expected Product 1, got %v", product1Trace.ProductMeta["name"])
	}
	if len(product1Trace.Traces) != 1 {
		t.Errorf("expected 1 trace for product_1, got %d", len(product1Trace.Traces))
	}

	product2Trace, exists := traces["product_2"]
	if !exists {
		t.Error("expected trace for product_2 to exist")
	}
	if product2Trace.ProductID != "product_2" {
		t.Errorf("expected product_2, got %s", product2Trace.ProductID)
	}
	if product2Trace.ProductMeta["name"] != "Product 2" {
		t.Errorf("expected Product 2, got %v", product2Trace.ProductMeta["name"])
	}
	if len(product2Trace.Traces) != 1 {
		t.Errorf("expected 1 trace for product_2, got %d", len(product2Trace.Traces))
	}
}

func TestGetTracesReturnsDeepCopies(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())
	productID := "product_123"
	meta := map[string]interface{}{"original": "value"}

	AddProduct(ctx, productID, meta)
	RecordProduct(ctx, productID, "test", "action", map[string]interface{}{"trace": "data"})

	traces1 := GetProductTraces(ctx)
	traces2 := GetProductTraces(ctx)

	product1Trace1, exists1 := traces1[productID]
	if !exists1 {
		t.Fatalf("expected trace for product %s to exist in traces1", productID)
	}
	product1Trace2, exists2 := traces2[productID]
	if !exists2 {
		t.Fatalf("expected trace for product %s to exist in traces2", productID)
	}

	product1Trace1.ProductMeta["modified"] = "new_value"

	if product1Trace2.ProductMeta["modified"] != nil {
		t.Error("modifications to traces1 should not affect traces2 (ProductMeta)")
	}

	product1Trace1.Traces[0].Meta["trace_modified"] = "new_trace_value"

	if product1Trace2.Traces[0].Meta["trace_modified"] != nil {
		t.Error("modifications to traces1 should not affect traces2 (TraceEvent.Meta)")
	}

	if product1Trace2.ProductMeta["original"] != "value" {
		t.Error("original ProductMeta value should be preserved in traces2")
	}
	if product1Trace2.Traces[0].Meta["trace"] != "data" {
		t.Error("original TraceEvent.Meta value should be preserved in traces2")
	}

	product1Trace1.Traces = append(product1Trace1.Traces, TraceEvent{
		Stage:     "new",
		Action:    "new_action",
		Timestamp: time.Now(),
	})

	if len(product1Trace1.Traces) == len(product1Trace2.Traces) {
		t.Error("modifying traces1 slice affected traces2 slice - slices should be independent")
	}
}

func TestGetTracesWithoutTracer(t *testing.T) {
	ctx := context.Background()

	traces := GetProductTraces(ctx)
	if traces != nil {
		t.Error("expected nil productTraces for context without tracer")
	}
}

func TestCopyMap(t *testing.T) {
	nilCopy := copyMap(nil)
	if nilCopy != nil {
		t.Error("copyMap(nil) should return nil")
	}

	emptyMap := make(map[string]interface{})
	emptyCopy := copyMap(emptyMap)
	if emptyCopy == nil {
		t.Error("copyMap should not return nil for empty map")
	}
	if len(emptyCopy) != 0 {
		t.Error("copied empty map should be empty")
	}

	original := map[string]interface{}{
		"string": "value",
		"number": 42,
		"float":  3.14,
		"bool":   true,
	}

	copied := copyMap(original)

	if !reflect.DeepEqual(original, copied) {
		t.Error("copied map should be equal to original")
	}

	if &original == &copied {
		t.Error("copied map should not be the same reference as original")
	}

	original["new_key"] = "new_value"
	if copied["new_key"] != nil {
		t.Error("modifying original should not affect copy")
	}

	copied["copy_key"] = "copy_value"
	if original["copy_key"] != nil {
		t.Error("modifying copy should not affect original")
	}
}

func TestConcurrentAccess(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())
	numGoroutines := 100
	numOperationsPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				productID := fmt.Sprintf("product_%d_%d", routineID, j)
				meta := map[string]interface{}{
					"routine":   routineID,
					"operation": j,
				}

				AddProduct(ctx, productID, meta)
				RecordProduct(ctx, productID, "test", "concurrent_action", meta)

				traces := GetProductTraces(ctx)
				if traces == nil {
					t.Errorf("GetTraces returned nil during concurrent access")
				}
			}
		}(i)
	}

	wg.Wait()

	traces := GetProductTraces(ctx)
	expectedCount := numGoroutines * numOperationsPerGoroutine

	if len(traces) != expectedCount {
		t.Errorf("expected %d productTraces, got %d", expectedCount, len(traces))
	}

	for productID, trace := range traces {
		if len(trace.Traces) != 1 {
			t.Errorf("product %s should have exactly 1 trace event, got %d",
				productID, len(trace.Traces))
		}
		if trace.ProductMeta == nil {
			t.Errorf("product %s should have metadata", productID)
		}
	}
}

func TestRecordGlobal(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	RecordGlobal(ctx, "search", "query_received", map[string]interface{}{
		"query":   "test search",
		"user_id": "12345",
	})

	RecordGlobal(ctx, "search", "filters_applied", map[string]interface{}{
		"filters": []string{"price", "category"},
	})

	RecordGlobal(ctx, "search", "results_prepared", map[string]interface{}{
		"count": 50,
	})

	globalTraces := GetGlobalTraces(ctx)

	if globalTraces == nil {
		t.Fatal("GetGlobalTraces() returned nil")
	}

	if len(globalTraces) != 3 {
		t.Fatalf("Expected 3 global productTraces, got %d", len(globalTraces))
	}

	if globalTraces[0].Stage != "search" || globalTraces[0].Action != "query_received" {
		t.Error("First trace does not match expected stage/action")
	}

	if globalTraces[1].Stage != "search" || globalTraces[1].Action != "filters_applied" {
		t.Error("Second trace does not match expected stage/action")
	}

	if globalTraces[2].Stage != "search" || globalTraces[2].Action != "results_prepared" {
		t.Error("Third trace does not match expected stage/action")
	}

	if globalTraces[0].Meta["query"] != "test search" {
		t.Error("First trace metadata does not match")
	}

	if globalTraces[0].Meta["user_id"] != "12345" {
		t.Error("First trace user_id metadata does not match")
	}
}

func TestRecordGlobalWithoutTracer(t *testing.T) {
	ctx := context.Background()

	RecordGlobal(ctx, "search", "query_received", map[string]interface{}{
		"query": "test search",
	})

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces != nil {
		t.Error("GetGlobalTraces() should return nil for context without tracer")
	}
}

func TestGetGlobalTracesEmptyTraces(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	globalTraces := GetGlobalTraces(ctx)
	if globalTraces != nil {
		t.Error("GetGlobalTraces() should return nil when no productTraces are recorded")
	}
}

func TestGlobalTracesWithProductTraces(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	AddProduct(ctx, "product1", map[string]interface{}{"name": "Test Product"})
	RecordProduct(ctx, "product1", "ranking", "scored", map[string]interface{}{"score": 0.95})

	RecordGlobal(ctx, "search", "query_received", map[string]interface{}{"query": "test"})
	RecordGlobal(ctx, "search", "response_sent", map[string]interface{}{"duration_ms": 150})

	productTraces := GetProductTraces(ctx)
	globalTraces := GetGlobalTraces(ctx)

	if productTraces == nil || len(productTraces) != 1 {
		t.Error("Product productTraces should still work")
	}

	if _, exists := productTraces["product1"]; !exists {
		t.Error("Product1 trace should exist")
	}

	if globalTraces == nil || len(globalTraces) != 2 {
		t.Error("Global productTraces should work alongside product productTraces")
	}

	if globalTraces[0].Action != "query_received" || globalTraces[1].Action != "response_sent" {
		t.Error("Global productTraces content incorrect")
	}
}

func TestGlobalTracesTimestamp(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	before := time.Now().UTC()
	RecordGlobal(ctx, "test", "timestamp_test", map[string]interface{}{})
	after := time.Now().UTC()

	globalTraces := GetGlobalTraces(ctx)
	if len(globalTraces) != 1 {
		t.Fatal("Expected 1 global trace")
	}

	timestamp := globalTraces[0].Timestamp
	if timestamp.Before(before) || timestamp.After(after) {
		t.Error("Timestamp should be between before and after times")
	}
}

func TestGlobalTracesMetaCopy(t *testing.T) {
	ctx := WithBreadcrumbTracer(context.Background())

	originalMeta := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	RecordGlobal(ctx, "test", "meta_test", originalMeta)

	originalMeta["key1"] = "modified"
	originalMeta["key3"] = "new"

	globalTraces := GetGlobalTraces(ctx)
	if len(globalTraces) != 1 {
		t.Fatal("Expected 1 global trace")
	}

	if globalTraces[0].Meta["key1"] != "value1" {
		t.Error("Meta should be copied, not referenced")
	}

	if _, exists := globalTraces[0].Meta["key3"]; exists {
		t.Error("New key should not exist in copied meta")
	}

	globalTraces[0].Meta["copy_key"] = "copy_value"

	globalTraces2 := GetGlobalTraces(ctx)
	if _, exists := globalTraces2[0].Meta["copy_key"]; exists {
		t.Error("Modifying returned meta should not affect stored meta")
	}
}

func BenchmarkAddProduct(b *testing.B) {
	ctx := WithBreadcrumbTracer(context.Background())
	meta := map[string]interface{}{
		"name":  "Test Product",
		"price": 99.99,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AddProduct(ctx, fmt.Sprintf("product_%d", i), meta)
	}
}

func BenchmarkRecordProduct(b *testing.B) {
	ctx := WithBreadcrumbTracer(context.Background())
	meta := map[string]interface{}{"filter": "category"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RecordProduct(ctx, fmt.Sprintf("product_%d", i%1000), "search", "action", meta)
	}
}

func BenchmarkGetProductTraces(b *testing.B) {
	ctx := WithBreadcrumbTracer(context.Background())

	for i := 0; i < 100; i++ {
		productID := fmt.Sprintf("product_%d", i)
		AddProduct(ctx, productID, map[string]interface{}{"id": i})
		RecordProduct(ctx, productID, "test", "action", nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetProductTraces(ctx)
	}
}
