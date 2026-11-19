# ttrace

The `ttrace` module provides OpenTelemetry tracing helpers for Go. It bootstraps a global
`TracerProvider` (`noop`, `stdout`, or OTLP/HTTP), installs W3C Trace Context and Baggage
propagation, and exposes convenience helpers for span creation, context extraction and injection,
HTTP instrumentation, and sampling. Optional Gin integration lives in a separate module
(`github.com/choveylee/ttrace/gin`) so the core package does not depend on Gin.

## Requirements

- **Go** 1.27.4 or later (core module, Gin submodule, and `go.work`).

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
- **Helper APIs:** Span helpers (`Start`), HTTP extraction (`ExtractHTTP`), manual context injection (`InjectTrace`, `InjectRemoteTrace`, `InjectContext`), baggage helpers (`ContextWithBaggage`), `Shutdown`, `ShutdownContext`, and more.
- **Sampling:** Configurable trace-ID ratio sampling can be combined with a per-second throughput cap (`GuaranteedThroughputProbabilitySampler`). Set either knob to `-1` to disable that stage, or set both to `-1` to enable always-on sampling.

**Endpoint:** Set **`TRACER_OTLP_ENDPOINT`** to the OTLP/HTTP `host:port` (for example, the
collector OTLP port). This is **not** the legacy Jaeger agent UDP protocol.

## Configuration

Values are resolved through `tcfg`, usually from environment variables. The following keys are used
by the package:

| Key | Description |
|-----|-------------|
| `TRACER_MODE` | `0` = disabled (`noop`), `1` = stdout exporter, `2` = OTLP/HTTP exporter |
| `TRACER_OTLP_ENDPOINT` | OTLP/HTTP `host:port` (for example `localhost:4318` or `[::1]:4318`). The port must be in `[1, 65535]`; schemes, paths, queries, and credentials are rejected. The current client uses HTTP. |
| `TRACER_SAMPLING_FRACTION` | Finite trace-ID ratio in `[0, 1]` (for example `0.1`), or **`-1`** to disable the ratio stage. |
| `TRACER_MAX_TRACES_PER_SEC` | Finite non-negative root-trace sampling rate after the ratio stage. `0` drops all root traces; **`-1`** disables the throughput cap. |
| `APP_NAME` | Maps to `service.name`. When empty, the executable base name is used. |
| `SERVICE_VERSION` | Optional `service.version` attribute. |
| `SERVICE_NAMESPACE` | Optional `service.namespace` attribute. |
| `SERVICE_INSTANCE_ID` | Optional `service.instance.id` attribute. |
| `DEPLOYMENT_ENVIRONMENT_NAME` | Optional `deployment.environment.name` attribute (for example `production`). |

If tracer initialization fails, the package logs the error and falls back to **noop** tracing while
keeping propagators installed. Negative sampling values other than `-1`, non-finite values, and
fractions above `1` are invalid configuration and trigger the same fallback behavior.

Sampling is parent-based: valid sampled parents remain sampled even when the root rate is zero;
valid unsampled parents remain unsampled. Positive rates use a token bucket with an initial
capacity of `max(rate, 1)`; this is a refill rate with a burst allowance, not a strict rolling-window
cap. Public sampler constructors accept concrete values, so invalid values (including `-1`)
disable root sampling. Only configuration interprets `-1` as disabling a sampling stage.

Without a valid parent, `InjectContext`, `InjectTrace`, and `SetTraceId` decide sampling once when
creating the local trace context, using the same sampler as the configured SDK provider. The
decision consumes sampling quota at context creation; subsequent spans inherit it without another
deduction. Disabled tracing and noop fallback create unsampled local contexts with valid IDs.
Existing valid parents keep their flags and tracestate; `InjectRemoteTrace` preserves the supplied
remote decision. These helpers create non-recording contexts; use `Start` to record a span.

The reusable rate limiter rejects negative or non-finite costs. Invalid rates or capacities clear
its balance and disable positive-cost admissions until a valid `Update`. Its zero value is disabled;
a zero refill rate with positive capacity permits spending the initial balance without refilling.

## Usage

**Shutdown** (recommended when an SDK-backed exporter is active):

```go
defer func() { _ = ttrace.Shutdown() }()
```

`Shutdown()` supplies a ten-second deadline. Use `ShutdownContext(ctx)` to pass the application's
own deadline. A nil context returns an error; a context already canceled does not start SDK
shutdown. After shutdown begins, the SDK treats it as final even if it times out: another call does
not retry flushing. Custom span processors must honor cancellation for their shutdown to be bounded.

**Create a span:**

```go
ctx, span := ttrace.Start(ctx, "operation")
defer span.End()
```

**Incoming HTTP**: Prefer standard headers (`traceparent`, `tracestate`, and optional `baggage`).

```go
ctx := ttrace.ExtractHTTP(request.Context(), request.Header)
```

**Outgoing requests**: Inject into a carrier such as request headers.

```go
ttrace.Inject(ctx, carrier)
```

**`net/http` server wrapper** ([otelhttp](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)):

```go
handler := ttrace.WrapHandler(yourHandler, "users.handler")
```

**Gin**: Use the optional submodule. [otelgin](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin)
performs extraction using the installed propagator. Install the global provider and propagator
before creating middleware. The Gin submodule does not read `TRACER_*` configuration or initialize
tracing itself. Import the core module to enable its configuration-driven initialization:

```go
import (
	"github.com/choveylee/ttrace"
	ttracegin "github.com/choveylee/ttrace/gin"
)

defer func() { _ = ttrace.Shutdown() }()
router.Use(ttracegin.Middleware(""))
```

Applications that initialize OpenTelemetry themselves may use the Gin submodule independently:
install their own global `TracerProvider` and a `TextMapPropagator` with W3C Trace Context and
Baggage before creating middleware, and shut down their provider on exit. The middleware's
`serverName` argument sets the HTTP server address attribute; `""` uses `unknown`. An omitted port is excluded from telemetry. Set
`service.name` on the provider resource (`APP_NAME` when using the core module). Avoid calling
`ExtractHTTP` again for the same request unless duplicate processing is intentional.

## API overview

| Symbol | Purpose |
|--------|---------|
| `ExtractHTTP` | Extract trace context from `http.Header`. |
| `Inject` / `Extract` | Use the global `TextMapPropagator` with a `propagation.TextMapCarrier`. |
| `InjectTrace` | Decode valid hexadecimal trace and span IDs, applying configured sampling when no valid parent exists. |
| `InjectRemoteTrace` | Build a remote `SpanContext` from valid hexadecimal IDs and a sampled flag when full header parsing is unavailable. |
| `InjectContext` | Generate new IDs with `NewTraceId` / `NewSpanId` and call `InjectTrace`. |
| `Start`, `GetTracer`, `GetSpan`, `GetSpanContext` | Span and tracer access by using [TracerName]. |
| `SetTraceId`, `GetTraceId`, `ValidTraceId` | Trace ID helpers on `context.Context`. |
| `ContextWithBaggage`, `GetBaggage` | W3C Baggage helpers. |
| `WrapHandler` | `net/http` server instrumentation helper. |
| `GetTracerProvider` | Non-nil only when stdout or OTLP mode starts successfully. |
| `Shutdown` | Shut down the SDK `TracerProvider` when it is installed. |
| `ShutdownContext` | Shut down using the application's context and deadline. |

## Validation

```bash
gofmt -l *.go gin/*.go
GOWORK=off go build ./...
GOWORK=off go vet ./...
(cd gin && GOWORK=off go build ./... && GOWORK=off go vet ./...)
```

In a multi-module checkout, also use `go build ./... ./gin/...` and `go vet ./... ./gin/...`
to verify the repository workspace.

Gin middleware uses the configured server address for telemetry. An empty name becomes `unknown`, and an omitted or invalid port is excluded from telemetry. Supply `server.example:8080` to record a concrete endpoint. Request Host values remain available to application handlers.
