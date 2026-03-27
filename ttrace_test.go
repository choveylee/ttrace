package ttrace

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestContextWithBaggage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	out, err := ContextWithBaggage(ctx, "key=value")
	if err != nil {
		t.Fatal(err)
	}
	b := GetBaggage(out)
	if b.Member("key").Key() == "" {
		t.Fatal("expected baggage key")
	}

	_, err = ContextWithBaggage(ctx, "%%%invalid")
	if err == nil {
		t.Fatal("want parse error")
	}
}

func TestExtractHTTP_NilHeader(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	out := ExtractHTTP(ctx, nil)
	if out != ctx {
		t.Fatal("nil header should return same ctx")
	}
}

func TestExtractHTTP_Traceparent(t *testing.T) {
	t.Parallel()

	// 00-{trace-id}-{parent-id}-01 (sampled)
	h := http.Header{}
	h.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	ctx := ExtractHTTP(context.Background(), h)
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		t.Fatal("expected valid SpanContext from traceparent")
	}
	wantT, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	wantS, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	if sc.TraceID() != wantT || sc.SpanID() != wantS {
		t.Fatalf("ids mismatch: %+v", sc)
	}
	if !sc.IsSampled() {
		t.Fatal("flag 01 should be sampled")
	}
}

func TestValidTraceId(t *testing.T) {
	t.Parallel()

	if ValidTraceId(trace.TraceID{}) {
		t.Fatal("zero TraceID should be invalid")
	}
	tid, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	if !ValidTraceId(tid) {
		t.Fatal("expected valid")
	}
}

func TestSetTraceId_InvalidNoOp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	out := SetTraceId(ctx, trace.TraceID{})
	if out != ctx {
		t.Fatal("invalid trace id should be no-op")
	}
}

func TestSetTraceId_NewSpanContext(t *testing.T) {
	t.Parallel()

	tid, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	out := SetTraceId(ctx, tid)
	if out == ctx {
		t.Fatal("expected new context")
	}
	sc := trace.SpanContextFromContext(out)
	if sc.TraceID() != tid {
		t.Fatalf("trace id: got %v", sc.TraceID())
	}
	if !sc.IsValid() {
		t.Fatal("expected valid SpanContext")
	}
	if !sc.IsSampled() {
		t.Fatal("expected sampled")
	}
}

func TestGetTracer(t *testing.T) {
	t.Parallel()

	if GetTracer() == nil {
		t.Fatal("GetTracer should not be nil")
	}
}

func TestStart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx2, sp := Start(ctx, "test-span")
	defer sp.End()
	if sp == nil {
		t.Fatal("Start returned nil span")
	}
	_ = ctx2
	// Exercise GetSpan path (noop provider may not attach a valid SpanContext).
	_ = GetSpan(ctx2)
}
