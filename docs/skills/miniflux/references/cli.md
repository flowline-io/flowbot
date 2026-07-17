# Miniflux CLI reference

Capability `miniflux`. Root command: `flowbot reader`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Create a new feed

`flowbot reader create --url <url> [flags]`

Add a new RSS feed to the Flowbot server

Flags: `--category` (`-c`) int64 — Category ID; `--url` (`-u`) string, required — Feed URL

### List entries

`flowbot reader entries [flags]`

Display RSS entries from Flowbot server

Flags: `--limit` (`-n`) int — Maximum number of entries; `--offset` int — Pagination offset; `--starred` bool — Starred entries only; `--status` (`-s`) string — Status filter (read, unread, removed)

### Get entries for a feed

`flowbot reader feed-entries <feed-id> [flags]`

Display RSS entries for a specific feed

Flags: `--limit` (`-n`) int — Maximum number of entries; `--offset` int — Pagination offset; `--starred` bool — Starred entries only; `--status` (`-s`) string — Status filter (read, unread, removed)

### Get a feed by ID

`flowbot reader get <id>`

Display details of a specific RSS feed

### List all feeds

`flowbot reader list`

Display all RSS feeds from Flowbot server

### Refresh a feed

`flowbot reader refresh <id>`

Trigger a refresh of a specific RSS feed

### Update a feed

`flowbot reader update <id> [flags]`

Modify an existing RSS feed

Flags: `--disable` bool — Disable the feed; `--enable` bool — Enable the feed; `--title` (`-t`) string — New title; `--url` (`-u`) string — New feed URL

### Update entries status

`flowbot reader update-entries --ids <ids> --status <status>`

Update the status of multiple entries

Flags: `--ids` (`-i`) int64Slice, required — Entry IDs to update; `--status` (`-s`) string, required — New status (read, unread, removed)
