package stepmark_test

import (
	"context"
	"fmt"

	"github.com/vivekpatidar/stepmark"
)

func ExampleNew() {
	ctx := stepmark.New(context.Background())
	fmt.Println(stepmark.Enabled(ctx))
	// Output: true
}

func ExampleEnabled() {
	ctx := context.Background()
	fmt.Println("before:", stepmark.Enabled(ctx))

	ctx = stepmark.New(ctx)
	fmt.Println("after:", stepmark.Enabled(ctx))
	// Output:
	// before: false
	// after: true
}

func ExampleRecord() {
	ctx := stepmark.New(context.Background())
	stepmark.Record(ctx, "search", "query_received", map[string]any{
		"query": "wireless headphones",
	})

	trace := stepmark.Collect(ctx)
	fmt.Println(trace.Events[0].Stage)
	fmt.Println(trace.Events[0].Action)
	// Output:
	// search
	// query_received
}

func ExampleRecordEntity() {
	ctx := stepmark.New(context.Background())

	stepmark.Track(ctx, "order_42", map[string]any{"customer": "alice"})
	stepmark.RecordEntity(ctx, "order_42", "validation", "passed", nil)
	stepmark.RecordEntity(ctx, "order_42", "payment", "charged", map[string]any{
		"amount": 99.99,
	})

	trace := stepmark.Collect(ctx)
	order := trace.Entities["order_42"]
	fmt.Println(order.EntityID)
	fmt.Println(len(order.Events))
	fmt.Println(order.Events[0].Stage, "-", order.Events[0].Action)
	fmt.Println(order.Events[1].Stage, "-", order.Events[1].Action)
	// Output:
	// order_42
	// 2
	// validation - passed
	// payment - charged
}

func ExampleCollect() {
	ctx := stepmark.New(context.Background())
	stepmark.Track(ctx, "user_7", map[string]any{"role": "admin"})
	stepmark.RecordEntity(ctx, "user_7", "auth", "login", nil)
	stepmark.Record(ctx, "request", "started", nil)

	trace := stepmark.Collect(ctx)
	fmt.Println("entities:", len(trace.Entities))
	fmt.Println("events:", len(trace.Events))
	// Output:
	// entities: 1
	// events: 1
}
