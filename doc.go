// Package ttrace provides OpenTelemetry tracing helpers for Go applications.
//
// The package initializes the global TracerProvider and TextMapPropagator, supports stdout and
// OTLP/HTTP exporters with noop fallback, and exposes helpers for span creation, context
// propagation, baggage handling, and manual trace-context injection. The instrumentation scope name
// used by [Start] and [GetTracer] is [TracerName].
//
// During package initialization, configuration is loaded through [github.com/choveylee/tcfg] using
// keys such as [TracerMode], [OTLPEndpoint], and [AppName]. Call [Shutdown] before process exit
// when an SDK-backed provider is active so pending spans are flushed.
//
// Applications that use [github.com/gin-gonic/gin] should import subpackage
// [github.com/choveylee/ttrace/gin]. The core module itself does not depend on Gin.
package ttrace
