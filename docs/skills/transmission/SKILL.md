---
name: transmission
description: >-
  Add, list, stop, and remove Transmission torrents via flowbot transmission. Use when the user mentions transmission, torrent, magnet, download, bittorrent, seed.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: transmission
  cli_root: transmission
---

# Transmission

Use `flowbot transmission` for capability `transmission`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

## Workflows

### Add a torrent

When a user wants to download a magnet link or torrent URL:
1. `flowbot transmission add -u "<magnet-or-http-url>"`
2. Report back with the torrent ID and name.

### Inspect and control downloads

When a user asks about current downloads or wants to stop/remove one:
1. `flowbot transmission list`
2. `flowbot transmission stop --ids <id>`
3. Use remove only when the user confirms the torrent should be deleted from Transmission.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| empty results | confirm server health and capability access scopes |
