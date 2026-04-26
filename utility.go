package ttrace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

// InjectContext generates a new trace ID and span ID with [NewTraceId] and [NewSpanId], then
// applies [InjectTrace]. On failure, it returns ctx together with the underlying ID-generation
// error.
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
// strSpanId must be valid, non-zero trace.TraceID and trace.SpanID values encoded as 32 and 16
// hexadecimal characters, respectively. When no valid parent span exists, the resulting context is
// treated as a sampled local root (Remote=false). For inbound W3C headers, prefer [ExtractHTTP]; for
// remote parents with explicit sampling, use [InjectRemoteTrace].
func InjectTrace(ctx context.Context, strTraceId, strSpanId string) (context.Context, error) {
	traceId, err := decodeTraceID(strTraceId)
	if err != nil {
		return ctx, err
	}

	spanId, err := decodeSpanID(strSpanId)
	if err != nil {
		return ctx, err
	}

	preSpanContext := trace.SpanFromContext(ctx).SpanContext()

	spanContext := spanContextWithTraceAndSpan(preSpanContext, traceId, spanId)

	ctx = trace.ContextWithSpanContext(ctx, spanContext)

	return ctx, nil
}

// InjectRemoteTrace constructs a remote [trace.SpanContext] from hexadecimal trace and parent span
// identifiers. strTraceId and strParentSpanId must be valid, non-zero IDs encoded as 32 and 16
// hexadecimal characters, respectively. The sampled argument should match the sampled bit in the W3C
// trace flags on the wire. When complete HTTP headers are available, prefer [ExtractHTTP] so
// tracestate and flags are preserved exactly.
func InjectRemoteTrace(ctx context.Context, strTraceId, strParentSpanId string, sampled bool) (context.Context, error) {
	traceId, err := decodeTraceID(strTraceId)
	if err != nil {
		return ctx, err
	}

	spanId, err := decodeSpanID(strParentSpanId)
	if err != nil {
		return ctx, err
	}

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

func decodeTraceID(value string) (trace.TraceID, error) {
	rawTraceID, err := decodeHexIdentifier("trace ID", value, 16)
	if err != nil {
		return trace.TraceID{}, err
	}

	var traceID trace.TraceID
	copy(traceID[:], rawTraceID)

	if !traceID.IsValid() {
		return trace.TraceID{}, fmt.Errorf("ttrace: invalid trace ID: value must not be all zeros")
	}

	return traceID, nil
}

func decodeSpanID(value string) (trace.SpanID, error) {
	rawSpanID, err := decodeHexIdentifier("span ID", value, 8)
	if err != nil {
		return trace.SpanID{}, err
	}

	var spanID trace.SpanID
	copy(spanID[:], rawSpanID)

	if !spanID.IsValid() {
		return trace.SpanID{}, fmt.Errorf("ttrace: invalid span ID: value must not be all zeros")
	}

	return spanID, nil
}

func decodeHexIdentifier(name, value string, expectedBytes int) ([]byte, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("ttrace: decode %s: %w", name, err)
	}

	if len(decoded) != expectedBytes {
		return nil, fmt.Errorf("ttrace: invalid %s length: got %d bytes, want %d (%d hex characters)", name, len(decoded), expectedBytes, expectedBytes*2)
	}

	return decoded, nil
}

// NewTraceId returns a new trace ID as a 32-character lowercase hexadecimal string backed by 16
// cryptographically random bytes.
func NewTraceId() (string, error) {
	var bytes [16]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("ttrace: generate trace ID: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}

// NewSpanId returns a new span ID as a 16-character lowercase hexadecimal string backed by 8
// cryptographically random bytes.
func NewSpanId() (string, error) {
	var bytes [8]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("ttrace: generate span ID: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}
