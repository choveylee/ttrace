# ttrace

OpenTelemetry helpers for Go: global `TracerProvider` bootstrap (noop, stdout, or OTLP/HTTP), W3C propagation and baggage, HTTP utilities, and sampling helpers. Optional Gin integration lives in a separate module (`github.com/choveylee/ttrace/gin`) so the core package does not depend on Gin.

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

- **Initialization:** On import, configuration is read via [tcfg](https://github.com/choveylee/tcfg) (typically environment variables). The global `TracerProvider` is started for stdout or OTLP/HTTP export, or a noop provider is used when tracing is disabled or startup fails.
- **Propagators:** W3C Trace Context and W3C Baggage are installed so `Inject` / `Extract` / `ExtractHTTP` behave consistently.
- **Resource attributes** (semconv v1.40.0): `service.name` (required; defaults to the executable base name if unset), plus optional `service.version`, `service.namespace`, `service.instance.id`, and `deployment.environment.name`.
- **Helpers:** Spans (`Start`), HTTP extraction (`ExtractHTTP`), manual ID injection (`InjectTrace`, `InjectRemoteTrace`, `InjectContext`), baggage (`ContextWithBaggage`), `Shutdown`, and more.
- **Sampling:** Configurable ratio sampling combined with a per-second throughput cap (`GuaranteedThroughputProbabilitySampler`), or forced always-on sampling when both tuning knobs are set to `-1`.

**Endpoint:** set **`TRACER_OTLP_ENDPOINT`** to the OTLP/HTTP `host:port` (e.g. collector OTLP port). This is **not** the legacy Jaeger agent UDP protocol.

## Configuration

Values are resolved through `tcfg` (usually from environment variables). Keys used in code:

| Key | Description |
|-----|-------------|
| `TRACER_MODE` | `0` = disabled (noop), `1` = stdout exporter, `2` = OTLP/HTTP exporter |
| `TRACER_OTLP_ENDPOINT` | OTLP/HTTP `host:port` (e.g. `localhost:4318`). TLS is not enabled by default in the current client. |
| `TRACER_SAMPLING_FRACTION` | Trace ID ratio sampler (e.g. `0.1`). Set to **`-1`** together with `TRACER_MAX_TRACES_PER_SEC=-1` to use **always** sampling. |
| `TRACER_MAX_TRACES_PER_SEC` | Upper bound on sampled traces per second after the ratio stage (throughput ceiling). |
| `APP_NAME` | Maps to `service.name`; if empty, the executable base name is used. |
| `SERVICE_VERSION` | Optional `service.version`. |
| `SERVICE_NAMESPACE` | Optional `service.namespace`. |
| `SERVICE_INSTANCE_ID` | Optional `service.instance.id`. |
| `DEPLOYMENT_ENVIRONMENT_NAME` | Optional `deployment.environment.name` (e.g. `production`). |

If tracer startup fails, the package logs the error and falls back to **noop** tracing while keeping propagators installed.

## Usage

**Shutdown** (recommended when an SDK exporter is active):

```go
defer func() { _ = ttrace.Shutdown() }()
```

**Create a span:**

```go
ctx, span := ttrace.Start(ctx, "operation")
defer span.End()
```

**Incoming HTTP — prefer standard headers (`traceparent`, `tracestate`, optional `baggage`):**

```go
ctx := ttrace.ExtractHTTP(r.Context(), r.Header)
```

**Outgoing requests — inject into a carrier (e.g. headers):**

```go
ttrace.Inject(ctx, carrier)
```

**`net/http` server wrapper** ([otelhttp](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)):

```go
h := ttrace.WrapHandler(yourHandler, "my-service")
```

**Gin** (optional submodule; [otelgin](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin) performs extraction; do not call `ExtractHTTP` again for the same request unless you intend duplicate processing):

```go
import ttracegin "github.com/choveylee/ttrace/gin"

r.Use(ttracegin.Middleware("my-service"))
```

## API overview

| Symbol | Purpose |
|--------|---------|
| `ExtractHTTP` | Extract trace context from `http.Header`. |
| `Inject` / `Extract` | Use the global `TextMapPropagator` with a `propagation.TextMapCarrier`. |
| `InjectTrace` | Decode hex trace and span IDs into the context (local-root semantics when no valid parent span). |
| `InjectRemoteTrace` | Build a remote `SpanContext` from hex IDs and a sampled flag when full header parsing is unavailable. |
| `InjectContext` | Generate new IDs with `NewTraceId` / `NewSpanId` and call `InjectTrace`. |
| `Start`, `GetTracer`, `GetSpan`, `GetSpanContext` | Span and tracer access using [TracerName]. |
| `SetTraceId`, `GetTraceId`, `ValidTraceId` | Trace ID helpers on `context.Context`. |
| `ContextWithBaggage`, `GetBaggage` | W3C Baggage helpers. |
| `WrapHandler` | HTTP server instrumentation. |
| `GetTracerProvider` | Non-nil only when stdout or OTLP mode started successfully. |
| `Shutdown` | Shut down the SDK `TracerProvider` when installed. |

## Testing

```bash
go test ./... -count=1
```

In a multi-module checkout, use the repository `go.work` or run tests from each module as needed.
