/**
 * @Author: lidonglin
 * @Description:
 * @File:  ttrace.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 14:01
 */

// Package ttrace provides OpenTelemetry helpers: W3C baggage, trace context propagation (including
// HTTP headers), manual trace and span ID injection, and span accessors. TracerProvider bootstrap
// and global propagator setup are in tracer.go. Optional Gin middleware is in module
// [github.com/choveylee/ttrace/gin].
package ttrace

import (
	"context"
	"crypto/rand"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracerName is the instrumentation scope name used with [go.opentelemetry.io/otel.Tracer].
const (
	TracerName = "github.com/choveylee/ttrace"
)

// ContextWithBaggage parses items as a W3C Baggage header value and returns ctx with that baggage attached.
// On failure it returns ctx unchanged and the error from [go.opentelemetry.io/otel/baggage.Parse].
func ContextWithBaggage(ctx context.Context, items string) (context.Context, error) {
	bag, err := baggage.Parse(items)
	if err != nil {
		return ctx, err
	}

	return baggage.ContextWithBaggage(ctx, bag), nil
}

// Start begins a span named spanName using the global TracerProvider and [TracerName].
// opts are forwarded to [trace.Tracer.Start].
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(TracerName).Start(ctx, spanName, opts...)
}

// GetTracer returns a [trace.Tracer] for [TracerName] from the global TracerProvider.
func GetTracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// GetSpan returns the current [trace.Span] from ctx. It may be non-recording when no span is present.
func GetSpan(ctx context.Context) trace.Span {
	span := trace.SpanFromContext(ctx)
	return span
}

// GetSpanContext returns the [trace.SpanContext] for the current span in ctx.
func GetSpanContext(ctx context.Context) trace.SpanContext {
	return trace.SpanFromContext(ctx).SpanContext()
}

// GetBaggage returns W3C Baggage from ctx (set via [ContextWithBaggage] or the baggage propagator).
func GetBaggage(ctx context.Context) baggage.Baggage {
	return baggage.FromContext(ctx)
}

// SetTraceId updates the trace ID on the span context in ctx. If the current [trace.SpanContext]
// is valid, only the trace ID is replaced. If it is invalid, a new sampled, non-remote root
// context is created with traceId and a new random span ID. Invalid traceId leaves ctx unchanged;
// if a new span ID is needed and [crypto/rand] fails, ctx is also left unchanged.
func SetTraceId(ctx context.Context, traceId trace.TraceID) context.Context {
	spanContext, ok := spanContextWithTraceID(trace.SpanFromContext(ctx).SpanContext(), traceId)
	if !ok {
		return ctx
	}

	return trace.ContextWithSpanContext(ctx, spanContext)
}

// spanContextWithTraceAndSpan returns a [trace.SpanContext] with the given trace and span IDs.
// If spanContext is valid, trace and span IDs are replaced and other fields are preserved.
// Otherwise it returns a new local root with [trace.FlagsSampled] and Remote=false.
func spanContextWithTraceAndSpan(spanContext trace.SpanContext, traceId trace.TraceID, spanId trace.SpanID) trace.SpanContext {
	if spanContext.IsValid() {
		return spanContext.WithTraceID(traceId).WithSpanID(spanId)
	}

	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,

		TraceFlags: trace.FlagsSampled,
	})
}

// spanContextWithTraceID sets traceId on a valid parent context, or builds a new sampled root
// with a random span ID when the parent is invalid. It returns (zero, false) if traceId is invalid
// or if [crypto/rand] fails while generating a span ID.
func spanContextWithTraceID(spanContext trace.SpanContext, traceId trace.TraceID) (trace.SpanContext, bool) {
	if !traceId.IsValid() {
		return trace.SpanContext{}, false
	}

	if spanContext.IsValid() {
		return spanContext.WithTraceID(traceId), true
	}

	var spanId trace.SpanID

	_, err := rand.Read(spanId[:])
	if err != nil {
		return trace.SpanContext{}, false
	}

	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
		SpanID:  spanId,

		TraceFlags: trace.FlagsSampled,
	}), true
}

// GetTraceId returns the trace ID from the span in ctx. It is invalid when no span is set.
func GetTraceId(ctx context.Context) trace.TraceID {
	span := trace.SpanFromContext(ctx)

	return span.SpanContext().TraceID()
}

// ValidTraceId reports whether traceId is valid per [trace.TraceID.IsValid].
func ValidTraceId(traceId trace.TraceID) bool {
	return traceId.IsValid()
}

// Inject writes trace and baggage state from ctx into supplier using the global TextMapPropagator.
func Inject(ctx context.Context, supplier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, supplier)
}

// Extract returns a context derived from ctx with trace and baggage context read from supplier.
func Extract(ctx context.Context, supplier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, supplier)
}

// ExtractHTTP runs the global TextMapPropagator against header (e.g. traceparent, tracestate, baggage).
// Use it for inbound HTTP requests instead of manual hex IDs so sampling flags and tracestate match the wire format.
func ExtractHTTP(ctx context.Context, header http.Header) context.Context {
	if header == nil {
		return ctx
	}

	return Extract(ctx, propagation.HeaderCarrier(header))
}
