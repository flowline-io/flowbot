# Homelab App Discovery

Automatic identification of API endpoints and authentication mechanisms for self-hosted
apps registered through the Homelab Scanner. The system uses two complementary strategies:
**label-based static discovery** (authoritative) and **runtime probe-based detection**
(optional best-effort).

## Overview

The discovery subsystem enriches scanned homelab apps with capability, endpoint, and
authentication metadata, enabling the Hub to present discovered capabilities and the
health checker to probe API endpoints directly.

```
docker-compose labels     runtime HTTP probes
        \                       /
         \                     /
        App Scanning (scanner.go)
                  |
          ParseLabels() + ProbeEngine
                  |
        App.Capabilities[] field
                  |
        hub.AutoBind() / LogDiscovered()
                  |
        hub.EndpointHealthChecker
```

Flowbot is assumed to run on the **host**. Probes only target **published** TCP ports
(`Host` / `localhost` + host port). Labels must use URLs that the Flowbot process can
reach. One compose file binds **one** capability.

## Strategy 1: Label-Based Discovery (Static)

Define the capability, endpoint, and auth mechanism directly in your
`docker-compose.yaml` using the `flowbot.*` label convention. Labels are parsed
during filesystem scanning with zero runtime cost and take precedence over probes
when `label_priority` is true.

### Label Convention

| Label                         | Required | Description                                               | Example                         |
| ----------------------------- | -------- | --------------------------------------------------------- | ------------------------------- |
| `flowbot.capability`          | Yes      | Capability type the app provides                          | `karakeep`                      |
| `flowbot.backend`             | No       | Deprecated; ignored                                       | —                               |
| `flowbot.endpoint.base`       | No       | Base URL reachable from the Flowbot process               | `http://127.0.0.1:3000/api/v1`  |
| `flowbot.endpoint.health`     | No       | Health check path (relative to base)                      | `/health`                       |
| `flowbot.endpoint.health_ttl` | No       | Health check cache TTL (duration string)                  | `30s`                           |
| `flowbot.auth.type`           | No       | Auth type: `api_token`, `basic`, `oauth2`, `oidc`, `none` | `api_token`                     |
| `flowbot.auth.header`         | No       | HTTP header name for auth                                 | `Authorization`                 |
| `flowbot.auth.prefix`         | No       | Header value prefix                                       | `Bearer`                        |
| `flowbot.auth.token_key`      | No       | Environment variable name holding the token               | `LW_API_KEY`                    |
| `flowbot.auth.token_source`   | No       | Where to read the token: `env`, `file`, `config`          | `env`                           |

`flowbot.endpoint.base` must be reachable by the Flowbot process (typically
`http://127.0.0.1:<published-port>/...`). Do not use container DNS names unless you
know that name resolves from where Flowbot runs.

### Supported Capabilities

| Label Value     | Capability Type | Typical Backend |
| --------------- | --------------- | --------------- |
| `karakeep`      | Bookmark        | karakeep        |
| `archive`       | Archive         | archivebox      |
| `miniflux`      | Reader (RSS)    | miniflux        |
| `kanboard`      | Kanban          | kanboard        |
| `finance`       | Finance         | fireflyiii      |
| `infra`         | Infrastructure  | —               |
| `shell_history` | Shell History   | atuin           |

Legacy domain labels (`bookmark`, `reader`, `kanban`, `note`, `memo`, `forge`) map to
the canonical provider IDs above and log a deprecation warning.

### Example

```yaml
# docker-compose.yaml
services:
  linkwarden:
    image: ghcr.io/linkwarden/linkwarden:latest
    container_name: linkwarden
    ports:
      - "3000:3000"
    labels:
      flowbot.capability: "karakeep"
      flowbot.endpoint.base: "http://127.0.0.1:3000/api/v1"
      flowbot.endpoint.health: "/health"
      flowbot.endpoint.health_ttl: "30s"
      flowbot.auth.type: "api_token"
      flowbot.auth.header: "Authorization"
      flowbot.auth.prefix: "Bearer"
      flowbot.auth.token_key: "LW_API_KEY"
      flowbot.auth.token_source: "env"
```

With this label configuration, the scanner automatically:

1. Registers `karakeep` as the discovered capability for this app.
2. Records the base URL, health path, and auth metadata.
3. The hub logs the discovery and exposes it through `/hub/capabilities` responses.
4. The health checker probes `http://127.0.0.1:3000/api/v1/health` on each check cycle.

## Strategy 2: Runtime Probe Engine (Dynamic)

When labels are absent or incomplete, an optional HTTP probe engine can discover
endpoints and auth mechanisms by making requests to **host-published** ports on
running containers. Probes are best-effort; they do not replace labels.

### What It Detects

| Discovery Target          | Method                                                                                          |
| ------------------------- | ----------------------------------------------------------------------------------------------- |
| API endpoint reachability | HTTP GET to each published TCP port (http and https)                                            |
| Health endpoint           | Probes common paths: `/health`, `/healthz`, `/api/health`, `/api/v1/health`, `/ping`, `/status` |
| Authentication mechanism  | Analyses HTTP 401/403 responses and `WWW-Authenticate` headers                                  |
| OIDC support              | Probes `/.well-known/openid-configuration`                                                      |
| Service fingerprint       | Matches known API paths against the fingerprint database                                        |

### Auth Detection Logic

| Response                                   | Detected Auth |
| ------------------------------------------ | ------------- |
| 401 with `WWW-Authenticate: Bearer`        | `oauth2`      |
| 401 with `WWW-Authenticate: Basic`         | `basic`       |
| 401/403 with no `WWW-Authenticate`         | `api_token`   |
| 200 at `/.well-known/openid-configuration` | `oidc`        |
| 200 at base URL                            | `none`        |
| Other                                      | `unknown`     |

### Fingerprint Database

The probe engine matches known self-hosted services by **path reachability only**.
When a detection path responds successfully, the capability type is inferred.

| Service     | Capability | Detection Path    |
| ----------- | ---------- | ----------------- |
| LinkWarden  | karakeep   | `/api/v1/health`  |
| ArchiveBox  | archive    | `/admin`          |
| Miniflux    | miniflux   | `/v1/healthcheck` |
| Kanboard    | kanboard   | `/jsonrpc.php`    |
| Firefly III | finance    | `/api/v1/about`   |

### Probe Behavior

- Probes only run when `homelab.discovery.probe_enabled` is `true`.
- Non-running apps are skipped (probes require an active container).
- Only TCP ports with a **published host port** are probed; unpublished container ports are skipped.
- Both HTTP and HTTPS are attempted for each published port.
- A configurable concurrency limit controls simultaneous probes.
- Each probe respects a configurable per-request timeout.
- Probes do not follow redirects (to avoid false positives on login pages).
- All probe requests include `User-Agent: Flowbot-Homelab-Probe/1.0`.

## Configuration

Add the `discovery` block under `homelab` in your `flowbot.yaml`:

```yaml
homelab:
  root: "/home/user/homelab"
  apps_dir: "/home/user/homelab/apps"
  compose_file: "docker-compose.yaml"
  runtime:
    mode: "docker_socket"
    docker_socket: "unix:///var/run/docker.sock"
  allowlist: []
  permissions:
    status: true
    logs: true
    start: false
    stop: false
    restart: false
    pull: false
    update: false
    exec: false
  discovery:
    probe_enabled: true
    probe_timeout: "5s"
    probe_concurrency: 4
    fingerprint_enabled: true
    label_priority: true
```

### Discovery Configuration Reference

| Field                 | Type     | Default | Description                                    |
| --------------------- | -------- | ------- | ---------------------------------------------- |
| `probe_enabled`       | bool     | `false` | Enable runtime HTTP probing of published ports |
| `probe_timeout`       | duration | `"5s"`  | Per-request timeout for probe HTTP calls       |
| `probe_concurrency`   | int      | `4`     | Maximum number of parallel probe goroutines    |
| `fingerprint_enabled` | bool     | `true`  | Enable service fingerprint matching by path    |
| `label_priority`      | bool     | `true`  | When true, label-derived data takes precedence |

## How Discovery Feeds Into the Hub

### Scanning Phase

1. `Scanner.Scan()` reads `apps/*/docker-compose.yaml` files.
2. `ParseLabels()` extracts at most one `AppCapability` from labels.
3. If discovery probes are enabled, `ProbeEngine.ProbeAll()` enriches apps with runtime data.
4. Results are merged into `App.Capabilities[]` and persisted to PostgreSQL via `hub_store`.

### Capability Awareness

- `hub.AutoBind()` reads homelab registry apps and returns a list of discovered bindings
  with their registration status in the hub.
- `hub.LogDiscovered()` logs discovered capabilities that are not yet configured as
  full hub descriptors.
- The `/hub/capabilities` API endpoint exposes binding information.
- The homelab app listing at `/hub/apps` includes `capabilities` in each app response.

### Health Check Integration

- `EndpointHealthChecker` probes discovered health URLs on each health check cycle.
- Results are included in the `/hub/health` response alongside registered hub descriptors.
- Capabilities already registered in the hub are excluded to avoid duplicate entries.
- The overall health status degrades if any endpoint health check fails.

## Design Decisions

Rationale for the host-side discovery contracts:
[.agents/notes/implemented/simplification/2026-09-11-homelab-discovery-host-contracts.md](../../.agents/notes/implemented/simplification/2026-09-11-homelab-discovery-host-contracts.md).

| Contract | Behaviour |
| -------- | --------- |
| Host-published ports only | Probes use `Host` or `localhost` plus the published host port; unpublished ports are skipped |
| Path-only fingerprints | Matching uses reachable detection paths only |
| One compose = one capability | `ParseLabels` accepts a single `flowbot.capability` per compose file |
| Reachable `endpoint.base` | Label URLs must be reachable from the Flowbot process |
| Labels beat probes | When `label_priority` is true, label data wins over probe enrichment |
| Flat `flowbot.*` labels | Compose label convention for static discovery |
| Probe engine nil when disabled | Callers check for nil instead of an empty engine |
| Homelab cap strings, not `hub.CapabilityType` | Avoids an import cycle between `homelab` and `hub` |
| TCP only; no redirects | UDP skipped; redirects not followed |
| Auth header names via labels | Probes may infer `api_token` from 401/403 without `WWW-Authenticate`, but not the exact header name |

## Adding New Service Fingerprints

To add fingerprint support for a new service, edit
`pkg/homelab/probe/fingerprints.go`:

```go
{
    Capability: "your-capability",
    Provider:   "your-provider",
    Paths:      []string{"/api/v1/health"},
}
```

Each reachable path contributes to the confidence score. When the cumulative score
exceeds zero, the engine emits a match.

## Testing

```bash
# Label parsing tests
go test -run TestParseLabels ./pkg/homelab/

# Auth detector tests
go test ./pkg/homelab/probe/ -run TestAuthDetector

# Auto-binding tests
go test ./pkg/hub/ -run TestAutoBind

# Full homelab suite
go test ./pkg/homelab/...
```
