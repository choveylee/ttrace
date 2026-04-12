package ttrace

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHandler returns handler instrumented with [go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp].
// The appName argument is used as the instrumentation scope and span name prefix.
func WrapHandler(handler http.Handler, appName string) http.Handler {
	return otelhttp.NewHandler(handler, appName)
}
