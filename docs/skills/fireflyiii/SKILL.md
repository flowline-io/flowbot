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

Use `flowbot fireflyiii` for capability `fireflyiii`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

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
| empty results | confirm server health and capability access scopes |
