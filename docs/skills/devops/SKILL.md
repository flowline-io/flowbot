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

Use `flowbot devops` for capability `devops`.
**CLI root is `devops`** — do not invent `flowbot devops` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**CLI limits:** There is no `flowbot devops health`; use `flowbot devops status` for aggregate readiness, then backend-specific health/metrics commands.

**JSON fields:** Prefer `-o json`. Start with `devops status` to see which backends are configured.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:devops:read`

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
5. `flowbot devops wakapi projects`
6. `flowbot devops dozzle health`
7. `flowbot devops netalertx totals`

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
| permission denied / 403 | token missing service scopes (`service:devops:read`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot devops`, not the capability id as the CLI verb |
