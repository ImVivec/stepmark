package stepmark_test

import (
	"context"
	"fmt"

	"github.com/ImVivec/stepmark"
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

func ExampleNewScope() {
	ctx := stepmark.New(context.Background())

	products := stepmark.NewScope(ctx, "product")
	products.Track("p_1", map[string]any{"name": "Widget"})
	products.Track("p_2", map[string]any{"name": "Gadget"})
	products.RecordEvent("p_1", "ranking", "scored", map[string]any{"score": 0.95})
	products.RecordEvent("p_2", "ranking", "scored", map[string]any{"score": 0.80})

	orders := stepmark.NewScope(ctx, "order")
	orders.Track("o_1", map[string]any{"total": 42.00})
	orders.RecordEvent("o_1", "validation", "passed", nil)

	trace := stepmark.Collect(ctx)
	fmt.Println("product kind:", trace.Entities["p_1"].Kind)
	fmt.Println("order kind:", trace.Entities["o_1"].Kind)
	fmt.Println("total entities:", len(trace.Entities))
	// Output:
	// product kind: product
	// order kind: order
	// total entities: 3
}

func ExampleWithTraceMeta() {
	ctx := stepmark.New(context.Background(), stepmark.WithTraceMeta(map[string]any{
		"request_id": "req_abc",
		"user_id":    "u_42",
	}))
	stepmark.Record(ctx, "search", "started", nil)

	trace := stepmark.Collect(ctx)
	fmt.Println("request_id:", trace.Meta["request_id"])
	fmt.Println("events:", len(trace.Events))
	// Output:
	// request_id: req_abc
	// events: 1
}

func ExampleWithEntityFilter() {
	ctx := stepmark.New(context.Background(), stepmark.WithEntityFilter(func(id string) bool {
		return id == "order_42"
	}))

	stepmark.RecordEntity(ctx, "order_42", "validation", "passed", nil)
	stepmark.RecordEntity(ctx, "order_99", "validation", "passed", nil)
	stepmark.Record(ctx, "request", "processed", nil)

	trace := stepmark.Collect(ctx)
	fmt.Println("entities:", len(trace.Entities))
	fmt.Println("events:", len(trace.Events))
	// Output:
	// entities: 1
	// events: 1
}

func ExampleStep() {
	ctx := stepmark.New(context.Background())
	exampleStepHelper(ctx)

	trace := stepmark.Collect(ctx)
	fmt.Println("stage:", trace.Events[0].Stage)
	fmt.Println("action:", trace.Events[0].Action)
	// Output:
	// stage: exampleStepHelper
	// action: processed
}

func exampleStepHelper(ctx context.Context) {
	stepmark.Step(ctx, "processed", nil)
}

func ExampleEnter() {
	ctx := stepmark.New(context.Background())
	exampleEnterHelper(ctx)

	trace := stepmark.Collect(ctx)
	fmt.Println("entered:", trace.Events[0].Stage, trace.Events[0].Action)
	fmt.Println("exited:", trace.Events[1].Stage, trace.Events[1].Action)
	// Output:
	// entered: exampleEnterHelper entered
	// exited: exampleEnterHelper exited
}

func exampleEnterHelper(ctx context.Context) {
	defer stepmark.Enter(ctx, nil)()
}
