// Package ttrace provides OpenTelemetry integration for distributed tracing: W3C Trace Context and
// Baggage propagation, injecting and extracting trace identifiers on [context.Context], creating
// spans, and configuring the global [go.opentelemetry.io/otel.TracerProvider] (stdout or OTLP export,
// with noop fallback). The instrumentation scope name is [TracerName].
//
// During package initialization, the package registers the global TracerProvider and TextMapPropagator,
// using configuration keys from [github.com/choveylee/tcfg] such as [TracerMode], [OTLPEndpoint], and
// [AppName]. Call [Shutdown] before process exit to flush pending spans when the SDK provider is active.
//
// For [github.com/gin-gonic/gin], use subpackage [github.com/choveylee/ttrace/gin] so applications
// that only use [net/http] do not depend on Gin.
package ttrace
