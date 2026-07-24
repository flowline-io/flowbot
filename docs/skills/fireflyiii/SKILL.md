---
name: fireflyiii
description: >-
  Create Firefly III transactions and inspect instance health via flowbot fireflyiii. Use when the user mentions fireflyiii, firefly, finance, transactions, expenses, budgeting, accounting.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: fireflyiii
  cli_root: fireflyiii
---

# Firefly III

Use `flowbot fireflyiii` for capability `fireflyiii`.
**CLI root is `fireflyiii`** — do not invent `flowbot fireflyiii` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**JSON fields:** Transaction id is a string field `id` in create output.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:fireflyiii:read` / `service:fireflyiii:write`

## Workflows

### Record an expense

When a user wants to log a withdrawal or purchase:
1. `flowbot fireflyiii create -t withdrawal --date <YYYY-MM-DD> -a <amount> -m "<description>" --source-name "<account>" --destination-name "<payee>"`
2. Report back with the transaction ID. Source and destination must each use --*-id or --*-name.

### Check Firefly III connectivity

When verifying the finance backend:
1. `flowbot fireflyiii health`
2. `flowbot fireflyiii about`
3. Summarize version and health status.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:fireflyiii:read` / `service:fireflyiii:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot fireflyiii`, not the capability id as the CLI verb |
