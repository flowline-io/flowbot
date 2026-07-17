---
name: memos
description: >-
  Create, list, update, and delete memos via flowbot memo. Use when the user mentions memos, memo notes, scratchpad, quick notes, jotting.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: memos
  cli_root: memo
---

# Memos

Use `flowbot memo` for capability `memos`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

## Workflows

### Capture a quick note

When a user wants to save a short memo:
1. `flowbot memo create -c "<content>"`
2. Report back with the memo name.

### Review recent memos

When a user wants to browse or open memos:
1. `flowbot memo list --limit 20`
2. `flowbot memo get <name>`
3. Present the memo content to the user.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| empty results | confirm server health and capability access scopes |
