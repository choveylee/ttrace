/**
 * @Author: lidonglin
 * @Description:
 * @File:  const.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 14:00
 */

package ttrace

// Configuration key strings for github.com/choveylee/tcfg (often environment variables).
const (
	// AppName selects service.name when set; otherwise the executable base name is used.
	AppName = "APP_NAME"

	// ServiceVersion is optional; maps to OpenTelemetry service.version.
	ServiceVersion = "SERVICE_VERSION"
	// ServiceNamespace optional; service.namespace (e.g. Kubernetes namespace).
	ServiceNamespace = "SERVICE_NAMESPACE"
	// ServiceInstanceID optional; service.instance.id (pod name, hostname, etc.).
	ServiceInstanceID = "SERVICE_INSTANCE_ID"
	// DeploymentEnvironmentName optional; deployment.environment.name (e.g. production, staging).
	DeploymentEnvironmentName = "DEPLOYMENT_ENVIRONMENT_NAME"

	TracerMode = "TRACER_MODE"

	JaegerEndpoint = "TRACER_JAEGER_ENDPOINT"

	TracerSamplingFraction = "TRACER_SAMPLING_FRACTION"
	TracerMaxTracesPerSec  = "TRACER_MAX_TRACES_PER_SEC"
)

// TracerModeDisable, TracerModeStdout, and TracerModeJaeger are the allowed integer values for key TracerMode (TRACER_MODE).
const (
	TracerModeDisable = iota
	TracerModeStdout
	TracerModeJaeger
)
