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

Use `flowbot bookmark` for capability `karakeep`.
**CLI root is `bookmark`** — do not invent `flowbot karakeep` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**CLI limits:** Tag attach/detach and health are capability ops without CLI commands; inspect tags via `bookmark get`.

**JSON fields:** Bookmark id is a string field `id` in `-o json` output.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:karakeep:read` / `service:karakeep:write`

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

### Archive a bookmark

When a user wants to archive a saved URL:
1. `flowbot bookmark archive <id> -y`
2. Confirm archive status from the command output.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:karakeep:read` / `service:karakeep:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot bookmark`, not the capability id as the CLI verb |
