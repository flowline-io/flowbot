# `devops` capability operations

DevOps aggregator for beszel, uptimekuma, traefik, grafana, wakapi, and dozzle

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `beszel_get_system`

Get a Beszel system

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | System ID |

**Usage:**

```yaml
  - name: beszel_get_system_step
    capability: devops
    operation: beszel_get_system
    params:
      id: "..."  # required
```

## `beszel_list_systems`

List Beszel systems

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: beszel_list_systems_step
    capability: devops
    operation: beszel_list_systems
```

## `dozzle_health`

Dozzle health and version

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: dozzle_health_step
    capability: devops
    operation: dozzle_health
```

## `grafana_health`

Grafana health

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: grafana_health_step
    capability: devops
    operation: grafana_health
```

## `grafana_list_datasources`

List Grafana datasources

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: grafana_list_datasources_step
    capability: devops
    operation: grafana_list_datasources
```

## `grafana_query`

Query prometheus/alloy/loki/tempo/pyroscope via Grafana

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `backend` | `string` | yes | prometheus|alloy|loki|tempo|pyroscope |
| `expr` | `string` | yes | Query expression |
| `datasource_uid` | `string` | no | Optional datasource UID |
| `from` | `string` | no | Grafana from time (default now-1h) |
| `to` | `string` | no | Grafana to time (default now) |
| `max_lines` | `number` | no | Loki max lines |

**Usage:**

```yaml
  - name: grafana_query_step
    capability: devops
    operation: grafana_query
    params:
      backend: "..."  # required
      expr: "..."  # required
      datasource_uid: "..."
      from: "..."
      to: "..."
      max_lines: 0
```

## `grafana_search_dashboards`

Search Grafana dashboards

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | no | Search query |

**Usage:**

```yaml
  - name: grafana_search_dashboards_step
    capability: devops
    operation: grafana_search_dashboards
    params:
      query: "..."
```

## `health`

Aggregate health of configured devops backends

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: devops
    operation: health
```

## `netalertx_health`

NetAlertX health

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: netalertx_health_step
    capability: devops
    operation: netalertx_health
```

## `netalertx_list_devices`

List NetAlertX devices

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: netalertx_list_devices_step
    capability: devops
    operation: netalertx_list_devices
```

## `netalertx_search_devices`

Search NetAlertX devices

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | yes | Search query |

**Usage:**

```yaml
  - name: netalertx_search_devices_step
    capability: devops
    operation: netalertx_search_devices
    params:
      query: "..."  # required
```

## `netalertx_totals`

NetAlertX device totals

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: netalertx_totals_step
    capability: devops
    operation: netalertx_totals
```

## `status`

Configured devops backends

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: status_step
    capability: devops
    operation: status
```

## `traefik_list_routers`

List Traefik routers

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: traefik_list_routers_step
    capability: devops
    operation: traefik_list_routers
```

## `traefik_list_services`

List Traefik services

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: traefik_list_services_step
    capability: devops
    operation: traefik_list_services
```

## `traefik_overview`

Traefik overview

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: traefik_overview_step
    capability: devops
    operation: traefik_overview
```

## `uptimekuma_health`

Uptime Kuma health

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: uptimekuma_health_step
    capability: devops
    operation: uptimekuma_health
```

## `uptimekuma_metrics`

Uptime Kuma metrics summary

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: uptimekuma_metrics_step
    capability: devops
    operation: uptimekuma_metrics
```

## `wakapi_list_projects`

List Wakapi projects

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: wakapi_list_projects_step
    capability: devops
    operation: wakapi_list_projects
```

## `wakapi_summary`

Wakapi activity summary

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `interval` | `string` | no | Interval (default today) |

**Usage:**

```yaml
  - name: wakapi_summary_step
    capability: devops
    operation: wakapi_summary
    params:
      interval: "..."
```
