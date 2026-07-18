# DevOps CLI reference

Capability `devops`. Root command: `flowbot devops`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

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

### List Wakapi projects

`flowbot devops wakapi projects`

### Show Wakapi activity summary

`flowbot devops wakapi summary [flags]`

Flags: `--interval` string — Summary interval
