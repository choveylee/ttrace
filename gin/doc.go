// Package gin provides OpenTelemetry middleware for the Gin web framework by wrapping otelgin from
// [go.opentelemetry.io/contrib].
//
// Import this package only in applications that already depend on
// [github.com/gin-gonic/gin]. The core [github.com/choveylee/ttrace] module does not require Gin.
// This submodule does not initialize tracing or read TRACER_* configuration. Import the core
// module for configuration-driven initialization, or install your own global OpenTelemetry
// provider and propagator before creating [Middleware].
package gin
