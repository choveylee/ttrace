/**
 * @Author: lidonglin
 * @Description:
 * @File:  const.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 14:00
 */

package ttrace

import (
	"go.opentelemetry.io/otel/codes"
)

const (
	AppName = "APP_NAME"

	JaegerEnable   = "JAEGER_ENABLE"
	JaegerEndpoint = "JAEGER_ENDPOINT"
)

const (
	StatusCodeUnset = codes.Unset

	StatusCodeOk    = codes.Ok
	StatusCodeError = codes.Error
)
