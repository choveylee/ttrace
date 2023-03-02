/**
 * @Author: lidonglin
 * @Description:
 * @File:  jaeger.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 15:46
 */

package ttrace

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/choveylee/tcfg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.17.0"
)

var (
	tracerProvider *trace.TracerProvider
)

func init() {
	tracerMode := tcfg.DefaultInt(tcfg.LocalKey(TracerMode), TracerModeDisable)
	if tracerMode != TracerModeStdout && tracerMode != TracerModeJaeger {
		return
	}

	err := startTracer(tracerMode)
	if err != nil {
		log.Printf("init err (start tracer %v).", err)
	}
}

func GetTracerProvider() *trace.TracerProvider {
	return tracerProvider
}

func Shutdown() error {
	err := tracerProvider.Shutdown(context.Background())
	if err != nil {
		log.Printf("shut down err (%v).", err)

		return err
	}

	return nil
}

func startTracer(tracerMode int) error {
	// init resource
	resource, err := newResource()
	if err != nil {
		log.Printf("start tracer err (new resource %v).", err)

		return nil
	}

	var tracerExporter trace.SpanExporter

	if tracerMode == TracerModeJaeger {
		// init jaeger exporter
		jaegerEndpoint := tcfg.DefaultString(tcfg.LocalKey(JaegerEndpoint), "")
		if jaegerEndpoint == "" {
			log.Printf("start tracer err (jaeger endpoint illegal).")

			return fmt.Errorf("jaeger endpoint illegal")
		}

		tracerExporter, err = newJaegerExporter(jaegerEndpoint)
		if err != nil {
			log.Printf("start tracer (%s) err (new jaeger exporter %v).", jaegerEndpoint, err)

			return err
		}
	} else {
		tracerExporter, err = newStdoutExporter()
		if err != nil {
			log.Printf("start tracer err (new stdout exporter %v).", err)

			return err
		}
	}

	sampler := trace.AlwaysSample()

	// 采样率
	samplingFraction := tcfg.DefaultFloat64(tcfg.LocalKey(TracerSamplingFraction), 0.1)
	// 单个实例每秒最大采样请求数量
	maxTracesPerSecond := tcfg.DefaultFloat64(tcfg.LocalKey(TracerMaxTracesPerSec), 1.0)

	if samplingFraction != -1 && maxTracesPerSecond != -1 {
		sampler = GuaranteedThroughputProbabilitySampler(samplingFraction, maxTracesPerSecond)
	}

	tracerProvider = trace.NewTracerProvider(
		trace.WithBatcher(tracerExporter),
		trace.WithSampler(sampler),
		trace.WithResource(resource),
	)

	// set tracer provider
	otel.SetTracerProvider(tracerProvider)

	// set context text map propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

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

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(appName),
		),
	)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// newJaegerExporter returns a jaeger export otel data.
func newJaegerExporter(jaegerEndpoint string) (trace.SpanExporter, error) {
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(jaegerEndpoint),
		),
	)
	return exporter, err
}

// newStdoutExporter returns a stdout export otel data.
func newStdoutExporter() (trace.SpanExporter, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())

	return exporter, err
}
