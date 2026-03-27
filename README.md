# ttrace

**EN:** Small OpenTelemetry tracing helpers for Go: global `TracerProvider` bootstrap (noop / stdout / OTLP over HTTP), Gin & `net/http` middleware, baggage, propagation, and trace-context utilities.

**中文：** 基于 OpenTelemetry 的 Go 链路追踪封装：全局 `TracerProvider` 初始化（noop / 标准输出 / OTLP HTTP）、Gin 与 `net/http` 中间件、Baggage、传播层，以及 Trace 上下文工具函数。

---

## English

### Requirements

- Go **1.26.1+** (see `go.mod`)

### Install

```bash
go get github.com/choveylee/ttrace@latest
```

### Features

- Auto **init** via [tcfg](https://github.com/choveylee/tcfg): select exporter mode, OTLP endpoint, sampling, resource attributes.
- **Noop** global provider when tracing is disabled or when startup fails (with explicit fallback); **W3C Trace Context** + **Baggage** propagator installed.
- **Resource (semconv v1.40.0):** `service.name` (required), optional `service.version`, `service.namespace`, `service.instance.id`, `deployment.environment.name`.
- Helpers: **`Start`**, **`ExtractHTTP`**, **`InjectTrace`**, **`InjectRemoteTrace`**, **`SetTraceId`**, **`ContextWithBaggage`**, **`Shutdown`**, etc.
- **Note:** `TRACER_JAEGER_ENDPOINT` configures **OTLP/HTTP** (e.g. Jaeger’s OTLP collector port), not the legacy Jaeger agent UDP protocol.

### Configuration

Values are read through `tcfg` (typically environment variables). Key names in code:

| Key | Description |
|-----|-------------|
| `TRACER_MODE` | `0` = off (noop), `1` = stdout exporter, `2` = OTLP HTTP exporter |
| `TRACER_JAEGER_ENDPOINT` | OTLP HTTP endpoint, e.g. `localhost:4318` (TLS not enabled by default in code) |
| `TRACER_SAMPLING_FRACTION` | Trace ID ratio (e.g. `0.1`). Use **`-1`** together with `TRACER_MAX_TRACES_PER_SEC=-1` to force **always_sample** |
| `TRACER_MAX_TRACES_PER_SEC` | Max traces/s after passing ratio sampler (throughput ceiling) |
| `APP_NAME` | `service.name`; if empty, uses the executable base name |
| `SERVICE_VERSION` | Optional `service.version` |
| `SERVICE_NAMESPACE` | Optional `service.namespace` |
| `SERVICE_INSTANCE_ID` | Optional `service.instance.id` |
| `DEPLOYMENT_ENVIRONMENT_NAME` | Optional `deployment.environment.name` (e.g. `production`) |

On tracer **startup error**, the package falls back to **noop** tracing and logs the error.

### Usage

**Graceful shutdown (when using SDK exporter):**

```go
defer func() { _ = ttrace.Shutdown() }()
```

**Manual span:**

```go
ctx, span := ttrace.Start(ctx, "operation")
defer span.End()
```

**Incoming HTTP — prefer W3C headers (traceparent / tracestate):**

```go
ctx := ttrace.ExtractHTTP(r.Context(), r.Header)
// then Start child span or pass ctx downstream
```

For Gin, you can use **`ttrace.GinTrace("service-name")`** (otelgin) so extraction is usually already handled; avoid **double** `ExtractHTTP` on the same request.

**Outgoing propagation:** use **`ttrace.Inject`** with a `propagation.TextMapCarrier` (e.g. request headers).

**`net/http` server wrapper:**

```go
h := ttrace.WrapHandler(yourHandler, "my-service")
```

### API highlights

| Function | Role |
|----------|------|
| `ExtractHTTP` | Extract trace context from `http.Header` |
| `InjectTrace` | Set trace/span IDs from hex (local root semantics when no valid parent) |
| `InjectRemoteTrace` | Set remote parent IDs + sampled flag when you cannot use full headers |
| `InjectContext` | Generate random IDs and inject (local root) |
| `GetTracerProvider` | Non-nil only when stdout/OTLP mode started successfully |
| `Shutdown` | Shut down SDK `TracerProvider` if installed |

### Tests

```bash
go test ./... -count=1
```

---

## 中文

### 环境要求

- **Go 1.26.1+**（以 `go.mod` 为准）

### 安装

```bash
go get github.com/choveylee/ttrace@latest
```

### 功能概览

- 使用 **[tcfg](https://github.com/choveylee/tcfg)** 在包 **`init`** 中完成初始化：导出方式、OTLP 地址、采样与资源属性。
- 关闭追踪或 **启动失败** 时会回退到 **全局 noop**，并安装 **W3C Trace Context + Baggage** 传播器。
- **资源属性（semconv v1.40.0）：** 必选 `service.name`，可选 `service.version`、`service.namespace`、`service.instance.id`、`deployment.environment.name`。
- 提供 **`Start`**、**`ExtractHTTP`**、**`InjectTrace`**、**`InjectRemoteTrace`**、**`SetTraceId`**、**`ContextWithBaggage`**、**`Shutdown`** 等辅助方法。
- **说明：** `TRACER_JAEGER_ENDPOINT` 实际为 **OTLP HTTP** 端点（例如对接 Jaeger Collector 的 OTLP 端口），**不是**旧版 Jaeger Agent 的 UDP 协议。

### 配置说明

配置通过 `tcfg` 读取（多为环境变量）。代码中的键名如下：

| 键名 | 说明 |
|------|------|
| `TRACER_MODE` | `0` 关闭（noop），`1` 标准输出，`2` OTLP HTTP 导出 |
| `TRACER_JAEGER_ENDPOINT` | OTLP HTTP 端点，例如 `localhost:4318`（当前实现默认不安全连接，生产需按代码评估 TLS） |
| `TRACER_SAMPLING_FRACTION` | 基于 Trace ID 的比例采样（如 `0.1`）。与 `TRACER_MAX_TRACES_PER_SEC` 同时为 `-1` 时使用 **总是采样** |
| `TRACER_MAX_TRACES_PER_SEC` | 通过比例采样后的**每秒条数上限**（与其它语言里 “保底吞吐” 组合采样器一致） |
| `APP_NAME` | `service.name`；为空则用可执行文件名（去扩展名） |
| `SERVICE_VERSION` | 可选，`service.version` |
| `SERVICE_NAMESPACE` | 可选，`service.namespace` |
| `SERVICE_INSTANCE_ID` | 可选，`service.instance.id` |
| `DEPLOYMENT_ENVIRONMENT_NAME` | 可选，部署环境名（如 `production`） |

若 Tracer **启动失败**，会记录日志并回退为 **noop**，全局行为可预期。

### 使用示例

**退出前关闭（使用 SDK 导出时建议）：**

```go
defer func() { _ = ttrace.Shutdown() }()
```

**手动创建 Span：**

```go
ctx, span := ttrace.Start(ctx, "operation")
defer span.End()
```

**HTTP 入站 —— 优先用标准头（traceparent / tracestate）：**

```go
ctx := ttrace.ExtractHTTP(r.Context(), r.Header)
```

使用 **Gin** 时可配合 **`ttrace.GinTrace("服务名")`**（基于 otelgin），中间件已做提取时**勿重复**对同一请求调用 `ExtractHTTP`。

**出站注入：** 使用 **`ttrace.Inject`** 写入 `propagation.TextMapCarrier`（如 HTTP 请求头）。

**包装 `net/http` Handler：**

```go
h := ttrace.WrapHandler(yourHandler, "my-service")
```

### 常用 API

| 函数 | 作用 |
|------|------|
| `ExtractHTTP` | 从 `http.Header` 恢复分布式上下文 |
| `InjectTrace` | 用十六进制 trace/span id 写入上下文（无有效父上下文时按本地根处理） |
| `InjectRemoteTrace` | 无法走完整 HTTP 头时，携带 **Remote** 与 **sampled** 手工重建 |
| `InjectContext` | 随机生成 ID 并注入（本地根） |
| `GetTracerProvider` | 仅在 stdout/OTLP 成功启动时非 nil |
| `Shutdown` | 关闭已安装的 SDK `TracerProvider` |

### 运行测试

```bash
go test ./... -count=1
```
