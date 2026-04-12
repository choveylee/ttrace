package ttrace

// Configuration key names for [github.com/choveylee/tcfg], typically supplied via environment variables.
const (
	// AppName is the tcfg key for OpenTelemetry service.name; when unset, the executable base name is used.
	AppName = "APP_NAME"

	// ServiceVersion is the tcfg key for OpenTelemetry service.version (optional).
	ServiceVersion = "SERVICE_VERSION"
	// ServiceNamespace is the tcfg key for service.namespace (optional), for example a Kubernetes namespace.
	ServiceNamespace = "SERVICE_NAMESPACE"
	// ServiceInstanceID is the tcfg key for service.instance.id (optional).
	ServiceInstanceID = "SERVICE_INSTANCE_ID"
	// DeploymentEnvironmentName is the tcfg key for deployment.environment.name (optional).
	DeploymentEnvironmentName = "DEPLOYMENT_ENVIRONMENT_NAME"

	// TracerMode is the tcfg key that selects the trace exporter; valid values are [TracerModeDisable], [TracerModeStdout], and [TracerModeOTLP].
	TracerMode = "TRACER_MODE"

	// OTLPEndpoint is the tcfg key for the OTLP/HTTP trace endpoint (host:port) when [TracerMode] is [TracerModeOTLP].
	OTLPEndpoint = "TRACER_OTLP_ENDPOINT"

	// TracerSamplingFraction is the tcfg key for the trace ID sampling ratio passed to [GuaranteedThroughputProbabilitySampler].
	TracerSamplingFraction = "TRACER_SAMPLING_FRACTION"
	// TracerMaxTracesPerSec is the tcfg key for the per-second trace cap after ratio sampling.
	TracerMaxTracesPerSec = "TRACER_MAX_TRACES_PER_SEC"
)

// Numeric values for the [TracerMode] configuration key (environment variable TRACER_MODE).
const (
	TracerModeDisable = iota
	TracerModeStdout
	TracerModeOTLP
)
