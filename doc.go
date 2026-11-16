// Package ttrace provides OpenTelemetry tracing helpers for Go applications.
//
// The package initializes the global TracerProvider and TextMapPropagator, supports stdout and
// OTLP/HTTP exporters with noop fallback, and exposes helpers for span creation, context
// propagation, baggage handling, and manual trace-context injection. The instrumentation scope name
// used by [Start] and [GetTracer] is [TracerName].
//
// During package initialization, configuration is loaded through [github.com/choveylee/tcfg] using
// keys such as [TracerMode], [OTLPEndpoint], and [AppName]. Call [Shutdown] before process exit
// when an SDK-backed provider is active so pending spans are flushed. Shutdown supplies a
// ten-second deadline; [ShutdownContext] accepts an application-specific context. SDK shutdown
// cannot retry a flush after a timeout. Invalid sampling configuration falls back to noop tracing,
// while zero root-sampling rates preserve valid parent sampling decisions.
// Manual local-context creation without a valid parent uses the configured sampler once;
// subsequent spans inherit that decision without consuming another root-sampling credit.
//
// Applications that use [github.com/gin-gonic/gin] should import subpackage
// [github.com/choveylee/ttrace/gin]. Initialize tracing before creating that middleware, either
// through this core module or through an application-owned provider and propagator.
// The core module itself does not depend on Gin.
package ttrace
