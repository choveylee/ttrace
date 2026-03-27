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

// InjectContext generates new trace and span IDs via [NewTraceId] and [NewSpanId], then calls [InjectTrace].
// It returns the input ctx unchanged if ID generation fails.
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

// InjectTrace decodes hex trace id (32 chars) and span id (16 chars) into the context.
// When ctx has no valid SpanContext, it behaves like a local root (Remote=false, sampled).
// For ids and sampling copied from an inbound W3C traceparent header, use [InjectRemoteTrace]
// or, preferably, [ExtractHTTP] so tracestate and flags stay aligned with the wire format.
func InjectTrace(ctx context.Context, strTraceId, strSpanId string) (context.Context, error) {
	srcTraceId, err := hex.DecodeString(strTraceId)
	if err != nil {
		return ctx, err
	}

	if len(srcTraceId) != 16 {
		return ctx, fmt.Errorf("trace id illegal")
	}

	srcSpanId, err := hex.DecodeString(strSpanId)
	if err != nil {
		return ctx, err
	}

	if len(srcSpanId) != 8 {
		return ctx, fmt.Errorf("span id illegal")
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

// InjectRemoteTrace sets trace id and parent span id from hex strings, as when reconstructing
// W3C trace context without full header parsing. sampled should match the trace-flags sampled bit.
// The SpanContext is marked Remote=true (upstream / cross-service). If you have raw HTTP headers,
// use [ExtractHTTP] instead to also recover tracestate and avoid manual flag handling.
func InjectRemoteTrace(ctx context.Context, strTraceId, strParentSpanId string, sampled bool) (context.Context, error) {
	srcTraceId, err := hex.DecodeString(strTraceId)
	if err != nil {
		return ctx, err
	}

	if len(srcTraceId) != 16 {
		return ctx, fmt.Errorf("trace id illegal")
	}

	srcSpanId, err := hex.DecodeString(strParentSpanId)
	if err != nil {
		return ctx, err
	}

	if len(srcSpanId) != 8 {
		return ctx, fmt.Errorf("span id illegal")
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

// NewTraceId returns a new W3C trace id as a 32-character lowercase hex string (16 bytes).
func NewTraceId() (string, error) {
	var bytes [16]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("ttrace: crypto/rand trace id: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}

// NewSpanId returns a new span id as a 16-character lowercase hex string (8 bytes).
func NewSpanId() (string, error) {
	var bytes [8]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", fmt.Errorf("ttrace: crypto/rand span id: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}
