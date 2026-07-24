---
name: nocodb
description: >-
  Discover NocoDB bases/tables and create, list, update, or delete records via flowbot nocodb. Use when the user mentions nocodb, base, table, record, spreadsheet, database, airtable.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: nocodb
  cli_root: nocodb
---

# NocoDB

Use `flowbot nocodb` for capability `nocodb`.
**CLI root is `nocodb`** — do not invent `flowbot nocodb` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**JSON fields:** Use base-id / table-id / record-id strings from discover commands; `--fields` keys must match column titles.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:nocodb:read` / `service:nocodb:write`

## Workflows

### Discover bases and tables

When a user needs to find which base or table to use:
1. `flowbot nocodb bases`
2. `flowbot nocodb tables --base-id <base-id>`
3. `flowbot nocodb table --table-id <table-id>`
4. Summarize available tables and column titles before writing records.

### Read and write records

When a user wants to inspect or change rows in a table:
1. `flowbot nocodb records list --table-id <table-id>`
2. `flowbot nocodb records create --table-id <table-id> --fields '{"Title":"value"}'`
3. Use update/delete only with an explicit record-id; prefer -o json when parsing results.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:nocodb:read` / `service:nocodb:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot nocodb`, not the capability id as the CLI verb |
