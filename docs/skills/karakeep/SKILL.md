---
name: karakeep
description: >-
  Create, list, search, archive, and delete bookmarks via flowbot bookmark. Use when the user mentions bookmarks, karakeep, saved URLs, reading list, link archiving, web clippings.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: karakeep
  cli_root: bookmark
---

# Karakeep

Use `flowbot bookmark` for capability `karakeep`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

## Workflows

### Save a URL from a chat message

When a user shares a URL they want to save:
1. `flowbot bookmark check-url -u <url>`
2. `flowbot bookmark create -u <url>`
3. Report back with the bookmark details including the assigned ID.

### Find and review bookmarks

When a user wants to find previously saved content:
1. `flowbot bookmark search -q "<keywords>" --limit 10`
2. `flowbot bookmark get <id>`
3. Present the bookmark details to the user.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| empty results | confirm server health and capability access scopes |
