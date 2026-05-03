# Homelab App Discovery

Automatic identification of API endpoints and authentication mechanisms for self-hosted
apps registered through the Homelab Scanner. The system uses two complementary strategies:
**label-based static discovery** and **runtime probe-based detection**.

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

## Strategy 1: Label-Based Discovery (Static)

Define capabilities, endpoints, and auth mechanisms directly in your
`docker-compose.yaml` using the `flowbot.*` label convention. Labels are parsed
during filesystem scanning with zero runtime cost.

### Label Convention

| Label                              | Required | Description                                  | Example                      |
| ---------------------------------- | -------- | -------------------------------------------- | ---------------------------- |
| `flowbot.capability`               | Yes      | Capability type the app provides             | `bookmark`                   |
| `flowbot.backend`                  | No       | Backend provider name (defaults to capability)| `karakeep`                   |
| `flowbot.endpoint.base`            | No       | Base URL for the API                         | `http://linkwarden:3000/api/v1`|
| `flowbot.endpoint.health`          | No       | Health check path (relative to base)         | `/health`                    |
| `flowbot.endpoint.health_ttl`      | No       | Health check cache TTL (duration string)     | `30s`                        |
| `flowbot.auth.type`                | No       | Auth type: `api_token`, `basic`, `oauth2`, `oidc`, `none` | `api_token` |
| `flowbot.auth.header`              | No       | HTTP header name for auth                    | `Authorization`              |
| `flowbot.auth.prefix`              | No       | Header value prefix                          | `Bearer`                     |
| `flowbot.auth.token_key`           | No       | Environment variable name holding the token  | `LW_API_KEY`                 |
| `flowbot.auth.token_source`        | No       | Where to read the token: `env`, `file`, `config` | `env`                    |

### Supported Capabilities

| Label Value      | Capability Type | Typical Backend |
| ---------------- | --------------- | --------------- |
| `bookmark`       | Bookmark        | karakeep        |
| `archive`        | Archive         | archivebox      |
| `reader`         | Reader (RSS)    | miniflux        |
| `kanban`         | Kanban          | kanboard        |
| `finance`        | Finance         | fireflyiii      |
| `infra`          | Infrastructure  | —               |
| `shell_history`  | Shell History   | atuin           |

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
      flowbot.capability: "bookmark"
      flowbot.backend: "karakeep"
      flowbot.endpoint.base: "http://linkwarden:3000/api/v1"
      flowbot.endpoint.health: "/health"
      flowbot.endpoint.health_ttl: "30s"
      flowbot.auth.type: "api_token"
      flowbot.auth.header: "Authorization"
      flowbot.auth.prefix: "Bearer"
      flowbot.auth.token_key: "LW_API_KEY"
      flowbot.auth.token_source: "env"
```

With this label configuration, the scanner automatically:

1. Registers `bookmark` as a discovered capability for this app.
2. Records the base URL, health path, and auth metadata.
3. The hub logs the discovery and exposes it through `/hub/capabilities` responses.
4. The health checker probes `http://linkwarden:3000/api/v1/health` on each check cycle.

## Strategy 2: Runtime Probe Engine (Dynamic)

When labels are absent or incomplete, an optional HTTP probe engine can discover
endpoints and auth mechanisms by making requests to running containers.

### What It Detects

| Discovery Target             | Method                                                          |
| ---------------------------- | --------------------------------------------------------------- |
| API endpoint reachability    | HTTP GET to each published TCP port (http and https)            |
| Health endpoint              | Probes common paths: `/health`, `/healthz`, `/api/health`, `/api/v1/health`, `/ping`, `/status` |
| Authentication mechanism     | Analyses HTTP 401/403 responses and `WWW-Authenticate` headers  |
| OIDC support                 | Probes `/.well-known/openid-configuration`                      |
| Service fingerprint          | Matches known API paths against the fingerprint database        |

### Auth Detection Logic

| Response                           | Detected Auth   |
| ---------------------------------- | --------------- |
| 401 with `WWW-Authenticate: Bearer` | `oauth2`        |
| 401 with `WWW-Authenticate: Basic`  | `basic`         |
| 401/403 with no `WWW-Authenticate`  | `api_token`     |
| 200 at `/.well-known/openid-configuration` | `oidc`   |
| 200 at base URL                    | `none`          |
| Other                              | `unknown`       |

### Fingerprint Database

The probe engine maintains a fingerprint database of known self-hosted services.
When a service's API paths match a fingerprint, the capability type is automatically
inferred. Currently supported fingerprints:

| Service     | Capability | Detection Path      |
| ----------- | ---------- | ------------------- |
| LinkWarden  | bookmark   | `/api/v1/health`    |
| ArchiveBox  | archive    | `/admin`            |
| Miniflux    | reader     | `/v1/healthcheck`   |
| Kanboard    | kanban     | `/jsonrpc.php`      |
| Firefly III | finance    | `/api/v1/about`     |

### Probe Behavior

- Probes only run when `homelab.discovery.probe_enabled` is `true`.
- Non-running apps are skipped (probes require an active container).
- Both HTTP and HTTPS are attempted for each TCP port.
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
    probe_port_strategy: "published"
    fingerprint_enabled: true
    label_priority: true
```

### Discovery Configuration Reference

| Field                  | Type     | Default      | Description                                                   |
| ---------------------- | -------- | ------------ | ------------------------------------------------------------- |
| `probe_enabled`        | bool     | `false`      | Enable runtime HTTP probing of running containers             |
| `probe_timeout`        | duration | `"5s"`       | Per-request timeout for probe HTTP calls                      |
| `probe_concurrency`    | int      | `4`          | Maximum number of parallel probe goroutines                   |
| `probe_networks`       | []string | `[]`         | Docker network names to resolve (reserved for future use)     |
| `probe_port_strategy`  | string   | `"published"`| Port resolution strategy: `published`, `container`, or `both` |
| `fingerprint_enabled`  | bool     | `true`       | Enable service fingerprint matching against known patterns    |
| `label_priority`       | bool     | `true`       | When true, label-derived data takes precedence over probes    |

### Port Strategy

| Strategy    | Behaviour                                                    |
| ----------- | ------------------------------------------------------------ |
| `published` | Probe the host-published port (e.g., `8080:3000` probes `8080`) |
| `container` | Probe the container-internal port (e.g., probes `3000` on `localhost`) |
| `both`      | Try published first, fall back to container port             |

## How Discovery Feeds Into the Hub

### Scanning Phase

1. `Scanner.Scan()` reads `apps/*/docker-compose.yaml` files.
2. `ParseLabels()` extracts `AppCapability` entries from labels.
3. If discovery probes are enabled, `ProbeEngine.ProbeAll()` enriches apps with runtime data.
4. Results are merged into `App.Capabilities[]` and persisted to MySQL via `hub_store`.

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

| Decision                                                                 | Rationale                                                    |
| ------------------------------------------------------------------------ | ------------------------------------------------------------ |
| Label convention uses a flat namespace (`flowbot.*`)                    | Matches docker-compose label conventions; discoverable by grep |
| Labels take priority over probe results when `label_priority` is true   | User intent is authoritative; probes are best-effort          |
| Probe engine returns `nil` when disabled (not an empty engine)          | Avoids unnecessary allocations; callers check for nil         |
| Homelab capability types use string constants, not `hub.CapabilityType` | Breaks import cycle between `homelab` and `hub` packages      |
| Only TCP ports are probed (UDP skipped)                                 | Self-hosted APIs are HTTP-based; UDP probing adds noise       |
| Probes do not follow redirects                                          | Avoids probing login pages that return 200 on redirect        |

## Adding New Service Fingerprints

To add fingerprint support for a new service, edit
`pkg/homelab/probe/fingerprints.go`:

```go
{
    Capability: "bookmark",
    Provider:   "your-provider",
    Patterns: []FingerprintPattern{
        {Field: "path", Key: "/api/v1/health", Value: ""},
    },
}
```

Each pattern contributes a confidence score. When the cumulative confidence exceeds
zero, the engine emits a match. Currently only the `path` field type is implemented;
future work may add `header` and `body_key` field types with regex `Value` patterns.

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

## Limitations

- **No container network resolution**: Probes use `localhost` for published ports.
  Docker network-level resolution (e.g., `docker inspect` for container IPs) is not
  yet implemented. This means probes may fail for containers on isolated networks.
- **Fingerprint engine is path-only**: Header pattern matching and body-key JSON
  parsing are defined in the type system but not yet implemented in the matcher.
- **Auth detection is heuristic**: API key header detection is inferred from the
  absence of a `WWW-Authenticate` header on 401/403 responses. The exact header
  name (e.g., `X-API-Key` vs `X-Auth-Token`) cannot be determined from the
  response alone.
- **Single capability per app**: The current label convention supports one
  capability per compose file. Multi-service apps with different capabilities
  require separate compose files or manual hub registration.
