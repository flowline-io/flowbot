# Distributed Tracing

OpenTelemetry-based distributed tracing for end-to-end visibility of requests, database queries, Redis commands, external API calls, pipeline executions, and Watermill event flows.

Source: `pkg/trace/` (core), with instrumentation spread across `pkg/event/`, `pkg/pipeline/`, `pkg/ability/`, `pkg/utils/`, `pkg/rdb/`, `pkg/flog/`, `internal/store/mysql/`, `internal/server/`.

## Architecture

```
                    ┌─────────────────────────────────┐
                    │       OTLP Collector            │
                    │  (Jaeger / Tempo / Datadog)     │
                    └──────────────┬──────────────────┘
                                   │ OTLP HTTP (protobuf)
                    ┌──────────────▼──────────────────┐
                    │   TracerProvider (SDK)           │
                    │   - BatchSpanProcessor           │
                    │   - OTLP HTTP exporter            │
                    │   - Resource (service.name, etc.) │
                    └──────────────┬──────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
  ┌──────▼──────┐   ┌──────────────▼──────┐   ┌──────────────▼──────┐
  │ Fiber OTel  │   │  Pipeline Engine    │   │  ability.Invoke     │
  │ middleware  │   │  (custom spans)     │   │  (custom spans)     │
  └──────┬──────┘   └──────────┬──────────┘   └──────────┬──────────┘
         │                     │                         │
  ┌──────▼──────┐   ┌──────────▼──────────┐   ┌──────────▼──────────┐
  │ GORM plugin │   │ Watermill handler   │   │ resty + otelhttp    │
  │ (DB traces) │   │ (pub/sub traces)    │   │ (outgoing HTTP)     │
  └─────────────┘   └─────────────────────┘   └─────────────────────┘
```

### Components

| Component | File | Role |
| --------- | ---- | ---- |
| `TracerProvider` | `pkg/trace/trace.go` | OTLP HTTP exporter init, sampler, lifecycle |
| Fiber middleware | `pkg/trace/fiber.go` | HTTP request spans, W3C context extraction |
| Span helpers | `pkg/trace/helper.go` | `StartSpan`, `RecordError`, `SetSpanAttributes` |
| GORM plugin | `internal/store/mysql/adapter.go` | Auto-span for all GORM queries |
| Redis hook | `pkg/rdb/rdb.go`, `pkg/event/redis.go` | Auto-span for all Redis commands |
| Pipeline spans | `pkg/pipeline/engine.go` | Pipeline + step execution spans |
| ability.Invoke span | `pkg/ability/invoke.go` | Capability invocation span |
| HTTP client | `pkg/utils/resty.go` | `otelhttp` transport for outgoing HTTP |
| Watermill trace | `pkg/event/pubsub.go` | Publish span + consumer span + W3C propagation |
| Log correlation | `pkg/flog/flog.go` | `Ctx(ctx)` annotates log entries with `trace_id` / `span_id` |
| Trace context | `pkg/types/context.go` | `TraceCtx` field in `types.Context` |

## Span Naming Convention

Spans follow a hierarchical dot-separated naming scheme. Each layer prefixes its span with the component namespace.

| Level | Span name | Location | Automatic |
| ----- | --------- | -------- | --------- |
| HTTP request | `HTTP {method} {route}` | `trace/fiber.go` | Yes |
| Event publish | `event.publish {topic}` | `event/pubsub.go` | Yes |
| Event consume | `event.receive {topic}` | `event/pubsub.go` | Yes |
| Pipeline execute | `pipeline.{name}.execute` | `pipeline/engine.go` | Yes |
| Pipeline step | `pipeline.{pipeline}.step.{step}` | `pipeline/engine.go` | Yes |
| Ability invoke | `ability.{capability}.{operation}` | `ability/invoke.go` | Yes |
| GORM query | `gorm.Query` / `gorm.Row` / `gorm.Transaction` | GORM plugin | Yes |
| Redis command | `GET` / `SET` / `LPUSH` / `XADD` / ... | redisotel hook | Yes |
| Outgoing HTTP | `HTTP {method}` | otelhttp transport | Yes |

### Span attribute conventions

| Span type | Key attributes |
| --------- | -------------- |
| HTTP server | `http.method`, `http.route`, `http.target`, `net.host.name`, `http.scheme`, `http.status_code` |
| Event publish | `messaging.destination`, `messaging.message.id` |
| Event consume | `messaging.operation` (`receive`), `messaging.destination`, `messaging.message.id` |
| Pipeline execute | `pipeline.name`, `event.id`, `event.type` |
| Pipeline step | `pipeline.step.name`, `pipeline.step.capability`, `pipeline.step.operation` |
| Ability invoke | `capability.name`, `capability.operation` |
| GORM | `db.system` (`mysql`), `db.statement`, `db.rows_affected` |
| Redis | `db.system` (`redis`), `db.statement` |
| Outgoing HTTP | `http.method`, `http.url`, `net.peer.name`, `http.status_code` |

## Call Chain

### Trace 1: Chat message → module → external API

```
HTTP POST /service/{module}/command          ← Fiber middleware span
  │
  ├── gorm.Query (user lookup)               ← GORM auto-span
  ├── GET (redis:get chat session)           ← Redis auto-span
  ├── ability.{capability}.{operation}       ← ability.Invoke span
  │     ├── HTTP GET https://api.example.com ← otelhttp auto-span
  │     └── gorm.Query (data fetch)          ← GORM auto-span
  └── event.publish message:send             ← PublishMessage span
        │
        └── [cross-process via W3C traceparent in metadata]
              │
              event.receive message:send     ← TraceConsumerMiddleware span
                └── gorm.Query (platform lookup) ← GORM auto-span
```

### Trace 2: Pipeline execution from durable event

```
event.receive pipeline:data_event            ← TraceConsumerMiddleware span
  │
  └── pipeline.{name}.execute               ← Pipeline engine span
        ├── gorm.Query (consumption check)   ← GORM auto-span
        ├── gorm.Query (create run)          ← GORM auto-span
        ├── pipeline.{name}.step.{step1}     ← Step span
        │     └── ability.{cap}.{operation}  ← ability.Invoke span
        │           └── HTTP GET ...         ← otelhttp auto-span
        ├── pipeline.{name}.step.{step2}     ← Step span
        │     └── ability.{cap}.{operation}
        └── gorm.Query (update run status)   ← GORM auto-span
```

### Trace 3: Webhook → pipeline

```
HTTP POST /webhook/{id}                      ← Fiber middleware span
  │
  ├── gorm.Query (webhook lookup)            ← GORM auto-span
  ├── ability.{cap}.{operation}              ← ability.Invoke span
  │     └── event.publish {data_event}       ← Emitted DataEvent span
  │           │
  │           └── [cross-process via Watermill]
  │                 │
  │                 event.receive pipeline:data_event
  │                   └── pipeline.{name}.execute
  │                         └── ... (steps as in Trace 2)
  └── HTTP 200 OK
```

## W3C Trace Context Propagation

Trace context flows through the system via two mechanisms:

1. **HTTP**: W3C `traceparent` and `tracestate` headers extracted by the Fiber middleware and injected by `otelhttp.Transport` on outgoing requests.

2. **Watermill (Redis Stream)**: `PublishMessage` injects `traceparent` into message metadata via `otel.GetTextMapPropagator().Inject()`. `TraceConsumerMiddleware` extracts it on the consumer side with `prop.Extract()`, restoring the parent-child span relationship across process boundaries.

```
Publish side:
  ctx (with span) → Inject() → msg.Metadata["traceparent"] = "00-..."

Consume side:
  msg.Metadata["traceparent"] → Extract() → ctx (restored span context)
```

## Log Correlation

Use `flog.Ctx(ctx)` to annotate log entries with `trace_id` and `span_id` from the current OpenTelemetry span:

```go
flog.Ctx(ctx).Info().Msg("processing event")
// Output: {"level":"info","trace_id":"abc...","span_id":"def...","message":"processing event"}
```

When both `trace_id` and `span_id` are present in logs, Jaeger/Tempo/Grafana can correlate log lines to specific spans.

## Configuration

```yaml
# flowbot.yaml
tracing:
  enabled: false                         # Set to true to enable trace export
  endpoint: "http://localhost:4318/v1/traces"  # OTLP HTTP endpoint
  service_name: "flowbot"                # Service name in traces
  environment: "development"             # deployment.environment attribute
  sample_rate: 1.0                       # 1.0 = all traces, 0.1 = 10%
```

### Collector endpoints

| Backend | Endpoint |
| ------- | -------- |
| Jaeger (OTLP) | `http://localhost:4318/v1/traces` |
| Grafana Tempo | `http://localhost:4318/v1/traces` |
| Datadog Agent | `http://localhost:4318/v1/traces` |
| Grafana Cloud | `https://otlp-gateway-{region}.grafana.net/otlp/v1/traces` |

## Performance

| Mode | Overhead |
| ---- | -------- |
| Disabled (`enabled: false`) | Zero — noop TracerProvider, no allocations |
| Enabled, 100% sampling | < 1% throughput impact (batch export, async) |
| Enabled, 10% sampling | Negligible |

Skipped paths (`/livez`, `/readyz`, `/healthz`, `/metrics`) create no spans, preventing noise from health-check and metrics scraping traffic.

## Development

### Running a local collector

```bash
# Jaeger all-in-one with OTLP HTTP
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest

# View traces at http://localhost:16686
```

### Verification

```bash
# Start server with tracing enabled
go run ./cmd/main.go

# Send a request and check for trace_id in response headers
curl -v http://localhost:8888/livez

# Check Jaeger UI for traces
open http://localhost:16686
```

### Testing

```bash
go test ./pkg/trace/...
go test ./pkg/event/...      # Watermill trace tests
go test ./pkg/pipeline/...   # Pipeline span tests
go test ./pkg/flog/...       # Log correlation tests
```
