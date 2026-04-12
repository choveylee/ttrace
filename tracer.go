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
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	tracerProvider *sdktrace.TracerProvider
)

// init loads tracer configuration from tcfg, starts the SDK or a noop TracerProvider, and on
// failure invokes installNoopTracing to continue with noop tracing.
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

// GetTracerProvider returns the package-level [*sdktrace.TracerProvider] after successful stdout or
// OTLP startup. It returns nil when tracing is disabled or the noop fallback is active.
func GetTracerProvider() *sdktrace.TracerProvider {
	return tracerProvider
}

// Shutdown flushes and shuts down the SDK TracerProvider returned by [GetTracerProvider] when that
// value is non-nil.
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

// startTracer constructs the resource, span exporter, and sampler, then installs the global
// TracerProvider for stdout or OTLP export. Disabled or unknown modes delegate to
// [installNoopTracing]. It returns an error if resource or exporter construction fails.
func startTracer(ctx context.Context, tracerMode int) error {
	if tracerMode != TracerModeStdout && tracerMode != TracerModeOTLP {
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
		otlpEndpoint := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(OTLPEndpoint), ""))
		if otlpEndpoint == "" {
			log.Printf("start tracer err (empty OTLP endpoint: set %s).", OTLPEndpoint)

			return fmt.Errorf("ttrace: OTLP endpoint is empty (set %s)", OTLPEndpoint)
		}

		tracerExporter, err = newTraceExporter(ctx, otlpEndpoint)
		if err != nil {
			log.Printf("start tracer OTLP (%s) err (new exporter %v).", otlpEndpoint, err)

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

// installNoopTracing registers a noop global TracerProvider, clears the SDK provider pointer, and
// reinstalls propagators so context propagation continues to function without exporting spans.
func installNoopTracing() error {
	tracerProvider = nil

	otel.SetTracerProvider(noop.NewTracerProvider())
	installPropagator()

	return nil
}

// installPropagator sets the global TextMapPropagator to a composite of W3C Trace Context and W3C Baggage.
func installPropagator() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

// newResource builds a [resource.Resource] with service.name and optional attributes derived from
// tcfg keys including [ServiceVersion] and [DeploymentEnvironmentName].
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

// newTraceExporter creates an OTLP/HTTP trace exporter for endpoint (host:port) using an insecure client.
func newTraceExporter(ctx context.Context, endpoint string) (*otlptrace.Exporter, error) {
	client := otlptracehttp.NewClient(
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(endpoint),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}

	return exporter, err
}

// newStdoutExporter constructs a stdout span exporter that prints OTLP-style span data with pretty formatting.
func newStdoutExporter() (sdktrace.SpanExporter, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())

	return exporter, err
}
