package ttrace

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestNewTraceId_NewSpanId(t *testing.T) {
	t.Parallel()

	tid, err := NewTraceId()
	if err != nil {
		t.Fatalf("NewTraceId: %v", err)
	}
	if len(tid) != 32 {
		t.Fatalf("trace id len: got %d want 32", len(tid))
	}
	if _, err := hex.DecodeString(tid); err != nil {
		t.Fatalf("NewTraceId not hex: %v", err)
	}

	sid, err := NewSpanId()
	if err != nil {
		t.Fatalf("NewSpanId: %v", err)
	}
	if len(sid) != 16 {
		t.Fatalf("span id len: got %d want 16", len(sid))
	}
	if _, err := hex.DecodeString(sid); err != nil {
		t.Fatalf("NewSpanId not hex: %v", err)
	}
}

func TestInjectTrace(t *testing.T) {
	t.Parallel()

	const (
		tHex = "4bf92f3577b34da6a3ce929d0e0e4736"
		sHex = "00f067aa0ba902b7"
	)

	ctx := context.Background()
	out, err := InjectTrace(ctx, tHex, sHex)
	if err != nil {
		t.Fatalf("InjectTrace: %v", err)
	}

	tid, err := trace.TraceIDFromHex(tHex)
	if err != nil {
		t.Fatal(err)
	}
	sid, err := trace.SpanIDFromHex(sHex)
	if err != nil {
		t.Fatal(err)
	}

	sc := trace.SpanContextFromContext(out)
	if sc.TraceID() != tid {
		t.Fatalf("trace id: got %v want %v", sc.TraceID(), tid)
	}
	if sc.SpanID() != sid {
		t.Fatalf("span id: got %v want %v", sc.SpanID(), sid)
	}
	if !sc.IsValid() {
		t.Fatal("expected valid SpanContext")
	}
	if sc.IsRemote() {
		t.Fatal("InjectTrace without parent should not be remote")
	}
	if !sc.IsSampled() {
		t.Fatal("expected sampled local root")
	}

	_, err = InjectTrace(out, "nothex", sHex)
	if err == nil {
		t.Fatal("want decode error")
	}

	_, err = InjectTrace(out, strings.Repeat("a", 30), sHex)
	if err == nil || !strings.Contains(err.Error(), "trace id illegal") {
		t.Fatalf("want trace id illegal, got %v", err)
	}

	_, err = InjectTrace(out, tHex, strings.Repeat("b", 14))
	if err == nil || !strings.Contains(err.Error(), "span id illegal") {
		t.Fatalf("want span id illegal, got %v", err)
	}
}

func TestInjectRemoteTrace(t *testing.T) {
	t.Parallel()

	const (
		tHex = "4bf92f3577b34da6a3ce929d0e0e4736"
		sHex = "00f067aa0ba902b7"
	)

	ctx := context.Background()

	out, err := InjectRemoteTrace(ctx, tHex, sHex, true)
	if err != nil {
		t.Fatalf("InjectRemoteTrace: %v", err)
	}
	sc := trace.SpanContextFromContext(out)
	if !sc.IsRemote() {
		t.Fatal("expected remote SpanContext")
	}
	if !sc.IsSampled() {
		t.Fatal("expected sampled")
	}

	out2, err := InjectRemoteTrace(ctx, tHex, sHex, false)
	if err != nil {
		t.Fatal(err)
	}
	sc2 := trace.SpanContextFromContext(out2)
	if sc2.IsSampled() {
		t.Fatal("expected not sampled")
	}
}

func TestInjectContext(t *testing.T) {
	t.Parallel()

	ctx, err := InjectContext(context.Background())
	if err != nil {
		t.Fatalf("InjectContext: %v", err)
	}
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		t.Fatal("expected valid SpanContext")
	}
}
