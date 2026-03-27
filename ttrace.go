/**
 * @Author: lidonglin
 * @Description:
 * @File:  ttrace.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 14:01
 */

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

// TracerName is the instrumentation scope name passed to [go.opentelemetry.io/otel.Tracer].
const (
	TracerName = "github.com/choveylee/ttrace"
)

// ContextWithBaggage parses items as a W3C baggage string and returns ctx carrying that baggage.
// On parse error it returns ctx and the error from [go.opentelemetry.io/otel/baggage.Parse].
func ContextWithBaggage(ctx context.Context, items string) (context.Context, error) {
	bag, err := baggage.Parse(items)
	if err != nil {
		return ctx, err
	}

	return baggage.ContextWithBaggage(ctx, bag), nil
}

// Start starts a new span with spanName using the global TracerProvider and [TracerName].
// opts are passed through to the underlying [trace.Tracer.Start].
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(TracerName).Start(ctx, spanName, opts...)
}

// GetTracer returns the Tracer from the current global TracerProvider.
func GetTracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// GetSpan returns the [trace.Span] from ctx, which may be non-recording when no span was attached.
func GetSpan(ctx context.Context) trace.Span {
	span := trace.SpanFromContext(ctx)
	return span
}

// GetSpanContext returns trace.SpanFromContext(ctx).SpanContext().
func GetSpanContext(ctx context.Context) trace.SpanContext {
	return trace.SpanFromContext(ctx).SpanContext()
}

// GetBaggage returns baggage previously associated with ctx via [ContextWithBaggage] or propagation.
func GetBaggage(ctx context.Context) baggage.Baggage {
	return baggage.FromContext(ctx)
}

// SetTraceId sets trace id on the context SpanContext. If the current SpanContext is invalid,
// a new SpanContext is built with this trace id, a new random span id, and the sampled flag set.
// Invalid traceId is a no-op (returns ctx unchanged). If crypto/rand fails when a new span id
// is required, returns ctx unchanged.
func SetTraceId(ctx context.Context, traceId trace.TraceID) context.Context {
	spanContext, ok := spanContextWithTraceID(trace.SpanFromContext(ctx).SpanContext(), traceId)
	if !ok {
		return ctx
	}

	return trace.ContextWithSpanContext(ctx, spanContext)
}

// spanContextWithTraceAndSpan builds a [trace.SpanContext] with the given trace and span IDs.
// If parent is valid, IDs replace those fields while keeping trace flags, trace state, and remote flag.
// Otherwise it returns a new local-root context with [trace.FlagsSampled] and not remote.
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

// spanContextWithTraceID updates traceId on an existing valid SpanContext, or creates a new
// sampled SpanContext with a random span id when parent is invalid. It returns (zero, false)
// if traceId is invalid or [crypto/rand] fails when generating a span id.
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

// GetTraceId returns the trace ID from the span context stored in ctx (may be invalid if no span).
func GetTraceId(ctx context.Context) trace.TraceID {
	span := trace.SpanFromContext(ctx)

	return span.SpanContext().TraceID()
}

// ValidTraceId reports whether traceId is non-zero per OpenTelemetry [trace.TraceID.IsValid].
func ValidTraceId(traceId trace.TraceID) bool {
	return traceId.IsValid()
}

// Inject serializes the trace and baggage context from ctx into supplier using the global propagator.
func Inject(ctx context.Context, supplier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, supplier)
}

// Extract merges context propagation data from supplier into ctx using the global propagator.
func Extract(ctx context.Context, supplier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, supplier)
}

// ExtractHTTP applies the global TextMapPropagator to HTTP headers (e.g. traceparent, tracestate,
// and baggage when enabled). Prefer this for incoming requests over parsing trace/span ids by hand:
// it preserves sampling flags and trace state as sent by the client.
func ExtractHTTP(ctx context.Context, header http.Header) context.Context {
	if header == nil {
		return ctx
	}

	return Extract(ctx, propagation.HeaderCarrier(header))
}
