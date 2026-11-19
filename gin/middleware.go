package gin

import (
	"net"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Middleware records server spans using the configured server address.
// Missing names become "unknown"; missing or invalid ports are omitted from telemetry.
// Provider and propagator initialization remain the application's responsibility.
// A concrete port prevents otelgin from falling back to the untrusted request Host.
func Middleware(serverName string) gin.HandlerFunc {
	serverName = strings.TrimSpace(serverName)
	host, port, err := net.SplitHostPort(serverName)
	if err != nil {
		host = strings.Trim(serverName, "[]")
		port = "0"
	}

	if host == "" {
		host = "unknown"
	}

	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		port = "0"
	}

	return otelgin.Middleware(net.JoinHostPort(host, port))
}
