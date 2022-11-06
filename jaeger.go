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
	jaegerEnable := tcfg.DefaultBool(tcfg.LocalKey(JaegerEnable), false)

	if jaegerEnable == false {
		return
	}

	startJaeger()
}

func InitJaeger() error {
	err := startJaeger()
	if err != nil {
		return err
	}

	return nil
}

func startJaeger() error {
	// init resource
	resource, err := newResource()
	if err != nil {
		log.Printf("init jaeger err (new resource %v).", err)

		return nil
	}

	// init jaeger exporter
	jaegerExporter, err := newJaegerExporter()
	if err != nil {
		log.Printf("init jaeger err (new jaeger exporter %v).", err)

		return err
	}

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithResource(resource),
		tracesdk.WithBatcher(jaegerExporter),
	)

	// set tracer provider
	otel.SetTracerProvider(tracerProvider)

	// set context text map propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))

	return nil
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
