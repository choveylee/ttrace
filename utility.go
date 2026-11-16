package ttrace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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

// InjectTrace decodes hexadecimal trace and span identifiers and stores them in ctx. traceIdText and
// spanIdText must be valid, non-zero trace.TraceID and trace.SpanID values encoded as 32 and 16
// hexadecimal characters, respectively. Without a valid parent, the configured sampler decides the
// local trace's sampled flag once here (Remote=false); later spans inherit the decision. Existing
// valid parents retain their flags and tracestate. For inbound W3C headers, prefer [ExtractHTTP];
// for remote parents with explicit sampling, use [InjectRemoteTrace].
func InjectTrace(ctx context.Context, traceIdText, spanIdText string) (context.Context, error) {
	traceId, err := decodeTraceID(traceIdText)
	if err != nil {
		return ctx, err
	}

	spanId, err := decodeSpanID(spanIdText)
	if err != nil {
		return ctx, err
	}

	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		spanContext = spanContext.WithTraceID(traceId).WithSpanID(spanId)
	} else {
		spanContext = newLocalSpanContext(ctx, traceId, spanId)
	}

	return trace.ContextWithSpanContext(ctx, spanContext), nil
}

// InjectRemoteTrace constructs a remote [trace.SpanContext] from hexadecimal trace and parent span
// identifiers. traceIdText and parentSpanIdText must be valid, non-zero IDs encoded as 32 and 16
// hexadecimal characters, respectively. The sampled argument should match the sampled bit in the W3C
// trace flags on the wire. When complete HTTP headers are available, prefer [ExtractHTTP] so
// tracestate and flags are preserved exactly.
func InjectRemoteTrace(ctx context.Context, traceIdText, parentSpanIdText string, sampled bool) (context.Context, error) {
	traceId, err := decodeTraceID(traceIdText)
	if err != nil {
		return ctx, err
	}

	spanId, err := decodeSpanID(parentSpanIdText)
	if err != nil {
		return ctx, err
	}

	flags := trace.TraceFlags(0)
	if sampled {
		flags = trace.FlagsSampled
	}

	spanContextConfig := trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,

		TraceFlags: flags,

		Remote: true,
	}

	spanContext := trace.NewSpanContext(spanContextConfig)

	return trace.ContextWithSpanContext(ctx, spanContext), nil
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

// decodeTraceID decodes an exact-length hexadecimal identifier and rejects all-zero IDs.
func decodeTraceID(value string) (trace.TraceID, error) {
	rawTraceID, err := decodeHexIdentifier("trace ID", value, 16)
	if err != nil {
		return trace.TraceID{}, err
	}

	var traceID trace.TraceID
	copy(traceID[:], rawTraceID)

	if !traceID.IsValid() {
		return trace.TraceID{}, errors.New("ttrace: trace ID must not be all zeros")
	}

	return traceID, nil
}

// decodeSpanID decodes an exact-length hexadecimal identifier and rejects all-zero IDs.
func decodeSpanID(value string) (trace.SpanID, error) {
	rawSpanID, err := decodeHexIdentifier("span ID", value, 8)
	if err != nil {
		return trace.SpanID{}, err
	}

	var spanID trace.SpanID
	copy(spanID[:], rawSpanID)

	if !spanID.IsValid() {
		return trace.SpanID{}, errors.New("ttrace: span ID must not be all zeros")
	}

	return spanID, nil
}

// decodeHexIdentifier checks the encoded length before allocating or decoding untrusted IDs.
func decodeHexIdentifier(name, value string, expectedBytes int) ([]byte, error) {
	if len(value) != expectedBytes*2 {
		return nil, fmt.Errorf("ttrace: %s must contain %d hex characters; got %d", name, expectedBytes*2, len(value))
	}

	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("ttrace: decode %s: %w", name, err)
	}

	return decoded, nil
}
