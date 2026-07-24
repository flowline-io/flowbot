---
name: miniflux
description: >-
  Subscribe to RSS/Atom feeds and manage entries via flowbot reader. Use when the user mentions RSS, Atom, miniflux, feed reader, unread entries, feed subscriptions, mark as read.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: miniflux
  cli_root: reader
---

# Miniflux

Use `flowbot reader` for capability `miniflux`.
**CLI root is `reader`** — do not invent `flowbot miniflux` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**CLI limits:** No `reader refresh` or star/unstar CLI; feed fetch is server-side. Mark read/unread with `reader update-entries`.

**JSON fields:** Feed and entry ids are integers in `-o json` (`id`).

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:miniflux:read` / `service:miniflux:write`

## Workflows

### Subscribe to a new feed

When a user shares a blog or feed URL they want to follow:
1. `flowbot reader create -u <feed_url>`
2. `flowbot reader list`
3. Pick the new feed id from list output (server fetches entries asynchronously).
4. `flowbot reader feed-entries <feed_id> -n 5`
5. Report the latest entries to the user; retry feed-entries shortly if still empty.

### Catch up on unread entries

When a user wants to see what's new across all feeds:
1. `flowbot reader entries -s unread -n 20`
2. Present the entries in a readable format.
3. If the user wants to mark as read, run: flowbot reader update-entries -i <ids> -s read

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:miniflux:read` / `service:miniflux:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot reader`, not the capability id as the CLI verb |
