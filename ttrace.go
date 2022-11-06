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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

// Trace start trace include span name, status code, tags
func Trace(ctx context.Context, spanName string, statusCode codes.Code, tags map[string]string) context.Context {
	ctx, span := ttracer.Start(ctx, spanName)

	span.SetStatus(statusCode, "")

	for key, value := range tags {
		span.SetAttributes(attribute.Key(key).String(value))
	}

	span.End()

	return ctx
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
