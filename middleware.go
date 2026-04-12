/**
 * @Author: lidonglin
 * @Description:
 * @File:  middleware.go
 * @Version: 1.0.0
 * @Date: 2023/02/27 13:59
 */

package ttrace

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHandler wraps handler with [go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp];
// appName is used as the span name prefix.
func WrapHandler(handler http.Handler, appName string) http.Handler {
	return otelhttp.NewHandler(handler, appName)
}
