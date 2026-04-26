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

// TracerName is the OpenTelemetry instrumentation scope name used by [Start] and [GetTracer].
const (
	TracerName = "github.com/choveylee/ttrace"
)

// ContextWithBaggage parses items as a W3C Baggage header value and returns a context that carries
// the parsed baggage. On parse failure, it returns ctx unchanged together with the error from
// [go.opentelemetry.io/otel/baggage.Parse].
func ContextWithBaggage(ctx context.Context, items string) (context.Context, error) {
	bag, err := baggage.Parse(items)
	if err != nil {
		return ctx, err
	}

	return baggage.ContextWithBaggage(ctx, bag), nil
}

// Start starts a span with the given name using the global TracerProvider and instrumentation scope
// [TracerName]. The opts arguments are passed through to [trace.Tracer.Start].
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(TracerName).Start(ctx, spanName, opts...)
}

// GetTracer returns the [trace.Tracer] for instrumentation scope [TracerName] from the global
// TracerProvider.
func GetTracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// GetSpan returns the current [trace.Span] from ctx. The returned span may be non-recording when no
// span is active in the context.
func GetSpan(ctx context.Context) trace.Span {
	span := trace.SpanFromContext(ctx)
	return span
}

// GetSpanContext returns the [trace.SpanContext] of the current span stored in ctx.
func GetSpanContext(ctx context.Context) trace.SpanContext {
	return trace.SpanFromContext(ctx).SpanContext()
}

// GetBaggage returns the W3C baggage stored in ctx, whether attached with [ContextWithBaggage] or
// restored by the global TextMapPropagator during [Extract].
func GetBaggage(ctx context.Context) baggage.Baggage {
	return baggage.FromContext(ctx)
}

// SetTraceId replaces the trace ID on the span context in ctx. If the existing [trace.SpanContext]
// is valid, only the trace ID changes. Otherwise, a new sampled local-root context is created with
// traceId and a randomly generated span ID. If traceId is invalid, or span ID generation fails, ctx
// is returned unchanged.
func SetTraceId(ctx context.Context, traceId trace.TraceID) context.Context {
	spanContext, ok := spanContextWithTraceID(trace.SpanFromContext(ctx).SpanContext(), traceId)
	if !ok {
		return ctx
	}

	return trace.ContextWithSpanContext(ctx, spanContext)
}

// spanContextWithTraceAndSpan constructs a [trace.SpanContext] with the given trace and span
// identifiers. When spanContext is valid, both IDs are replaced and remaining fields are
// preserved. Otherwise it returns a new local root with [trace.FlagsSampled] and Remote set to false.
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

// spanContextWithTraceID applies traceId to a valid parent [trace.SpanContext], or constructs a
// new sampled local root with a cryptographically random span ID when the parent is invalid. It
// returns the zero [trace.SpanContext] and false if traceId is invalid or random generation fails.
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

// GetTraceId returns the trace ID of the current span in ctx. The result is invalid when no span is
// present in the context.
func GetTraceId(ctx context.Context) trace.TraceID {
	span := trace.SpanFromContext(ctx)

	return span.SpanContext().TraceID()
}

// ValidTraceId reports whether traceId is considered valid by [trace.TraceID.IsValid].
func ValidTraceId(traceId trace.TraceID) bool {
	return traceId.IsValid()
}

// Inject writes trace context and baggage from ctx into supplier by using the global
// TextMapPropagator.
func Inject(ctx context.Context, supplier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, supplier)
}

// Extract returns a child of ctx whose trace and baggage state is deserialized from supplier by
// using the global TextMapPropagator.
func Extract(ctx context.Context, supplier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, supplier)
}

// ExtractHTTP applies the global TextMapPropagator to header values such as traceparent,
// tracestate, and baggage. Prefer this helper for inbound HTTP requests over manual hex identifiers
// so sampling flags and tracestate remain consistent with the wire representation.
func ExtractHTTP(ctx context.Context, header http.Header) context.Context {
	if header == nil {
		return ctx
	}

	return Extract(ctx, propagation.HeaderCarrier(header))
}
