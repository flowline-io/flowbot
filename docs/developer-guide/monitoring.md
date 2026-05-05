# Monitoring

Observability stack — Prometheus metrics via PushGateway, OpenTelemetry traces via OTLP, and a pre-built Grafana dashboard for visualization.

## Architecture

```
┌─────────────┐     push (15s)     ┌──────────────┐     scrape       ┌─────────────┐
│   Flowbot   │ ──────────────────▶│ PushGateway  │ ◀─────────────── │  Prometheus │
│  pkg/stats/ │                    │  :9091       │                  │             │
└──────┬──────┘                    └──────────────┘                  └──────┬──────┘
       │                                                                    │
       │ OTLP HTTP (/v1/traces)                                             │ datasource
       │                                                                    │
       ▼                                                                    ▼
┌──────────────┐                                                   ┌─────────────┐
│  Tempo/Jaeger│ ◀─────────────────────────────────────────────── │   Grafana   │
│  :4318       │                                                   │   :3000     │
└──────────────┘                                                   └─────────────┘
       ▲                                                                  │
       │ OTLP traces                                                      │
       │                                                                  │
┌──────┴──────┐                              ┌──────────────┐             │
│   Flowbot   │                              │  Meilisearch  │            │
│  pkg/trace/ │                              │  Prometheus   │────────────┘
│  Fiber OTel │                              │  /metrics     │  (optional)
│  GORM OTel  │                              └──────────────┘
│  Redis OTel │
└─────────────┘
```

Two data paths feed into Grafana:

| Path | Protocol | Exporter | Default Port |
| ---- | -------- | -------- | ------------ |
| Metrics | PushGateway → Prometheus scrape | `pkg/stats/` push every 15s | `:9091` |
| Traces | OTLP HTTP (protobuf) | `pkg/trace/` batch export | `:4318` |

## Prerequisites

Start the observability services before configuring Flowbot:

```bash
# PushGateway — metrics relay
docker run -d --name pushgateway \
  -p 9091:9091 \
  prom/pushgateway:latest

# Tempo — trace storage (all-in-one for development)
docker run -d --name tempo \
  -p 4318:4318 \
  -p 3200:3200 \
  grafana/tempo:latest

# Grafana — dashboards & visualization
docker run -d --name grafana \
  -p 3000:3000 \
  -e "GF_AUTH_ANONYMOUS_ENABLED=true" \
  grafana/grafana:latest
```

For production, add Prometheus:

```bash
docker run -d --name prometheus \
  -p 9090:9090 \
  -v ./prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus:latest
```

## Flowbot Configuration

Enable both metrics push and trace export in `flowbot.yaml`:

```yaml
# Metrics — pushed to PushGateway every 15s
metrics:
  enabled: true
  endpoint: "http://localhost:9091"

# Tracing — OTLP HTTP batch export
tracing:
  enabled: true
  endpoint: "http://localhost:4318/v1/traces"
  service_name: "flowbot"
  environment: "production"
  sample_rate: 1.0
```

| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `metrics.enabled` | bool | `false` | Enable PushGateway push |
| `metrics.endpoint` | string | `http://localhost:9091` | PushGateway base URL |
| `tracing.enabled` | bool | `false` | Enable OTLP trace export |
| `tracing.endpoint` | string | `http://localhost:4318/v1/traces` | OTLP HTTP collector |
| `tracing.service_name` | string | `flowbot` | `service.name` resource attribute |
| `tracing.environment` | string | `development` | `deployment.environment` attribute |
| `tracing.sample_rate` | float | `1.0` | 1.0 = all, 0.1 = 10% |

## Prometheus Configuration

Point Prometheus at the PushGateway:

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: pushgateway
    honor_labels: true
    static_configs:
      - targets: ["pushgateway:9091"]

  # Optional: scrape Flowbot's internal /metrics endpoint for Go runtime metrics
  - job_name: flowbot
    static_configs:
      - targets: ["flowbot:8888"]
```

After restarting Prometheus, verify metrics are flowing:

```bash
# Check PushGateway has flowbot metrics
curl -s http://localhost:9091/metrics | grep "job=\"flowbot\""

# Check Prometheus can see them
curl -s "http://localhost:9090/api/v1/query?query=module_total_gauge" | jq .
```

## Grafana Setup

### 1. Add datasources

In Grafana (http://localhost:3000), go to **Connections → Data sources**:

**Prometheus:**
- Name: `Prometheus`
- URL: `http://prometheus:9090`
- Click **Save & test**

**Tempo:**
- Name: `Tempo`
- URL: `http://tempo:3200`
- Click **Save & test**

### 2. Import the dashboard

**Dashboards → New → Import**, paste the contents of [`../grafana-dashboard.json`](../grafana-dashboard.json).

Or import programmatically:

```bash
# Via Grafana API
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d "{\"dashboard\": $(cat docs/grafana-dashboard.json), \"overwrite\": true}"
```

### 3. Select datasources

After import, use the dropdowns at the top of the dashboard to select your Prometheus and Tempo datasources.

## Dashboard Reference

The dashboard is organized into 5 rows.

### Overview (top row)

| Panel | Query | Type |
| ----- | ----- | ---- |
| Active Modules | `module_total_gauge` | Stat |
| Docker Containers | `docker_container_total_gauge` | Stat |
| Monitors DOWN | `monitor_down_total_gauge` | Stat |
| Monitors UP | `monitor_up_total_gauge` | Stat |
| Module Runs by Ruleset | `rate(module_run_total_counter[5m])` | Time series |
| Event Processing Rate | `rate(event_total_counter[5m])` | Time series |

### Features

| Panel | Query | Type |
| ----- | ----- | ---- |
| Bookmarks | `bookmark_total_gauge` | Stat + Trend |
| Torrent Downloads | `torrent_download_total_gauge` | Stat |
| Torrents by Status | `torrent_status_total_gauge` | Time series |
| RSS Unread | `reader_unread_total_gauge` | Stat |
| RSS (total vs unread) | `reader_total_gauge`, `reader_unread_total_gauge` | Time series |
| Kanban Tasks | `kanban_task_total_gauge` | Stat + Trend |
| Kanban Events | `rate(kanban_event_total_counter[5m])` | Time series |
| Gitea Open Issues | `gitea_issue_total_gauge{status="open"}` | Stat |

### Search

| Panel | Query | Type |
| ----- | ----- | ---- |
| Search Query Rate | `rate(search_total_counter[5m])` by `index` | Time series |
| Document Indexing Rate | `rate(search_processed_document_total_counter[5m])` by `index` | Time series |

### Infrastructure

| Panel | Query | Type |
| ----- | ----- | ---- |
| Docker Containers | `docker_container_total_gauge` | Time series |
| Uptime Monitors | `monitor_up_total_gauge`, `monitor_down_total_gauge` | Time series |

### Traces (Tempo)

| Panel | Query | Type |
| ----- | ----- | ---- |
| HTTP Request Traces | `serviceName=flowbot spanName=HTTP` | Table |
| Pipeline Execution | `serviceName=flowbot spanName=pipeline` | Table |
| Ability Invocation | `serviceName=flowbot spanName=ability` | Table |
| Event Processing | `serviceName=flowbot spanName=event` | Table |
| Recent Pipelines | Trace search | Trace view |
| Recent Events | Trace search | Trace view |

## Metrics Reference

All 21 custom metrics, each producing a `_counter` and `_gauge` suffix variant:

| Base Name | Labels | Updated By | Type |
| --------- | ------ | ---------- | ---- |
| `module_total` | — | `internal/server/module.go` | Gauge |
| `module_run_total` | `ruleset` | `internal/server/router.go`, `func.go` | Counter |
| `event_total` | — | `pkg/event/pubsub.go` | Counter |
| `bookmark_total` | — | `internal/modules/bookmark/cron.go` | Gauge |
| `search_total` | `index` | `pkg/search/search.go` | Counter |
| `search_processed_document_total` | `index` | `pkg/search/search.go` | Counter |
| `torrent_download_total` | — | `internal/modules/torrent/cron.go` | Gauge |
| `torrent_status_total` | `status` | `internal/modules/torrent/cron.go` | Gauge |
| `gitea_issue_total` | `status` | `internal/modules/gitea/cron.go` | Gauge |
| `kanban_event_total` | `event_name` | `internal/modules/kanban/webhook.go` | Counter |
| `kanban_task_total` | — | `internal/modules/kanban/cron.go` | Gauge |
| `reader_total` | — | `internal/modules/reader/cron.go` | Gauge |
| `reader_unread_total` | — | `internal/modules/reader/cron.go` | Gauge |
| `monitor_up_total` | — | `internal/modules/server/cron.go` | Gauge |
| `monitor_down_total` | — | `internal/modules/server/cron.go` | Gauge |
| `docker_container_total` | — | `internal/modules/server/cron.go` | Gauge |

**PushGateway labels:** `job` (default `flowbot`), `instance` (hostid), `hostname`.

**Ruleset label values:** `input`, `agent`, `command`, `cron`, `form`.

### Query patterns

Since each metric exists as both Counter and Gauge, choose the right suffix:

```promql
# Current value — use _gauge
module_total_gauge{job="flowbot"}
bookmark_total_gauge{job="flowbot"}

# Rate of change — use rate() on _counter
rate(event_total_counter{job="flowbot"}[5m])
rate(module_run_total_counter{job="flowbot"}[5m])
```

### Redis-backed metrics API

A subset of metrics is also available as JSON via the internal API:

```
GET /user/metrics
```

```json
{
  "bot_total": 16,
  "bookmark_total": 42,
  "torrent_download_total": 3,
  "gitea_issue_total": 7,
  "reader_unread_total": 15,
  "kanban_task_total": 8,
  "monitor_up_total": 12,
  "monitor_down_total": 0,
  "docker_container_total": 20
}
```

## Alerting

Example Prometheus alert rules for common failure conditions:

```yaml
# flowbot-alerts.yml
groups:
  - name: flowbot
    rules:
      - alert: FlowbotDown
        expr: absent(module_total_gauge{job="flowbot"})
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Flowbot instance {{ $labels.instance }} is down"
          description: "No metrics pushed for 2 minutes."

      - alert: MonitorDown
        expr: monitor_down_total_gauge{job="flowbot"} > 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "{{ $value }} UptimeKuma monitor(s) are DOWN"

      - alert: HighEventRate
        expr: rate(event_total_counter{job="flowbot"}[5m]) > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Event rate above 100/s sustained for 5 minutes"

      - alert: NoDockerContainers
        expr: docker_container_total_gauge{job="flowbot"} == 0
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "No Docker containers detected by homelab scanner"
```

Load alert rules into Prometheus:

```yaml
# prometheus.yml
rule_files:
  - "flowbot-alerts.yml"
```

## Known Gaps

1. **No HTTP RED metrics.** Fiber does not emit request duration histograms. The `/metrics` scrape endpoint serves only Go runtime defaults — no application-level HTTP metrics. To get request latency and error rate, enable Fiber's built-in metrics middleware or add a custom histogram in `pkg/stats/`.

2. **Push-only metrics.** All custom metrics go through PushGateway. The `/metrics` endpoint (`prometheus.DefaultGatherer`) is empty of app metrics. If you need pull-based scraping, register metrics to both the custom registry and the default registry in `pkg/stats/stats.go`.

3. **Queue metrics are dead code.** `queue_processed_tasks_total`, `queue_failed_tasks_total`, `queue_in_progress_tasks` are defined but never called. They exist for future async task tracking.

4. **No OTel metrics (meters).** The `go.opentelemetry.io/otel/metric` package is available but unused. No custom counters, histograms, or gauges are defined in the OTel SDK.

5. **Cache metrics missing.** The Ristretto cache in `pkg/cache/` has no hit/miss/size instrumentation.

## Troubleshooting

### No metrics in Grafana

```bash
# 1. Check PushGateway has data
curl -s http://localhost:9091/metrics | grep flowbot

# 2. Check Prometheus targets
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.job=="pushgateway")'

# 3. Check Flowbot logs for push errors
# Look for: "Failed to push metrics"
```

### No traces in Grafana

```bash
# 1. Check Tempo is receiving
curl -s http://localhost:3200/ready

# 2. Verify Flowbot tracing config
# Set tracing.enabled: true in flowbot.yaml and restart

# 3. Check for trace export errors in logs
# Look for: "Failed to export spans"
```

### PushGateway shows stale metrics

PushGateway retains the last pushed value indefinitely. If Flowbot stops, the last value persists. Use `push_time_seconds` to detect staleness:

```promql
# Alert if metrics are older than 60s
(push_time_seconds{job="flowbot"} - time()) > 60
```
