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

// init loads tracing configuration through tcfg, installs an SDK-backed TracerProvider when
// possible, and falls back to noop tracing if initialization fails.
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

// GetTracerProvider returns the package-level [*sdktrace.TracerProvider] when stdout or OTLP startup
// succeeds. It returns nil when tracing is disabled or the noop fallback is active.
func GetTracerProvider() *sdktrace.TracerProvider {
	return tracerProvider
}

// Shutdown flushes and shuts down the SDK [sdktrace.TracerProvider] returned by
// [GetTracerProvider], when present.
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

// startTracer constructs the resource, exporter, and sampler, then installs the global
// TracerProvider for stdout or OTLP/HTTP export. Disabled or unknown modes delegate to
// [installNoopTracing]. It returns an error when resource, sampler, or exporter construction fails.
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

	samplingFraction := tcfg.DefaultFloat64(tcfg.LocalKey(TracerSamplingFraction), 0.1)
	maxTracesPerSecond := tcfg.DefaultFloat64(tcfg.LocalKey(TracerMaxTracesPerSec), 1.0)
	sampler, err := configuredSampler(samplingFraction, maxTracesPerSecond)
	if err != nil {
		log.Printf("start tracer err (configure sampler %v).", err)

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

			return fmt.Errorf("ttrace: missing %s for OTLP exporter", OTLPEndpoint)
		}

		tracerExporter, err = newTraceExporter(ctx, otlpEndpoint)
		if err != nil {
			log.Printf("start tracer OTLP (%s) err (new exporter %v).", otlpEndpoint, err)

			return err
		}
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

// configuredSampler interprets -1 as disabling an individual sampling stage. When both values are
// -1, root sampling is always enabled. Values below -1 are rejected. The returned sampler is
// parent-based so child spans inherit their parent decision.
func configuredSampler(samplingFraction, maxTracesPerSecond float64) (sdktrace.Sampler, error) {
	if err := validateSamplingConfigValue(TracerSamplingFraction, samplingFraction); err != nil {
		return nil, err
	}

	if err := validateSamplingConfigValue(TracerMaxTracesPerSec, maxTracesPerSecond); err != nil {
		return nil, err
	}

	switch {
	case samplingFraction == -1 && maxTracesPerSecond == -1:
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case samplingFraction == -1:
		return RateLimitingSampler(maxTracesPerSecond), nil
	case maxTracesPerSecond == -1:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingFraction)), nil
	default:
		return GuaranteedThroughputProbabilitySampler(samplingFraction, maxTracesPerSecond), nil
	}
}

func validateSamplingConfigValue(key string, value float64) error {
	if value < -1 {
		return fmt.Errorf("ttrace: invalid %s: must be -1 or >= 0 (got %v)", key, value)
	}

	return nil
}

// installNoopTracing registers a noop global TracerProvider, clears the SDK provider pointer, and
// reinstalls propagators so context propagation remains available without exporting spans.
func installNoopTracing() error {
	tracerProvider = nil

	otel.SetTracerProvider(noop.NewTracerProvider())
	installPropagator()

	return nil
}

// installPropagator installs a global TextMapPropagator composed of W3C Trace Context and W3C
// Baggage propagators.
func installPropagator() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

// newResource builds a [resource.Resource] containing service.name and optional attributes derived
// from tcfg keys such as [ServiceVersion] and [DeploymentEnvironmentName].
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

// newTraceExporter creates an OTLP/HTTP trace exporter for endpoint (host:port) using an insecure
// client configuration.
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

// newStdoutExporter constructs a stdout span exporter with pretty-print formatting.
func newStdoutExporter() (sdktrace.SpanExporter, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())

	return exporter, err
}
