# Transmission CLI reference

Capability `transmission`. Root command: `flowbot transmission`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Add a torrent

`flowbot transmission add --url <url>`

Add a torrent by magnet link or HTTP(S) .torrent URL

Flags: `--url` (`-u`) string, required — Magnet link or torrent file URL

### Check Transmission backend health

`flowbot transmission health`

Check whether the Transmission backend is reachable

### List torrents

`flowbot transmission list`

Display torrents from Transmission

### Remove torrents

`flowbot transmission remove --ids <ids>`

Remove one or more torrents by ID (downloaded data is kept)

Flags: `--ids` int64Slice, required — Torrent IDs to remove

### Stop torrents

`flowbot transmission stop --ids <ids>`

Stop one or more torrents by ID

Flags: `--ids` int64Slice, required — Torrent IDs to stop
