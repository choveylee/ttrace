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
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func WarpHandler(handler http.Handler, appName string) http.Handler {
	return otelhttp.NewHandler(handler, appName)
}

func GinTrace(appName string) gin.HandlerFunc {
	return otelgin.Middleware(appName)
}
