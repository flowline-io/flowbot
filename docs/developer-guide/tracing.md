# Distributed Tracing

OpenTelemetry-based distributed tracing for end-to-end visibility of requests, database queries, Redis commands, external API calls, pipeline executions, chatagent runs, and Watermill event flows.

Source: `pkg/trace/` (core), with instrumentation in `pkg/event/`, `pkg/pipeline/`, `pkg/capability/`, `pkg/utils/`, `pkg/rdb/`, `pkg/agent/`, `pkg/flog/`, `internal/store/postgres/`, `internal/server/`.

## Architecture

```
                    ┌─────────────────────────────────┐
                    │       OTLP Collector            │
                    │  (Jaeger / Tempo / Datadog)     │
                    └──────────────┬──────────────────┘
                                   │ OTLP HTTP (protobuf)
                    ┌──────────────▼──────────────────┐
                    │   TracerProvider (SDK)           │
                    │   - BatchSpanProcessor (~1s)     │
                    │   - OTLP HTTP exporter            │
                    │   - Resource (service.name, etc.) │
                    └──────────────┬──────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
  ┌──────▼──────┐   ┌──────────────▼──────┐   ┌──────────────▼──────┐
  │ Fiber OTel  │   │  Pipeline Engine    │   │  capability.Invoke  │
  │ middleware  │   │  (custom spans)     │   │  (custom spans)     │
  └──────┬──────┘   └──────────┬──────────┘   └──────────┬──────────┘
         │                     │                         │
  ┌──────▼──────┐   ┌──────────▼──────────┐   ┌──────────────▼──────┐
  │ otelsql     │   │ Watermill handler   │   │ resty + otelhttp    │
  │ (DB traces) │   │ (pub/sub traces)    │   │ (outgoing HTTP)     │
  └─────────────┘   └─────────────────────┘   └─────────────────────┘
```

### Components

| Component              | File                                   | Role                                                         |
| ---------------------- | -------------------------------------- | ------------------------------------------------------------ |
| `TracerProvider`       | `pkg/trace/trace.go`                   | OTLP HTTP exporter init, sampler, lifecycle                  |
| Fiber middleware       | `pkg/trace/fiber.go`                   | HTTP request spans; route filled after `Next()`              |
| Span helpers           | `pkg/trace/helper.go`                  | `StartSpan`, `DetachContext`, `RecordError`, …               |
| Postgres               | `internal/store/postgres/adapter.go`   | `otelsql` spans for queries (`db.system=postgresql`)         |
| Redis hook             | `pkg/rdb/rdb.go`, `pkg/event/redis.go` | Auto-span for Redis commands                                 |
| Pipeline spans         | `pkg/pipeline/engine.go`               | execute / step / `pipeline.cron`                             |
| capability.Invoke span | `pkg/capability/invoke.go`             | Capability invocation span                                   |
| HTTP client            | `pkg/utils/resty.go`                   | `DefaultRestyClient` wraps `otelhttp`                        |
| LLM HTTP               | `pkg/agent/llm/http_client.go`         | `otelhttp` on OpenAI-compatible transport                    |
| Watermill trace        | `pkg/event/action.go`, `pubsub.go`     | Publish inject + consumer extract (W3C)                      |
| Chatagent              | `internal/server/chatagent/`           | scheduled root; Web/API uses `DetachContext`                 |
| Log correlation        | `pkg/flog/flog.go`                     | `Ctx(ctx)` → `trace_id` / `span_id`                          |

## Async continuation (`DetachContext`)

HTTP handlers that return `202` (or Watermill handlers that return before work finishes) must not use bare `context.Background()` — that breaks the Tempo tree.

Pattern:

1. Before returning, `StartSpan(ctx, "….async")` (continuation parent).
2. In the goroutine/pool: `DetachContext(asyncCtx)` then `WithTimeout`.
3. `defer asyncSpan.End()` in the worker.

`DetachContext` is `context.WithoutCancel`: keeps SpanContext, drops cancel. Used by pipeline webhooks (`pipeline.webhook.async`), event-source webhooks (`event_source.webhook.async`), platform chatagent, and Web UI SSE runs.

## Span Naming Convention

| Level            | Span name                              | Location                    |
| ---------------- | -------------------------------------- | --------------------------- |
| HTTP request     | `HTTP {method} {route}`                | `trace/fiber.go`            |
| Webhook async    | `pipeline.webhook.async` / `event_source.webhook.async` | webhook handlers |
| Event publish    | `event.publish {topic}`                | `event/action.go`           |
| Event consume    | `event.receive {topic}`                | `event/pubsub.go`           |
| Pipeline cron    | `pipeline.cron`                        | `pipeline/engine.go`        |
| Pipeline execute | `pipeline.{name}.execute`              | `pipeline/engine.go`        |
| Pipeline step    | `pipeline.{pipeline}.step.{step}`      | `pipeline/engine.go`        |
| Capability       | `capability.{capability}.{operation}`  | `capability/invoke.go`      |
| Agent turn/tool  | `agent.turn` / `agent.tool.*` / `agent.llm.stream` | `pkg/agent/`     |
| Subagent         | `agent.subagent`                       | `pkg/agent/subagent`        |
| Scheduled chat   | `chatagent.scheduled_task`             | `chatagent/scheduler.go`    |
| Database         | `db.Query` / `db.Exec` (otelsql)       | postgres adapter            |
| Redis            | command name                           | redisotel                   |
| Outgoing HTTP    | client span from otelhttp              | resty / LLM transport       |

## Call chains

### Pipeline — event trigger

```
… → event.publish {topic}
      └── event.receive {topic}
            └── pipeline.{name}.execute
                  └── pipeline.{name}.step.{step}
                        └── capability.{cap}.{op}
                              ├── HTTP client (provider; requires SetContext)
                              └── db.* (otelsql)
```

### Pipeline — webhook

```
HTTP POST …
  └── pipeline.webhook.async          ← started before 202; Detach into worker
        └── pipeline.{name}.execute
              └── …
```

### Pipeline — cron (independent root)

```
pipeline.cron                         ← attrs: pipeline.name
  └── pipeline.{name}.execute
        └── …
```

### Chatagent — Web UI

```
HTTP POST /service/web/agents/:id/messages
  └── agent.turn / agent.llm.stream (+ HTTP client) / agent.tool.*
        ├── agent.subagent → agent.turn / tools …
        └── capability.* → db / Redis / provider HTTP
```

### Chatagent — scheduled (independent root)

```
chatagent.scheduled_task              ← attrs: task.id, task.name, session.id
  └── agent.* …
```

## Provider HTTP (`SetContext`)

`DefaultRestyClient` always uses `otelhttp`. Client spans only nest under the caller when the request uses `SetContext(ctx)`. Prefer the `example` / `grafana` / `nocodb` pattern. Requests without context create orphan client roots in Tempo.

## Database noise

With `sample_rate: 1.0`, otelsql emits a span per query. Chatagent session/message traffic makes trees dense — expected for homelab debugging. Prefer lowering `sample_rate` in busy environments rather than disabling DB spans.

## Explicitly out of scope

- Workflow engine full instrumentation
- `platforms.Caller.Do` taking `context.Context`
- Merging confirm POSTs into the in-flight agent run
- Nesting `agent.run` under `agent.turn` (they remain siblings)

## W3C Trace Context Propagation

1. **HTTP**: Fiber extracts; `otelhttp` injects on outbound.
2. **Watermill**: `publishWith` injects the **publish** span context into metadata; `TraceConsumerMiddleware` extracts so `event.receive` is a child of `event.publish`.

## Log Correlation

```go
flog.Ctx(ctx).Info().Msg("processing event")
```

## Configuration

```yaml
tracing:
  enabled: false
  endpoint: "http://localhost:4318/v1/traces"
  service_name: "flowbot"
  environment: "development"
  sample_rate: 1.0
```

## Development

```bash
go test ./pkg/trace/...
go test ./pkg/event/...
go test ./pkg/pipeline/...
go test ./pkg/utils/...
go test ./pkg/agent/subagent/...
```
