package gin

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Middleware returns a [gin.HandlerFunc] that extracts incoming trace context and records server
// spans. The appName argument is forwarded to otelgin as the service name reported on span
// attributes.
func Middleware(appName string) gin.HandlerFunc {
	return otelgin.Middleware(appName)
}
