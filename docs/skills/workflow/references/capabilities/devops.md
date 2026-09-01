# `devops` capability actions

DevOps aggregator for beszel, uptimekuma, traefik, grafana, wakapi, dozzle, netalertx, and scanopy

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:devops.beszel_get_system`

Get a Beszel system

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | System ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: beszel_get_system_step
    action: capability:devops.beszel_get_system
    params:
      id: "..."  # required
```

## `capability:devops.beszel_list_systems`

List Beszel systems

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: beszel_list_systems_step
    action: capability:devops.beszel_list_systems
```

## `capability:devops.dozzle_health`

Dozzle health and version

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: dozzle_health_step
    action: capability:devops.dozzle_health
```

## `capability:devops.grafana_health`

Grafana health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: grafana_health_step
    action: capability:devops.grafana_health
```

## `capability:devops.grafana_list_datasources`

List Grafana datasources

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: grafana_list_datasources_step
    action: capability:devops.grafana_list_datasources
```

## `capability:devops.grafana_query`

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

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: grafana_query_step
    action: capability:devops.grafana_query
    params:
      backend: "..."  # required
      expr: "..."  # required
      datasource_uid: "..."
      from: "..."
      to: "..."
      max_lines: 0
```

## `capability:devops.grafana_search_dashboards`

Search Grafana dashboards

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | no | Search query |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: grafana_search_dashboards_step
    action: capability:devops.grafana_search_dashboards
    params:
      query: "..."
```

## `capability:devops.health`

Aggregate health of configured devops backends

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:devops.health
```

## `capability:devops.netalertx_health`

NetAlertX health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: netalertx_health_step
    action: capability:devops.netalertx_health
```

## `capability:devops.netalertx_list_devices`

List NetAlertX devices

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: netalertx_list_devices_step
    action: capability:devops.netalertx_list_devices
```

## `capability:devops.netalertx_search_devices`

Search NetAlertX devices

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | yes | Search query |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: netalertx_search_devices_step
    action: capability:devops.netalertx_search_devices
    params:
      query: "..."  # required
```

## `capability:devops.netalertx_totals`

NetAlertX device totals

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: netalertx_totals_step
    action: capability:devops.netalertx_totals
```

## `capability:devops.scanopy_get_host`

Get a Scanopy host

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Host UUID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: scanopy_get_host_step
    action: capability:devops.scanopy_get_host
    params:
      id: "..."  # required
```

## `capability:devops.scanopy_health`

Scanopy health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: scanopy_health_step
    action: capability:devops.scanopy_health
```

## `capability:devops.scanopy_list_daemons`

List Scanopy daemons

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `network_id` | `string` | no | Filter by network UUID |
| `limit` | `number` | no | Page size (default 50, max 1000) |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: scanopy_list_daemons_step
    action: capability:devops.scanopy_list_daemons
    params:
      network_id: "..."
      limit: 0
      cursor: "..."
```

## `capability:devops.scanopy_list_hosts`

List Scanopy hosts

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `network_id` | `string` | no | Filter by network UUID |
| `search` | `string` | no | Free-text search |
| `limit` | `number` | no | Page size (default 50, max 1000) |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: scanopy_list_hosts_step
    action: capability:devops.scanopy_list_hosts
    params:
      network_id: "..."
      search: "..."
      limit: 0
      cursor: "..."
```

## `capability:devops.scanopy_list_networks`

List Scanopy networks

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `number` | no | Page size (default 50, max 1000) |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: scanopy_list_networks_step
    action: capability:devops.scanopy_list_networks
    params:
      limit: 0
      cursor: "..."
```

## `capability:devops.scanopy_list_services`

List Scanopy services

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `network_id` | `string` | no | Filter by network UUID |
| `host_id` | `string` | no | Filter by host UUID |
| `search` | `string` | no | Free-text search |
| `limit` | `number` | no | Page size (default 50, max 1000) |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: scanopy_list_services_step
    action: capability:devops.scanopy_list_services
    params:
      network_id: "..."
      host_id: "..."
      search: "..."
      limit: 0
      cursor: "..."
```

## `capability:devops.scanopy_version`

Scanopy API/server version

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: scanopy_version_step
    action: capability:devops.scanopy_version
```

## `capability:devops.status`

Configured devops backends

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: status_step
    action: capability:devops.status
```

## `capability:devops.traefik_list_routers`

List Traefik routers

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: traefik_list_routers_step
    action: capability:devops.traefik_list_routers
```

## `capability:devops.traefik_list_services`

List Traefik services

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: traefik_list_services_step
    action: capability:devops.traefik_list_services
```

## `capability:devops.traefik_overview`

Traefik overview

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: traefik_overview_step
    action: capability:devops.traefik_overview
```

## `capability:devops.uptimekuma_health`

Uptime Kuma health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: uptimekuma_health_step
    action: capability:devops.uptimekuma_health
```

## `capability:devops.uptimekuma_metrics`

Uptime Kuma metrics summary

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: uptimekuma_metrics_step
    action: capability:devops.uptimekuma_metrics
```

## `capability:devops.wakapi_all_time`

Wakapi all-time coding stats

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: wakapi_all_time_step
    action: capability:devops.wakapi_all_time
```

## `capability:devops.wakapi_health`

Wakapi health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: wakapi_health_step
    action: capability:devops.wakapi_health
```

## `capability:devops.wakapi_list_projects`

List Wakapi projects

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | no | Project name filter |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: wakapi_list_projects_step
    action: capability:devops.wakapi_list_projects
    params:
      query: "..."
```

## `capability:devops.wakapi_summary`

Wakapi activity summary

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `interval` | `string` | no | Range interval (default today) |
| `project` | `string` | no | Project filter |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: wakapi_summary_step
    action: capability:devops.wakapi_summary
    params:
      interval: "..."
      project: "..."
```
