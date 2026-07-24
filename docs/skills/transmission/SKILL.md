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

Use `flowbot transmission` for capability `transmission`.
**CLI root is `transmission`** — do not invent `flowbot transmission` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**JSON fields:** Torrent ids are integers; use `list` JSON `id`.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:transmission:read` / `service:transmission:write`

## Workflows

### Add a torrent

When a user wants to download a magnet link or torrent URL:
1. `flowbot transmission add -u "<magnet-or-http-url>"`
2. Report back with the torrent ID and name.

### Inspect and control downloads

When a user asks about current downloads or wants to stop/remove one:
1. `flowbot transmission list`
2. `flowbot transmission stop --ids <id>`
3. Remove only after explicit confirmation: flowbot transmission remove --ids <id> (no -y flag; confirm with the user first).

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:transmission:read` / `service:transmission:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot transmission`, not the capability id as the CLI verb |
