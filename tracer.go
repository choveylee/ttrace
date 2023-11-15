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
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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
	ctx := context.Background()

	tracerMode := tcfg.DefaultInt(tcfg.LocalKey(TracerMode), TracerModeDisable)

	err := startTracer(ctx, tracerMode)
	if err != nil {
		log.Printf("init err (start tracer %v).", err)
	}
}

func GetTracerProvider() *trace.TracerProvider {
	return tracerProvider
}

func Shutdown() error {
	if tracerProvider != nil {
		err := tracerProvider.Shutdown(context.Background())
		if err != nil {
			log.Printf("shut down err (%v).", err)

			return err
		}
	}

	return nil
}

func startTracer(ctx context.Context, tracerMode int) error {
	// init resource
	resource, err := newResource()
	if err != nil {
		log.Printf("start tracer err (new resource %v).", err)

		return nil
	}

	var tracerExporter trace.SpanExporter

	if tracerMode == TracerModeStdout {
		// init stdout exporter
		tracerExporter, err = newStdoutExporter()
		if err != nil {
			log.Printf("start tracer err (new stdout exporter %v).", err)

			return err
		}
	} else if tracerMode == TracerModeJaeger {
		// init jaeger exporter
		jaegerEndpoint := tcfg.DefaultString(tcfg.LocalKey(JaegerEndpoint), "")
		if jaegerEndpoint == "" {
			log.Printf("start tracer err (jaeger endpoint illegal).")

			return fmt.Errorf("jaeger endpoint illegal")
		}

		tracerExporter, err = newTraceExporter(ctx, jaegerEndpoint)
		if err != nil {
			log.Printf("start tracer (%s) err (new jaeger exporter %v).", jaegerEndpoint, err)

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
		trace.WithBatcher(
			tracerExporter,
			// trace.WithMaxExportBatchSize(maxExportBatchSize),
			// trace.WithBatchTimeout(exportBatchCron),
		),
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

	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(appName),
	)

	return r, nil
}

// newTraceExporter returns a trace exporter.
func newTraceExporter(ctx context.Context, jaegerEndpoint string) (*otlptrace.Exporter, error) {
	client := otlptracehttp.NewClient(
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(jaegerEndpoint),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}

	return exporter, err
}

// newStdoutExporter returns a stdout export otel data.
func newStdoutExporter() (trace.SpanExporter, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())

	return exporter, err
}
