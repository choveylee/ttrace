/**
 * @Author: lidonglin
 * @Description: OpenTelemetry TracerProvider setup (noop, stdout, OTLP), resource attributes, propagator.
 * @File: tracer.go
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	tracerProvider *sdktrace.TracerProvider
)

// init reads tracer configuration, starts the configured SDK provider or noop fallback,
// and on start failure installs noop tracing via [installNoopTracing].
func init() {
	ctx := context.Background()

	tracerMode := tcfg.DefaultInt(tcfg.LocalKey(TracerMode), TracerModeDisable)

	err := startTracer(ctx, tracerMode)
	if err != nil {
		log.Printf("init err (start tracer %v), falling back to noop tracing.", err)

		err = installNoopTracing()
		if err != nil {
			log.Printf("init err (install noop tracing %v).", err)
		}
	}
}

// GetTracerProvider returns the package-level SDK TracerProvider when stdout or OTLP mode
// succeeded, or nil when tracing is disabled or running in noop fallback.
func GetTracerProvider() *sdktrace.TracerProvider {
	return tracerProvider
}

// Shutdown flushes and shuts down the SDK TracerProvider if one was installed; it is a no-op
// when GetTracerProvider returns nil.
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

// startTracer configures resource, exporter, sampler, and global TracerProvider for stdout or OTLP
// modes. Disable or unknown modes return [installNoopTracing]. Errors are returned for resource
// or exporter construction failures.
func startTracer(ctx context.Context, tracerMode int) error {
	if tracerMode != TracerModeStdout && tracerMode != TracerModeJaeger {
		if tracerMode != TracerModeDisable {
			log.Printf("start tracer: unknown trace mode %d", tracerMode)
		}

		return installNoopTracing()
	}

	res, err := newResource()
	if err != nil {
		log.Printf("start tracer err (new resource %v).", err)

		return err
	}

	var tracerExporter sdktrace.SpanExporter

	if tracerMode == TracerModeStdout {
		tracerExporter, err = newStdoutExporter()
		if err != nil {
			log.Printf("start tracer err (new stdout exporter %v).", err)

			return err
		}
	} else {
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

	sampler := sdktrace.AlwaysSample()

	samplingFraction := tcfg.DefaultFloat64(tcfg.LocalKey(TracerSamplingFraction), 0.1)
	maxTracesPerSecond := tcfg.DefaultFloat64(tcfg.LocalKey(TracerMaxTracesPerSec), 1.0)

	if samplingFraction != -1 && maxTracesPerSecond != -1 {
		sampler = GuaranteedThroughputProbabilitySampler(samplingFraction, maxTracesPerSecond)
	}

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(tracerExporter),
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	installPropagator()

	return nil
}

// installNoopTracing sets the global TracerProvider to a noop implementation, clears the SDK pointer,
// and reinstalls the composite propagator so context propagation still works without exporting spans.
func installNoopTracing() error {
	tracerProvider = nil

	otel.SetTracerProvider(noop.NewTracerProvider())
	installPropagator()

	return nil
}

// installPropagator sets the global TextMapPropagator to W3C trace context plus baggage.
func installPropagator() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

// newResource builds an OpenTelemetry [resource.Resource] with service.name (required) and optional
// service metadata from configuration keys such as [ServiceVersion] and [DeploymentEnvironmentName].
func newResource() (*resource.Resource, error) {
	appName := tcfg.DefaultString(AppName, "")
	if appName == "" {
		basePath := filepath.Base(os.Args[0])
		appName = strings.TrimSuffix(basePath, filepath.Ext(basePath))
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(appName),
	}

	version := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(ServiceVersion), ""))
	if version != "" {
		attrs = append(attrs, semconv.ServiceVersionKey.String(version))
	}

	namespace := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(ServiceNamespace), ""))
	if namespace != "" {
		attrs = append(attrs, semconv.ServiceNamespaceKey.String(namespace))
	}

	instanceId := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(ServiceInstanceID), ""))
	if instanceId != "" {
		attrs = append(attrs, semconv.ServiceInstanceIDKey.String(instanceId))
	}

	environment := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(DeploymentEnvironmentName), ""))
	if environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentNameKey.String(environment))
	}

	r := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	return r, nil
}

// newTraceExporter creates an OTLP/HTTP trace exporter targeting jaegerEndpoint (host:port, TLS not used).
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

// newStdoutExporter returns a span exporter that writes OTLP-style trace data to stdout with pretty printing.
func newStdoutExporter() (sdktrace.SpanExporter, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())

	return exporter, err
}
