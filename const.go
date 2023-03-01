/**
 * @Author: lidonglin
 * @Description:
 * @File:  const.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 14:00
 */

package ttrace

const (
	AppName = "APP_NAME"

	TracerMode = "TRACER_MODE"

	JaegerEndpoint = "TRACER_JAEGER_ENDPOINT"

	TracerSamplingFraction = "TRACER_SAMPLING_FRACTION"
	TracerMaxTracesPerSec  = "TRACER_MAX_TRACES_PER_SEC"
)

const (
	TracerModeDisable = iota
	TracerModeStdout
	TracerModeJaeger
)
