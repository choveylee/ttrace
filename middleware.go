package ttrace

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHandler returns an [http.Handler] instrumented with
// [go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp]. The operation argument becomes
// the HTTP server span name and should therefore be a stable, low-cardinality handler or route
// label rather than a service name.
func WrapHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}
