---
name: devops
description: >-
  Query devops backends (beszel, uptimekuma, traefik, grafana, wakapi, dozzle, netalertx) via flowbot devops. Use when the user mentions devops, beszel, uptimekuma, traefik, grafana, wakapi, dozzle, netalertx, prometheus, loki, tempo, pyroscope, alloy, monitoring, infrastructure.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: devops
  cli_root: devops
---

# DevOps

Use `flowbot devops` for capability `devops`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

## Workflows

### Check which backends are configured

When a user asks what devops tools are available:
1. `flowbot devops status`
2. Only call subcommands for backends reported as configured.

### Inspect monitoring and routing

When a user wants a quick infrastructure snapshot:
1. `flowbot devops beszel systems`
2. `flowbot devops uptimekuma health`
3. `flowbot devops traefik routers`
4. `flowbot devops grafana health`
5. `flowbot devops netalertx totals`

### Query observability backends via Grafana

When a user wants metrics, logs, traces, or profiles:
1. `flowbot devops grafana datasources`
2. `flowbot devops grafana query --backend prometheus --expr 'up'`
3. Use backend alloy|loki|tempo|pyroscope with the matching expression language; prefer -o json for parsing.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| empty results | confirm server health and capability access scopes |
