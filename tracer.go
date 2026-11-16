package ttrace

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/choveylee/tcfg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace/noop"
)

// Tracing state is configured once during package initialization.
var (
	tracerProvider *sdktrace.TracerProvider
	tracerSampler  sdktrace.Sampler = sdktrace.NeverSample()
)

// init loads tracing configuration through tcfg, installs an SDK-backed TracerProvider when
// possible, and falls back to noop tracing if initialization fails.
func init() {
	ctx := context.Background()
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	otel.SetTextMapPropagator(propagator)

	tracerMode := tcfg.DefaultInt(tcfg.LocalKey(TracerMode), TracerModeDisable)

	err := startTracer(ctx, tracerMode)
	if err != nil {
		log.Printf("ttrace: initialize tracer: %v; falling back to noop tracing", err)
		tracerProvider = nil
		tracerSampler = sdktrace.NeverSample()
		otel.SetTracerProvider(noop.NewTracerProvider())
	}
}

// GetTracerProvider returns the package-level [*sdktrace.TracerProvider] when stdout or OTLP startup
// succeeds. It returns nil when tracing is disabled or the noop fallback is active.
func GetTracerProvider() *sdktrace.TracerProvider {
	return tracerProvider
}

// Shutdown calls [ShutdownContext] with a ten-second deadline for flushing and shutdown.
// Call ShutdownContext directly when the application's shutdown deadline differs.
func Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return ShutdownContext(ctx)
}

// ShutdownContext flushes and stops the SDK provider using a non-nil context.
// Disabled tracing returns nil for a live context. A context already canceled leaves the provider running.
// Once SDK shutdown begins, it is final even if the deadline expires; subsequent SDK shutdown
// calls do not retry the flush. Custom span processors must honor ctx to bound their shutdown.
func ShutdownContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ttrace: context must not be nil")
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("ttrace: shut down tracer provider: %w", err)
	}

	if tracerProvider == nil {
		return nil
	}

	if err := tracerProvider.Shutdown(ctx); err != nil {
		log.Printf("ttrace: shut down tracer provider: %v", err)

		return fmt.Errorf("ttrace: shut down tracer provider: %w", err)
	}

	return nil
}

// startTracer constructs the resource, exporter, and sampler, then installs the global
// TracerProvider for stdout or OTLP/HTTP export. Disabled mode installs a noop provider.
// Invalid modes, sampling settings, and exporter construction failures return errors to init.
func startTracer(ctx context.Context, tracerMode int) error {
	if tracerMode == TracerModeDisable {
		tracerSampler = sdktrace.NeverSample()
		otel.SetTracerProvider(noop.NewTracerProvider())

		return nil
	}

	if tracerMode != TracerModeStdout && tracerMode != TracerModeOTLP {
		return fmt.Errorf("ttrace: tracer mode must be %d, %d, or %d; got %d", TracerModeDisable, TracerModeStdout, TracerModeOTLP, tracerMode)
	}

	samplingFraction := tcfg.DefaultFloat64(tcfg.LocalKey(TracerSamplingFraction), 0.1)
	maxTracesPerSecond := tcfg.DefaultFloat64(tcfg.LocalKey(TracerMaxTracesPerSec), 1.0)
	if math.IsNaN(samplingFraction) || math.IsInf(samplingFraction, 0) || (samplingFraction != -1 && (samplingFraction < 0 || samplingFraction > 1)) {
		return fmt.Errorf("ttrace: sampling fraction must be -1 or a finite value between 0 and 1; got %g", samplingFraction)
	}

	if math.IsNaN(maxTracesPerSecond) || math.IsInf(maxTracesPerSecond, 0) || (maxTracesPerSecond < 0 && maxTracesPerSecond != -1) {
		return fmt.Errorf("ttrace: maximum traces per second must be -1 or a finite non-negative value; got %g", maxTracesPerSecond)
	}

	var sampler sdktrace.Sampler

	// Only configuration accepts -1; public sampler constructors require concrete non-negative values.
	switch {
	case samplingFraction == -1 && maxTracesPerSecond == -1:
		sampler = sdktrace.ParentBased(sdktrace.AlwaysSample())
	case samplingFraction == -1:
		sampler = RateLimitingSampler(maxTracesPerSecond)
	case maxTracesPerSecond == -1:
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingFraction))
	default:
		sampler = GuaranteedThroughputProbabilitySampler(samplingFraction, maxTracesPerSecond)
	}

	var tracerExporter sdktrace.SpanExporter

	if tracerMode == TracerModeStdout {
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return fmt.Errorf("ttrace: create stdout trace exporter: %w", err)
		}

		tracerExporter = exporter
	} else {
		otlpEndpoint := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(OTLPEndpoint), ""))
		if otlpEndpoint == "" {
			return errors.New("ttrace: OTLP endpoint must not be empty")
		}

		if strings.ContainsAny(otlpEndpoint, "/?#@") {
			return errors.New("ttrace: OTLP endpoint must use host:port format without a scheme, path, query, or credentials")
		}

		endpointHost, endpointPort, err := net.SplitHostPort(otlpEndpoint)
		if err != nil {
			return errors.New("ttrace: OTLP endpoint must use host:port format")
		}

		if endpointHost == "" {
			return errors.New("ttrace: OTLP endpoint host must not be empty")
		}

		port, err := strconv.ParseUint(endpointPort, 10, 16)
		if err != nil || port == 0 {
			return errors.New("ttrace: OTLP endpoint port must be an integer between 1 and 65535")
		}

		if strings.HasPrefix(otlpEndpoint, "[") {
			address, err := netip.ParseAddr(endpointHost)
			if err != nil || !address.Is6() {
				return errors.New("ttrace: bracketed OTLP endpoint host must be an IPv6 address")
			}
		}

		// Match the exporter's URL construction without exposing the configured address in errors.
		endpointURL := url.URL{Scheme: "http", Host: otlpEndpoint}

		if _, err := url.ParseRequestURI(endpointURL.String()); err != nil {
			return errors.New("ttrace: OTLP endpoint host must be a valid hostname or IP address")
		}

		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithInsecure(), otlptracehttp.WithEndpoint(otlpEndpoint))
		if err != nil {
			return fmt.Errorf("ttrace: create OTLP trace exporter: %w", err)
		}

		tracerExporter = exporter
	}

	tracingResource := newResource()
	tracerSampler = sampler
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(tracerExporter),
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(tracingResource),
	)

	otel.SetTracerProvider(tracerProvider)

	return nil
}

// newResource builds a [resource.Resource] containing service.name and optional attributes derived
// from tcfg keys such as [ServiceVersion] and [DeploymentEnvironmentName].
func newResource() *resource.Resource {
	appName := strings.TrimSpace(tcfg.DefaultString(AppName, ""))
	if appName == "" {
		basePath := filepath.Base(os.Args[0])
		appName = strings.TrimSuffix(basePath, filepath.Ext(basePath))
	}

	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String(appName),
	}

	version := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(ServiceVersion), ""))
	if version != "" {
		attributes = append(attributes, semconv.ServiceVersionKey.String(version))
	}

	namespace := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(ServiceNamespace), ""))
	if namespace != "" {
		attributes = append(attributes, semconv.ServiceNamespaceKey.String(namespace))
	}

	instanceId := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(ServiceInstanceID), ""))
	if instanceId != "" {
		attributes = append(attributes, semconv.ServiceInstanceIDKey.String(instanceId))
	}

	environment := strings.TrimSpace(tcfg.DefaultString(tcfg.LocalKey(DeploymentEnvironmentName), ""))
	if environment != "" {
		attributes = append(attributes, semconv.DeploymentEnvironmentNameKey.String(environment))
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...)
}
