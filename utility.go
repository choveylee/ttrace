/**
 * @Author: lidonglin
 * @Description:
 * @File:  utility.go
 * @Version: 1.0.0
 * @Date: 2023/11/14 09:34
 */

package ttrace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

// InjectContext generates trace and span IDs with [NewTraceId] and [NewSpanId], then applies [InjectTrace].
// On ID generation failure it returns ctx and the error from those helpers.
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

// InjectTrace decodes hex strings into trace and span IDs and stores them in ctx.
// strTraceId and strSpanId must be valid hex encoding 16 and 8 bytes respectively (32 and 16 hex digits).
// With no valid parent span, the result behaves as a sampled local root (Remote=false).
// For inbound W3C headers, prefer [ExtractHTTP]; for remote parents and flags, see [InjectRemoteTrace].
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

// InjectRemoteTrace builds a remote [trace.SpanContext] from hex-encoded trace and span IDs.
// sampled should match the trace-flags sampled bit on the wire. Remote is set to true (upstream span).
// When full HTTP headers are available, prefer [ExtractHTTP] to restore tracestate and flags.
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

// NewTraceId returns a new trace ID as a 32-character lowercase hex string (16 random bytes).
func NewTraceId() (string, error) {
	var bytes [16]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("ttrace: crypto/rand trace id: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}

// NewSpanId returns a new span ID as a 16-character lowercase hex string (8 random bytes).
func NewSpanId() (string, error) {
	var bytes [8]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("ttrace: crypto/rand span id: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}
