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

Use `flowbot nocodb` for capability `nocodb`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

List APIs return the first page from NocoDB; check `page.has_more` / raise `--limit` / `--offset` when needed.
Record ops assume the default NocoDB `Id` primary key (numeric IDs are preferred).

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
3. Use update/delete only with an explicit record-id; prefer `-o json` when parsing results.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| empty results | confirm server health and capability access scopes |
