# DevOps CLI reference

Capability `devops`. Root command: `flowbot devops`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict` or `already_exists`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### Get a Beszel system

`flowbot devops beszel get --id <id>`

Flags: `--id` string, required — System ID

### List Beszel systems

`flowbot devops beszel systems`

### Check Dozzle health

`flowbot devops dozzle health`

### Search Grafana dashboards

`flowbot devops grafana dashboards [flags]`

Flags: `--query` string — Search query

### List Grafana datasources

`flowbot devops grafana datasources`

### Check Grafana health

`flowbot devops grafana health`

### Query prometheus, alloy, loki, tempo, or pyroscope via Grafana

`flowbot devops grafana query --backend <backend> --expr <expr> [flags]`

Flags: `--backend` string, required — prometheus|alloy|loki|tempo|pyroscope; `--datasource-uid` string — Optional Grafana datasource UID; `--expr` string, required — Query expression (PromQL/LogQL/TraceQL/label selector); `--from` string — Grafana from time; `--max-lines` int — Loki max lines; `--to` string — Grafana to time

### List NetAlertX devices

`flowbot devops netalertx devices`

### Check NetAlertX health

`flowbot devops netalertx health`

### Search NetAlertX devices

`flowbot devops netalertx search --query <query>`

Flags: `--query` string, required — Search query (MAC, name, or IP)

### Show NetAlertX device totals

`flowbot devops netalertx totals`

### List Scanopy daemons

`flowbot devops scanopy daemons [flags]`

Flags: `--cursor` string — Opaque pagination cursor; `--limit` int — Page size (default 50, max 1000); `--network-id` string — Filter by network UUID; `--search` string — Free-text search

### Check Scanopy health

`flowbot devops scanopy health`

### Get a Scanopy host by ID

`flowbot devops scanopy host --id <id>`

Flags: `--id` string, required — Host UUID

### List Scanopy hosts

`flowbot devops scanopy hosts [flags]`

Flags: `--cursor` string — Opaque pagination cursor; `--limit` int — Page size (default 50, max 1000); `--network-id` string — Filter by network UUID; `--search` string — Free-text search

### List Scanopy networks

`flowbot devops scanopy networks [flags]`

Flags: `--cursor` string — Opaque pagination cursor; `--limit` int — Page size (default 50, max 1000); `--network-id` string — Filter by network UUID; `--search` string — Free-text search

### List Scanopy services

`flowbot devops scanopy services [flags]`

Flags: `--cursor` string — Opaque pagination cursor; `--host-id` string — Filter by host UUID; `--limit` int — Page size (default 50, max 1000); `--network-id` string — Filter by network UUID; `--search` string — Free-text search

### Show Scanopy API/server version

`flowbot devops scanopy version`

### Show configured devops backends

`flowbot devops status`

### Show Traefik overview counts

`flowbot devops traefik overview`

### List Traefik HTTP routers

`flowbot devops traefik routers`

### List Traefik HTTP services

`flowbot devops traefik services`

### Check Uptime Kuma health

`flowbot devops uptimekuma health`

### Summarize Uptime Kuma Prometheus metrics

`flowbot devops uptimekuma metrics`

### Show Wakapi all-time coding stats

`flowbot devops wakapi all-time`

### Check Wakapi health

`flowbot devops wakapi health`

### List Wakapi projects

`flowbot devops wakapi projects [flags]`

Flags: `--query` string — Project name filter

### Show Wakapi activity summary

`flowbot devops wakapi summary [flags]`

Flags: `--interval` string — Summary interval; `--project` string — Project filter
