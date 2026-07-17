---
name: miniflux
description: >-
  Subscribe to RSS/Atom feeds and manage entries via flowbot reader. Use when the user mentions RSS, Atom, miniflux, feed reader, unread entries, starring articles, feed subscriptions.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: miniflux
  cli_root: reader
---

# Miniflux

Use `flowbot reader` for capability `miniflux`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

## Workflows

### Subscribe to a new feed

When a user shares a blog or feed URL they want to follow:
1. `flowbot reader create -u <feed_url>`
2. `flowbot reader refresh <feed_id>`
3. `flowbot reader feed-entries <feed_id> -n 5`
4. Report the latest entries to the user.

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
| empty results | confirm server health and capability access scopes |
