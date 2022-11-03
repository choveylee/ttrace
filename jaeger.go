/**
 * @Author: lidonglin
 * @Description:
 * @File:  jaeger.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 15:46
 */

package ttrace

import (
	"log"

	"github.com/choveylee/tcfg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.12.0"
)

func init() {
	// init resource
	resource, err := newResource()
	if err != nil {
		log.Printf("init jaeger err (new resource %v).", err)

		return
	}

	jaegerEnable := tcfg.DefaultBool(tcfg.LocalKey(JaegerEnable), false)

	if jaegerEnable == false {
		return
	}

	// init jaeger exporter
	jaegerExporter, err := newJaegerExporter()
	if err != nil {
		log.Printf("init jaeger err (new jaeger exporter %v).", err)

		return
	}

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithResource(resource),
		tracesdk.WithBatcher(jaegerExporter),
	)

	// set tracer provider
	otel.SetTracerProvider(tracerProvider)

	//设置context传播载体
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))
}

// newResource returns a resource describing this application.
func newResource() (*resource.Resource, error) {
	appName, err := tcfg.String(AppName)
	if err != nil {
		return nil, err
	}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(appName)),
	)

	return r, nil
}

// newJaegerExporter returns a jaeger export otel data.
func newJaegerExporter() (tracesdk.SpanExporter, error) {
	jagerAddr, err := tcfg.String(JaegerEndpoint)
	if err != nil {
		return nil, err
	}

	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(jagerAddr),
		),
	)
	return exporter, err
}
