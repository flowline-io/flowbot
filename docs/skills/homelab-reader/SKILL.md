---
name: homelab-reader
description: >
  Manage RSS and Atom feed subscriptions via the Flowbot CLI. Add feeds, list entries, mark items read/unread, star entries, and manage feed lifecycle.
  Make sure to use this skill whenever the user mentions RSS feeds, RSS reader, feed reader, news feeds, Atom feeds, subscribing to blogs, reading feeds, feed management, marking read, starring articles.
---

# Flowbot Reader

Manage RSS and Atom feed subscriptions via the Flowbot CLI. Add feeds, list entries, mark items read/unread, star entries, and manage feed lifecycle.

## Prerequisites

- The `flowbot` CLI must be installed and logged in (`flowbot login`).
- The Flowbot server must be running and reachable.
- Global flags: `--server-url` (server address), `--profile` (config profile), `--debug` (enable debug logging).

## Global Flags Reference

| Flag | Shorthand | Type | Description |
|------|-----------|------|-------------|
| `--server-url` | | string | Flowbot server URL (or set `FLOWBOT_SERVER_URL` env var) |
| `--profile` | | string | Configuration profile name |
| `--debug` | `-d` | bool | Enable debug mode |

## Common Output Options

Most commands support `--output` / `-o` to choose between `table` (default, human-readable) and `json` (structured) output.

## Operations

### List all feeds

**Command:** `flowbot reader list`
Display all RSS feeds from Flowbot server

---

### Get a feed by ID

**Command:** `flowbot reader get <id>`
Display details of a specific RSS feed

**Positional Arguments:**
- `<id>`

---

### Create a new feed

**Command:** `flowbot reader create --url <url> [flags]`
Add a new RSS feed to the Flowbot server

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--url` | `-u` | string | yes | Feed URL |
| `--category` | `-c` | int | no | Category ID |

---

### Update a feed

**Command:** `flowbot reader update <id> [flags]`
Modify an existing RSS feed

**Positional Arguments:**
- `<id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--title` | `-t` | string | no | New title |
| `--url` | `-u` | string | no | New feed URL |
| `--disable` |  | bool | no | Disable the feed |
| `--enable` |  | bool | no | Enable the feed |

---

### Refresh a feed

**Command:** `flowbot reader refresh <id>`
Trigger a refresh of a specific RSS feed

**Positional Arguments:**
- `<id>`

---

### List entries

**Command:** `flowbot reader entries [flags]`
Display RSS entries from Flowbot server

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--status` | `-s` | string | no | Status filter (read, unread, removed) |
| `--limit` | `-n` | int | no | Maximum number of entries |
| `--offset` |  | int | no | Pagination offset |
| `--starred` |  | bool | no | Starred entries only |

---

### Update entries status

**Command:** `flowbot reader update-entries --ids <ids> --status <status>`
Update the status of multiple entries

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--ids` | `-i` | int | yes | Entry IDs to update |
| `--status` | `-s` | string | yes | New status (read, unread, removed) |

---

### Get entries for a feed

**Command:** `flowbot reader feed-entries <feed-id> [flags]`
Display RSS entries for a specific feed

**Positional Arguments:**
- `<feed-id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--status` | `-s` | string | no | Status filter (read, unread, removed) |
| `--limit` | `-n` | int | no | Maximum number of entries |
| `--offset` |  | int | no | Pagination offset |
| `--starred` |  | bool | no | Starred entries only |

---

## Common Workflows

### Subscribe to a new feed

When a user shares a blog or feed URL they want to follow:

1. `flowbot reader create -u <feed_url>`
2. `flowbot reader refresh <feed_id>`
3. `flowbot reader feed-entries <feed_id> -n 5`
4. `Report the latest entries to the user.`


### Catch up on unread entries

When a user wants to see what's new across all feeds:

1. `flowbot reader entries -s unread -n 20`
2. `Present the entries in a readable format.`
3. `If the user wants to mark as read: flowbot reader update-entries -i <ids> -s read`


## Troubleshooting

- **"not logged in"**: Run `flowbot login` first.
- **"server URL is required"**: Set `FLOWBOT_SERVER_URL` env var or use `--server-url` flag.
- **Empty results**: Check the server is running and you have access to the requested resources.
