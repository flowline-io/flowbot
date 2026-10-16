# `devops` capability operations

DevOps aggregator for beszel, uptimekuma, traefik, grafana, wakapi, dozzle, netalertx, and scanopy

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `beszel_get_system`

Get a Beszel system for the signed-in caller. Read-only.
Example: {"id":"example"}

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

List Beszel systems for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: beszel_list_systems_step
    capability: devops
    operation: beszel_list_systems
```

## `dozzle_health`

Dozzle health and version for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: dozzle_health_step
    capability: devops
    operation: dozzle_health
```

## `grafana_health`

Grafana health for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: grafana_health_step
    capability: devops
    operation: grafana_health
```

## `grafana_list_datasources`

List Grafana datasources for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: grafana_list_datasources_step
    capability: devops
    operation: grafana_list_datasources
```

## `grafana_query`

Query prometheus/alloy/loki/tempo/pyroscope via Grafana for the signed-in caller. Read-only.
Example: {"backend":"example"}

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

Search Grafana dashboards for the signed-in caller. Read-only.
Example: {}

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

Aggregate health of configured devops backends for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: devops
    operation: health
```

## `netalertx_health`

NetAlertX health for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: netalertx_health_step
    capability: devops
    operation: netalertx_health
```

## `netalertx_list_devices`

List NetAlertX devices for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: netalertx_list_devices_step
    capability: devops
    operation: netalertx_list_devices
```

## `netalertx_search_devices`

Search NetAlertX devices for the signed-in caller. Read-only.
Example: {"query":"example"}

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

NetAlertX device totals for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: netalertx_totals_step
    capability: devops
    operation: netalertx_totals
```

## `scanopy_get_host`

Get a Scanopy host for the signed-in caller. Read-only.
Example: {"id":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Host UUID |

**Usage:**

```yaml
  - name: scanopy_get_host_step
    capability: devops
    operation: scanopy_get_host
    params:
      id: "..."  # required
```

## `scanopy_health`

Scanopy health for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: scanopy_health_step
    capability: devops
    operation: scanopy_health
```

## `scanopy_list_daemons`

List Scanopy daemons for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `network_id` | `string` | no | Filter by network UUID |
| `limit` | `number` | no | Page size (default 50, max 1000) |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: scanopy_list_daemons_step
    capability: devops
    operation: scanopy_list_daemons
    params:
      network_id: "..."
      limit: 0
      cursor: "..."
```

## `scanopy_list_hosts`

List Scanopy hosts for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `network_id` | `string` | no | Filter by network UUID |
| `search` | `string` | no | Free-text search |
| `limit` | `number` | no | Page size (default 50, max 1000) |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: scanopy_list_hosts_step
    capability: devops
    operation: scanopy_list_hosts
    params:
      network_id: "..."
      search: "..."
      limit: 0
      cursor: "..."
```

## `scanopy_list_networks`

List Scanopy networks for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `number` | no | Page size (default 50, max 1000) |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: scanopy_list_networks_step
    capability: devops
    operation: scanopy_list_networks
    params:
      limit: 0
      cursor: "..."
```

## `scanopy_list_services`

List Scanopy services for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `network_id` | `string` | no | Filter by network UUID |
| `host_id` | `string` | no | Filter by host UUID |
| `search` | `string` | no | Free-text search |
| `limit` | `number` | no | Page size (default 50, max 1000) |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: scanopy_list_services_step
    capability: devops
    operation: scanopy_list_services
    params:
      network_id: "..."
      host_id: "..."
      search: "..."
      limit: 0
      cursor: "..."
```

## `scanopy_version`

Scanopy API/server version for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: scanopy_version_step
    capability: devops
    operation: scanopy_version
```

## `status`

Configured devops backends for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: status_step
    capability: devops
    operation: status
```

## `traefik_list_routers`

List Traefik routers for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: traefik_list_routers_step
    capability: devops
    operation: traefik_list_routers
```

## `traefik_list_services`

List Traefik services for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: traefik_list_services_step
    capability: devops
    operation: traefik_list_services
```

## `traefik_overview`

Traefik overview for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: traefik_overview_step
    capability: devops
    operation: traefik_overview
```

## `uptimekuma_health`

Uptime Kuma health for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: uptimekuma_health_step
    capability: devops
    operation: uptimekuma_health
```

## `uptimekuma_metrics`

Uptime Kuma metrics summary for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: uptimekuma_metrics_step
    capability: devops
    operation: uptimekuma_metrics
```

## `wakapi_all_time`

Wakapi all-time coding stats for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: wakapi_all_time_step
    capability: devops
    operation: wakapi_all_time
```

## `wakapi_health`

Wakapi health for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: wakapi_health_step
    capability: devops
    operation: wakapi_health
```

## `wakapi_list_projects`

List Wakapi projects for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | no | Project name filter |

**Usage:**

```yaml
  - name: wakapi_list_projects_step
    capability: devops
    operation: wakapi_list_projects
    params:
      query: "..."
```

## `wakapi_summary`

Wakapi activity summary for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `interval` | `string` | no | Range interval (default today) |
| `project` | `string` | no | Project filter |

**Usage:**

```yaml
  - name: wakapi_summary_step
    capability: devops
    operation: wakapi_summary
    params:
      interval: "..."
      project: "..."
```
