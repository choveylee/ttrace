package ttrace

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWrapHandler(t *testing.T) {
	t.Parallel()

	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	wrapped := WrapHandler(h, "test-app")
	if wrapped == nil {
		t.Fatal("WrapHandler returned nil")
	}
}

func TestGinTrace(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	mw := GinTrace("test-app")
	if mw == nil {
		t.Fatal("GinTrace returned nil")
	}
}
