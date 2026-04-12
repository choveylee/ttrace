/**
 * @Author: lidonglin
 * @Description:
 * @File:  const.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 14:00
 */

package ttrace

// Configuration keys for [github.com/choveylee/tcfg] (commonly backed by environment variables).
const (
	// AppName sets service.name; if unset, the executable base name is used.
	AppName = "APP_NAME"

	// ServiceVersion maps to OpenTelemetry service.version (optional).
	ServiceVersion = "SERVICE_VERSION"
	// ServiceNamespace maps to service.namespace (optional; e.g. Kubernetes namespace).
	ServiceNamespace = "SERVICE_NAMESPACE"
	// ServiceInstanceID maps to service.instance.id (optional).
	ServiceInstanceID = "SERVICE_INSTANCE_ID"
	// DeploymentEnvironmentName maps to deployment.environment.name (optional).
	DeploymentEnvironmentName = "DEPLOYMENT_ENVIRONMENT_NAME"

	// TracerMode selects exporter mode; values are [TracerModeDisable], [TracerModeStdout], [TracerModeOTLP].
	TracerMode = "TRACER_MODE"

	// OTLPEndpoint is the OTLP/HTTP trace endpoint (host:port) when TracerMode is [TracerModeOTLP].
	OTLPEndpoint = "TRACER_OTLP_ENDPOINT"

	// TracerSamplingFraction is the trace ID ratio for [GuaranteedThroughputProbabilitySampler].
	TracerSamplingFraction = "TRACER_SAMPLING_FRACTION"
	// TracerMaxTracesPerSec caps throughput after the ratio stage (traces per second).
	TracerMaxTracesPerSec = "TRACER_MAX_TRACES_PER_SEC"
)

// Tracer mode values for [TracerMode] / TRACER_MODE.
const (
	TracerModeDisable = iota
	TracerModeStdout
	TracerModeOTLP
)
