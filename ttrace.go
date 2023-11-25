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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var ttracer trace.Tracer

const (
	TracerName = "github.com/choveylee/ttrace"
)

func init() {
	//初始化默认tracer
	ttracer = otel.Tracer(TracerName)
}

func ContextWithBaggage(ctx context.Context, items string) context.Context {
	bag, _ := baggage.Parse(items)
	ctx = baggage.ContextWithBaggage(ctx, bag)

	return ctx
}

// Start start trace include span name, opts
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	ctx, span := ttracer.Start(ctx, spanName)

	return ctx, span
}

// GetTracer get global trace
func GetTracer() trace.Tracer {
	return ttracer
}

// GetSpan get span from context
func GetSpan(ctx context.Context) trace.Span {
	span := trace.SpanFromContext(ctx)
	return span
}

// GetSpanContext get span context from context
func GetSpanContext(ctx context.Context) trace.SpanContext {
	return trace.SpanFromContext(ctx).SpanContext()
}

// GetBaggage get baggage from context
func GetBaggage(ctx context.Context) baggage.Baggage {
	return baggage.FromContext(ctx)
}

// SetTraceID set trace id to context
func SetTraceID(ctx context.Context, traceId trace.TraceID) context.Context {
	span := trace.SpanFromContext(ctx)
	newSpanContext := span.SpanContext().WithTraceID(traceId)

	return trace.ContextWithSpanContext(ctx, newSpanContext)
}

// GetTraceID get trace id from context从context获取TraceId
func GetTraceID(ctx context.Context) trace.TraceID {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().TraceID()
}

// ValidTraceID valid trace id
func ValidTraceID(traceId trace.TraceID) bool {
	return traceId.IsValid()
}

// Inject inject text map carrier to context
func Inject(ctx context.Context, supplier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, supplier)
}

// Extract extract text map carrier from context
func Extract(ctx context.Context, supplier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, supplier)
}
