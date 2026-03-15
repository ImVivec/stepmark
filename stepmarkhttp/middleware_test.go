package stepmarkhttp_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ImVivec/stepmark"
	"github.com/ImVivec/stepmark/stepmarkhttp"
)

func TestMiddleware_HeaderTrigger_Enabled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !stepmark.Enabled(r.Context()) {
			t.Error("tracer should be enabled")
		}
		stepmark.Record(r.Context(), "handler", "reached", nil)
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
	)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Stepmark", "true")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_HeaderTrigger_Disabled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stepmark.Enabled(r.Context()) {
			t.Error("tracer should NOT be enabled without trigger header")
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
	)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_QueryTrigger(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !stepmark.Enabled(r.Context()) {
			t.Error("tracer should be enabled with query param")
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.QueryTrigger("debug"),
	)(handler)

	req := httptest.NewRequest("GET", "/test?debug=1", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_QueryTrigger_Absent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stepmark.Enabled(r.Context()) {
			t.Error("tracer should NOT be enabled without query param")
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.QueryTrigger("debug"),
	)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)
}

func TestMiddleware_ResponseHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stepmark.Record(r.Context(), "test", "handled", map[string]any{"k": "v"})
		stepmark.RecordEntity(r.Context(), "item_1", "search", "found", nil)
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
		stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
	)(handler)

	req := httptest.NewRequest("GET", "/api", nil)
	req.Header.Set("X-Stepmark", "true")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	raw := rec.Header().Get("X-Stepmark-Trace")
	if raw == "" {
		t.Fatal("expected X-Stepmark-Trace response header")
	}

	var trace stepmark.Trace
	if err := json.Unmarshal([]byte(raw), &trace); err != nil {
		t.Fatalf("failed to unmarshal trace header: %v", err)
	}
	if len(trace.Events) != 1 {
		t.Fatalf("expected 1 unscoped event, got %d", len(trace.Events))
	}
	if len(trace.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(trace.Entities))
	}
	if trace.Events[0].Stage != "test" {
		t.Errorf("expected stage 'test', got '%s'", trace.Events[0].Stage)
	}
}

func TestMiddleware_ResponseHeader_WithBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stepmark.Record(r.Context(), "api", "processed", nil)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
		stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
	)(handler)

	req := httptest.NewRequest("GET", "/api", nil)
	req.Header.Set("X-Stepmark", "true")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	if string(body) != `{"ok":true}` {
		t.Fatalf("body mismatch: %s", body)
	}

	raw := rec.Header().Get("X-Stepmark-Trace")
	if raw == "" {
		t.Fatal("trace header should be set even when body is written")
	}

	var trace stepmark.Trace
	if err := json.Unmarshal([]byte(raw), &trace); err != nil {
		t.Fatalf("invalid trace JSON: %v", err)
	}
	if len(trace.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(trace.Events))
	}
}

func TestMiddleware_ResponseHeader_NoBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stepmark.Record(r.Context(), "api", "empty", nil)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
		stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
	)(handler)

	req := httptest.NewRequest("GET", "/api", nil)
	req.Header.Set("X-Stepmark", "true")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	raw := rec.Header().Get("X-Stepmark-Trace")
	if raw == "" {
		t.Fatal("trace header should be set even for empty responses")
	}
}

func TestMiddleware_ResponseHeader_NotSet_WhenDisabled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
		stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
	)(handler)

	req := httptest.NewRequest("GET", "/api", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Header().Get("X-Stepmark-Trace") != "" {
		t.Error("trace header should NOT be set when trigger is inactive")
	}
}

func TestMiddleware_OnFinish(t *testing.T) {
	var collected *stepmark.Trace

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stepmark.Record(r.Context(), "handler", "done", nil)
		stepmark.RecordEntity(r.Context(), "e1", "s", "a", nil)
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
		stepmarkhttp.WithOnFinish(func(_ context.Context, trace *stepmark.Trace) {
			collected = trace
		}),
	)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Stepmark", "true")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if collected == nil {
		t.Fatal("OnFinish callback should have been called")
	}
	if len(collected.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(collected.Events))
	}
	if len(collected.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(collected.Entities))
	}
}

func TestMiddleware_OnFinish_NotCalled_WhenDisabled(t *testing.T) {
	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
		stepmarkhttp.WithOnFinish(func(_ context.Context, _ *stepmark.Trace) {
			called = true
		}),
	)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if called {
		t.Error("OnFinish should NOT be called when trigger is inactive")
	}
}

func TestMiddleware_TracerOptions(t *testing.T) {
	var collected *stepmark.Trace

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 20; i++ {
			stepmark.Record(r.Context(), "s", "a", nil)
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
		stepmarkhttp.WithTracerOptions(stepmark.WithMaxEvents(5)),
		stepmarkhttp.WithOnFinish(func(_ context.Context, trace *stepmark.Trace) {
			collected = trace
		}),
	)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Stepmark", "true")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if collected == nil {
		t.Fatal("OnFinish should have been called")
	}
	if len(collected.Events) != 5 {
		t.Fatalf("expected 5 events (capped), got %d", len(collected.Events))
	}
}

func TestMiddleware_HeaderAndOnFinish_Combined(t *testing.T) {
	var finished *stepmark.Trace

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stepmark.Record(r.Context(), "api", "handled", nil)
		w.WriteHeader(http.StatusOK)
	})

	mw := stepmarkhttp.Middleware(
		stepmarkhttp.HeaderTrigger("X-Stepmark"),
		stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
		stepmarkhttp.WithOnFinish(func(_ context.Context, trace *stepmark.Trace) {
			finished = trace
		}),
	)(handler)

	req := httptest.NewRequest("GET", "/api", nil)
	req.Header.Set("X-Stepmark", "true")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Header().Get("X-Stepmark-Trace") == "" {
		t.Error("response header should be set")
	}
	if finished == nil {
		t.Error("OnFinish should also be called")
	}
}

func TestMiddleware_PreservesStatusCode(t *testing.T) {
	for _, code := range []int{200, 201, 204, 400, 404, 500} {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			stepmark.Record(r.Context(), "s", "a", nil)
			w.WriteHeader(code)
		})

		mw := stepmarkhttp.Middleware(
			stepmarkhttp.HeaderTrigger("X-Stepmark"),
			stepmarkhttp.WithResponseHeader("X-Stepmark-Trace"),
		)(handler)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Stepmark", "true")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != code {
			t.Errorf("status %d: expected %d, got %d", code, code, rec.Code)
		}
	}
}
