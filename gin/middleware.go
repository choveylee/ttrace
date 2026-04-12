package gin

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Middleware returns Gin middleware that extracts distributed trace context and records server spans.
// The appName argument is passed to otelgin as the service name for span attributes.
func Middleware(appName string) gin.HandlerFunc {
	return otelgin.Middleware(appName)
}
