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

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

func WarpGinHandler(handler http.Handler, appName string) http.Handler {
	return otelhttp.NewHandler(handler, appName)
}

func GinTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		Inject(c.Request.Context(), propagation.HeaderCarrier(c.Writer.Header()))

		c.Next()

		if labeler, ok := otelhttp.LabelerFromContext(c.Request.Context()); ok {
			labeler.Add(attribute.String("http.router", c.FullPath()))
		}
	}
}
