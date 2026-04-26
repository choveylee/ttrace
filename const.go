package ttrace

// Configuration key names consumed through [github.com/choveylee/tcfg], typically sourced from
// environment variables.
const (
	// AppName is the tcfg key for OpenTelemetry service.name. When unset, the executable base name is
	// used.
	AppName = "APP_NAME"

	// ServiceVersion is the tcfg key for the optional OpenTelemetry service.version attribute.
	ServiceVersion = "SERVICE_VERSION"
	// ServiceNamespace is the tcfg key for the optional service.namespace attribute, for example a
	// Kubernetes namespace.
	ServiceNamespace = "SERVICE_NAMESPACE"
	// ServiceInstanceID is the tcfg key for the optional service.instance.id attribute.
	ServiceInstanceID = "SERVICE_INSTANCE_ID"
	// DeploymentEnvironmentName is the tcfg key for the optional deployment.environment.name
	// attribute.
	DeploymentEnvironmentName = "DEPLOYMENT_ENVIRONMENT_NAME"

	// TracerMode selects the trace exporter. Valid values are [TracerModeDisable],
	// [TracerModeStdout], and [TracerModeOTLP].
	TracerMode = "TRACER_MODE"

	// OTLPEndpoint is the tcfg key for the OTLP/HTTP trace endpoint (host:port) used when
	// [TracerMode] is [TracerModeOTLP].
	OTLPEndpoint = "TRACER_OTLP_ENDPOINT"

	// TracerSamplingFraction is the tcfg key for the trace ID ratio stage used by
	// [GuaranteedThroughputProbabilitySampler]. Set it to -1 to disable ratio sampling.
	TracerSamplingFraction = "TRACER_SAMPLING_FRACTION"
	// TracerMaxTracesPerSec is the tcfg key for the per-second root-trace cap applied after ratio
	// sampling. Set it to -1 to disable the throughput cap.
	TracerMaxTracesPerSec = "TRACER_MAX_TRACES_PER_SEC"
)

// Numeric values accepted by the [TracerMode] configuration key (environment variable TRACER_MODE).
const (
	TracerModeDisable = iota
	TracerModeStdout
	TracerModeOTLP
)
