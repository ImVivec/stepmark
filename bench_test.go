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

func BenchmarkStep_Disabled(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		Step(ctx, "action", nil)
	}
}

func BenchmarkStep_Enabled(b *testing.B) {
	ctx := New(context.Background())
	b.ReportAllocs()
	for b.Loop() {
		Step(ctx, "action", nil)
	}
}

func BenchmarkStepEntity_Disabled(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		StepEntity(ctx, "entity_1", "action", nil)
	}
}

func BenchmarkEnter_Disabled(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		Enter(ctx, nil)()
	}
}

func BenchmarkEnterEntity_Disabled(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		EnterEntity(ctx, "entity_1", nil)()
	}
}

func BenchmarkRecordEntity_Filtered(b *testing.B) {
	ctx := New(context.Background(), WithEntityFilter(func(id string) bool {
		return id == "wanted"
	}))
	b.ReportAllocs()
	for b.Loop() {
		RecordEntity(ctx, "blocked", "stage", "action", nil)
	}
}

func BenchmarkStepEntity_Filtered(b *testing.B) {
	ctx := New(context.Background(), WithEntityFilter(func(id string) bool {
		return id == "wanted"
	}))
	b.ReportAllocs()
	for b.Loop() {
		StepEntity(ctx, "blocked", "action", nil)
	}
}
