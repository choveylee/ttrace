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
	"os"
	"path/filepath"
	"strings"

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

	jaegerEndpoint := tcfg.DefaultString(tcfg.LocalKey(JaegerEndpoint), "")
	if jaegerEndpoint == "" {
		return
	}

	startJaeger(jaegerEndpoint)
}

func InitJaeger(jaegerEndpoint string) error {
	err := startJaeger(jaegerEndpoint)
	if err != nil {
		return err
	}

	return nil
}

func startJaeger(jaegerEndpoint string) error {
	// init resource
	resource, err := newResource()
	if err != nil {
		log.Printf("init jaeger (%s) err (new resource %v).", jaegerEndpoint, err)

		return nil
	}

	// init jaeger exporter
	jaegerExporter, err := newJaegerExporter(jaegerEndpoint)
	if err != nil {
		log.Printf("init jaeger (%s) err (new jaeger exporter %v).", jaegerEndpoint, err)

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
	appName := tcfg.DefaultString(AppName, "")
	if appName == "" {
		_, fileName := filepath.Split(os.Args[0])
		fileExt := filepath.Ext(os.Args[0])

		appName = strings.TrimSuffix(fileName, fileExt)
	}

	r, err := resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(appName)))
	if err != nil {
		return nil, err
	}

	return r, nil
}

// newJaegerExporter returns a jaeger export otel data.
func newJaegerExporter(jaegerEndpoint string) (tracesdk.SpanExporter, error) {
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(jaegerEndpoint),
		),
	)
	return exporter, err
}
