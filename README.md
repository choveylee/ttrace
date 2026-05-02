# ttrace

The `ttrace` module provides OpenTelemetry tracing helpers for Go. It bootstraps a global
`TracerProvider` (`noop`, `stdout`, or OTLP/HTTP), installs W3C Trace Context and Baggage
propagation, and exposes convenience helpers for span creation, context extraction and injection,
HTTP instrumentation, and sampling. Optional Gin integration lives in a separate module
(`github.com/choveylee/ttrace/gin`) so the core package does not depend on Gin.

## Requirements

- **Go** 1.25 or later (see `go.mod`).

## Installation

Core module:

```bash
go get github.com/choveylee/ttrace@latest
```

Optional Gin middleware (only if you use [gin-gonic/gin](https://github.com/gin-gonic/gin)):

```bash
go get github.com/choveylee/ttrace/gin@latest
```

## Features

- **Initialization:** On import, configuration is loaded through [tcfg](https://github.com/choveylee/tcfg) (typically from environment variables). The global `TracerProvider` is started for stdout or OTLP/HTTP export, or a noop provider is installed when tracing is disabled or initialization fails.
- **Propagation:** W3C Trace Context and W3C Baggage propagators are installed so `Inject`, `Extract`, and `ExtractHTTP` behave consistently.
- **Resource attributes** (semconv v1.40.0): `service.name` is always set and defaults to the executable base name when unset. Optional attributes include `service.version`, `service.namespace`, `service.instance.id`, and `deployment.environment.name`.
- **Helper APIs:** Span helpers (`Start`), HTTP extraction (`ExtractHTTP`), manual context injection (`InjectTrace`, `InjectRemoteTrace`, `InjectContext`), baggage helpers (`ContextWithBaggage`), `Shutdown`, and more.
- **Sampling:** Configurable trace-ID ratio sampling can be combined with a per-second throughput cap (`GuaranteedThroughputProbabilitySampler`). Set either knob to `-1` to disable that stage, or set both to `-1` to enable always-on sampling.

**Endpoint:** Set **`TRACER_OTLP_ENDPOINT`** to the OTLP/HTTP `host:port` (for example, the
collector OTLP port). This is **not** the legacy Jaeger agent UDP protocol.

## Configuration

Values are resolved through `tcfg`, usually from environment variables. The following keys are used
by the package:

| Key | Description |
|-----|-------------|
| `TRACER_MODE` | `0` = disabled (`noop`), `1` = stdout exporter, `2` = OTLP/HTTP exporter |
| `TRACER_OTLP_ENDPOINT` | OTLP/HTTP `host:port` (for example `localhost:4318`). TLS is not enabled by default in the current client. |
| `TRACER_SAMPLING_FRACTION` | Trace-ID ratio sampler value (for example `0.1`). Use values `>= 0`, or **`-1`** to disable the ratio stage. |
| `TRACER_MAX_TRACES_PER_SEC` | Upper bound on sampled root traces per second after the ratio stage. Use values `>= 0`, or **`-1`** to disable the throughput cap. |
| `APP_NAME` | Maps to `service.name`. When empty, the executable base name is used. |
| `SERVICE_VERSION` | Optional `service.version` attribute. |
| `SERVICE_NAMESPACE` | Optional `service.namespace` attribute. |
| `SERVICE_INSTANCE_ID` | Optional `service.instance.id` attribute. |
| `DEPLOYMENT_ENVIRONMENT_NAME` | Optional `deployment.environment.name` attribute (for example `production`). |

If tracer initialization fails, the package logs the error and falls back to **noop** tracing while
keeping propagators installed. Sampling values below `-1` are treated as invalid configuration and
therefore trigger the same fallback behavior.

## Usage

**Shutdown** (recommended when an SDK-backed exporter is active):

```go
defer func() { _ = ttrace.Shutdown() }()
```

**Create a span:**

```go
ctx, span := ttrace.Start(ctx, "operation")
defer span.End()
```

**Incoming HTTP**: Prefer standard headers (`traceparent`, `tracestate`, and optional `baggage`).

```go
ctx := ttrace.ExtractHTTP(r.Context(), r.Header)
```

**Outgoing requests**: Inject into a carrier such as request headers.

```go
ttrace.Inject(ctx, carrier)
```

**`net/http` server wrapper** ([otelhttp](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)):

```go
h := ttrace.WrapHandler(yourHandler, "users.handler")
```

**Gin**: Use the optional submodule. [otelgin](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin)
performs extraction, so do not call `ExtractHTTP` again for the same request unless duplicate
processing is intentional.

```go
import ttracegin "github.com/choveylee/ttrace/gin"

r.Use(ttracegin.Middleware("my-service"))
```

## API overview

| Symbol | Purpose |
|--------|---------|
| `ExtractHTTP` | Extract trace context from `http.Header`. |
| `Inject` / `Extract` | Use the global `TextMapPropagator` with a `propagation.TextMapCarrier`. |
| `InjectTrace` | Decode valid hexadecimal trace and span IDs into the context (local-root semantics when no valid parent span exists). |
| `InjectRemoteTrace` | Build a remote `SpanContext` from valid hexadecimal IDs and a sampled flag when full header parsing is unavailable. |
| `InjectContext` | Generate new IDs with `NewTraceId` / `NewSpanId` and call `InjectTrace`. |
| `Start`, `GetTracer`, `GetSpan`, `GetSpanContext` | Span and tracer access by using [TracerName]. |
| `SetTraceId`, `GetTraceId`, `ValidTraceId` | Trace ID helpers on `context.Context`. |
| `ContextWithBaggage`, `GetBaggage` | W3C Baggage helpers. |
| `WrapHandler` | `net/http` server instrumentation helper. |
| `GetTracerProvider` | Non-nil only when stdout or OTLP mode starts successfully. |
| `Shutdown` | Shut down the SDK `TracerProvider` when it is installed. |

## Testing

```bash
go test ./... -count=1
```

In a multi-module checkout, use the repository `go.work` or run tests from each module as needed.
