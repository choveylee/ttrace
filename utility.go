package ttrace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

// InjectContext generates new trace and span identifiers with [NewTraceId] and [NewSpanId], then
// applies [InjectTrace]. On failure it returns ctx and the error from ID generation.
func InjectContext(ctx context.Context) (context.Context, error) {
	traceId, err := NewTraceId()
	if err != nil {
		return ctx, err
	}

	spanId, err := NewSpanId()
	if err != nil {
		return ctx, err
	}

	return InjectTrace(ctx, traceId, spanId)
}

// InjectTrace decodes hexadecimal trace and span identifiers and stores them in ctx. strTraceId and
// strSpanId must decode to 16 and 8 bytes respectively (32 and 16 hexadecimal digits). When no valid
// parent span exists, the resulting context behaves as a sampled local root (Remote=false). For
// inbound W3C headers, prefer [ExtractHTTP]; for remote parents with explicit sampling, use
// [InjectRemoteTrace].
func InjectTrace(ctx context.Context, strTraceId, strSpanId string) (context.Context, error) {
	srcTraceId, err := hex.DecodeString(strTraceId)
	if err != nil {
		return ctx, err
	}

	if len(srcTraceId) != 16 {
		return ctx, fmt.Errorf("ttrace: trace ID length invalid: decoded %d byte(s), expected 16 (32 hex digits)", len(srcTraceId))
	}

	srcSpanId, err := hex.DecodeString(strSpanId)
	if err != nil {
		return ctx, err
	}

	if len(srcSpanId) != 8 {
		return ctx, fmt.Errorf("ttrace: span ID length invalid: decoded %d byte(s), expected 8 (16 hex digits)", len(srcSpanId))
	}

	var desTraceId [16]byte
	var desSpanId [8]byte

	copy(desTraceId[:], srcTraceId)
	copy(desSpanId[:], srcSpanId)

	traceId := trace.TraceID(desTraceId)
	spanId := trace.SpanID(desSpanId)

	preSpanContext := trace.SpanFromContext(ctx).SpanContext()

	spanContext := spanContextWithTraceAndSpan(preSpanContext, traceId, spanId)

	ctx = trace.ContextWithSpanContext(ctx, spanContext)

	return ctx, nil
}

// InjectRemoteTrace constructs a remote [trace.SpanContext] from hexadecimal trace and parent span
// identifiers. The sampled flag should match the sampled bit in W3C trace flags on the wire; Remote
// is set to true. When complete HTTP headers are available, prefer [ExtractHTTP] to preserve
// tracestate and flags.
func InjectRemoteTrace(ctx context.Context, strTraceId, strParentSpanId string, sampled bool) (context.Context, error) {
	srcTraceId, err := hex.DecodeString(strTraceId)
	if err != nil {
		return ctx, err
	}

	if len(srcTraceId) != 16 {
		return ctx, fmt.Errorf("ttrace: trace ID length invalid: decoded %d byte(s), expected 16 (32 hex digits)", len(srcTraceId))
	}

	srcSpanId, err := hex.DecodeString(strParentSpanId)
	if err != nil {
		return ctx, err
	}

	if len(srcSpanId) != 8 {
		return ctx, fmt.Errorf("ttrace: span ID length invalid: decoded %d byte(s), expected 8 (16 hex digits)", len(srcSpanId))
	}

	var desTraceId [16]byte
	var desSpanId [8]byte

	copy(desTraceId[:], srcTraceId)
	copy(desSpanId[:], srcSpanId)

	traceId := trace.TraceID(desTraceId)
	spanId := trace.SpanID(desSpanId)

	flags := trace.TraceFlags(0)
	if sampled {
		flags = trace.FlagsSampled
	}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,

		TraceFlags: flags,

		Remote: true,
	})

	return trace.ContextWithSpanContext(ctx, spanContext), nil
}

// NewTraceId returns a new trace identifier as a 32-character lowercase hexadecimal string (16
// cryptographically random bytes).
func NewTraceId() (string, error) {
	var bytes [16]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("ttrace: crypto/rand trace id: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}

// NewSpanId returns a new span identifier as a 16-character lowercase hexadecimal string (8
// cryptographically random bytes).
func NewSpanId() (string, error) {
	var bytes [8]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("ttrace: crypto/rand span id: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}
