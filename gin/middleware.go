/**
 * @Author: lidonglin
 * @Description: Optional Gin middleware (separate module; core ttrace does not depend on Gin).
 * @File:  middleware.go
 * @Version: 1.0.0
 * @Date: 2026/04/12
 */

// Package gin registers OpenTelemetry middleware for Gin via [otelgin]. Import this module only
// when using [github.com/gin-gonic/gin]; the [github.com/choveylee/ttrace] module has no Gin dependency.
package gin

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Middleware is the otelgin server middleware for Gin: extracts trace context and records spans using appName.
func Middleware(appName string) gin.HandlerFunc {
	return otelgin.Middleware(appName)
}
