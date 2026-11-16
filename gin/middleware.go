package gin

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Middleware returns a [gin.HandlerFunc] that extracts incoming trace context and records server
// spans. Install a global TracerProvider and TextMapPropagator before creating the middleware.
// serverName supplies the HTTP server address attribute; an empty value uses the request Host.
// Configure service.name on the provider's resource.
func Middleware(serverName string) gin.HandlerFunc {
	return otelgin.Middleware(serverName)
}
