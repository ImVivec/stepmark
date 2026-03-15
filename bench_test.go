package stepmark

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkEnabled_Disabled(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		Enabled(ctx)
	}
}

func BenchmarkEnabled_Enabled(b *testing.B) {
	ctx := New(context.Background())
	b.ReportAllocs()
	for b.Loop() {
		Enabled(ctx)
	}
}

func BenchmarkRecord_Disabled(b *testing.B) {
	ctx := context.Background()
	meta := map[string]any{"key": "value"}
	b.ReportAllocs()
	for b.Loop() {
		Record(ctx, "stage", "action", meta)
	}
}

func BenchmarkRecord_Enabled(b *testing.B) {
	ctx := New(context.Background())
	meta := map[string]any{"key": "value"}
	b.ReportAllocs()
	for b.Loop() {
		Record(ctx, "stage", "action", meta)
	}
}

func BenchmarkRecordEntity_Disabled(b *testing.B) {
	ctx := context.Background()
	meta := map[string]any{"key": "value"}
	b.ReportAllocs()
	for b.Loop() {
		RecordEntity(ctx, "entity_1", "stage", "action", meta)
	}
}

func BenchmarkRecordEntity_Enabled(b *testing.B) {
	ctx := New(context.Background())
	meta := map[string]any{"key": "value"}
	b.ReportAllocs()
	for b.Loop() {
		RecordEntity(ctx, fmt.Sprintf("entity_%d", b.N%100), "stage", "action", meta)
	}
}

func BenchmarkTrack_Disabled(b *testing.B) {
	ctx := context.Background()
	meta := map[string]any{"key": "value"}
	b.ReportAllocs()
	for b.Loop() {
		Track(ctx, "entity_1", meta)
	}
}

func BenchmarkCollect(b *testing.B) {
	ctx := New(context.Background())
	for i := 0; i < 50; i++ {
		id := fmt.Sprintf("entity_%d", i)
		Track(ctx, id, map[string]any{"i": i})
		RecordEntity(ctx, id, "stage", "action", nil)
	}
	for i := 0; i < 10; i++ {
		Record(ctx, "stage", "action", map[string]any{"i": i})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Collect(ctx)
	}
}

func BenchmarkRecord_Disabled_NilMeta(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		Record(ctx, "stage", "action", nil)
	}
}

func BenchmarkRecordEntity_Disabled_NilMeta(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		RecordEntity(ctx, "entity_1", "stage", "action", nil)
	}
}
