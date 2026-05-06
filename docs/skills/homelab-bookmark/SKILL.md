---
name: homelab-bookmark
description: >
  Manage bookmarks via the Flowbot CLI. Create, list, search, archive, and tag bookmarks stored in the Flowbot server.
  Make sure to use this skill whenever the user mentions bookmarks, saving URLs, link collection, web clippings, reading list, tagging URLs, URL archiving, checking saved links.
---

# Flowbot Bookmark

Manage bookmarks via the Flowbot CLI. Create, list, search, archive, and tag bookmarks stored in the Flowbot server.

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

### Create a new bookmark

**Command:** `flowbot bookmark create --url <url>`
Add a new bookmark to the Flowbot server

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--url` | `-u` | string | yes | Bookmark URL |

---

### List all bookmarks

**Command:** `flowbot bookmark list [flags]`
Display bookmarks from the Flowbot server

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--limit` | `-n` | int | no | Maximum number of bookmarks |

---

### Get a bookmark by ID

**Command:** `flowbot bookmark get <id>`
Display details of a specific bookmark

**Positional Arguments:**
- `<id>`

---

### Toggle archive status of a bookmark

**Command:** `flowbot bookmark archive <id> [flags]`
Archive or unarchive a bookmark by ID

**Positional Arguments:**
- `<id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--yes` | `-y` | bool | no | Skip confirmation |

---

### Delete (archive) a bookmark

**Command:** `flowbot bookmark delete <id> [flags]`
Archive a bookmark by ID

**Positional Arguments:**
- `<id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--yes` | `-y` | bool | no | Skip confirmation |

---

### Check if a URL is already bookmarked

**Command:** `flowbot bookmark check-url --url <url>`
Check if a URL exists in the bookmark collection

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--url` | `-u` | string | yes | URL to check |

---

### Search bookmarks

**Command:** `flowbot bookmark search --query <query> [flags]`
Full-text search across all bookmarks

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--query` | `-q` | string | yes | Search query |
| `--sort-order` | `-s` | string | no | Sort order (asc, desc, relevance) |
| `--limit` | `-n` | int | no | Maximum number of results |
| `--cursor` | `-c` | string | no | Pagination cursor |
| `--include-content` | `-i` | bool | no | Include full content in results |

---

## Common Workflows

### Save a URL from a chat message

When a user shares a URL they want to save:

1. `flowbot bookmark check-url -u <url>`
2. `flowbot bookmark create -u <url>`
3. `Report back with the bookmark details including the assigned ID.`


### Find and review bookmarks

When a user wants to find previously saved content:

1. `flowbot bookmark search -q "<keywords>" --limit 10`
2. `flowbot bookmark get <id>`
3. `Present the bookmark details to the user.`


## Troubleshooting

- **"not logged in"**: Run `flowbot login` first.
- **"server URL is required"**: Set `FLOWBOT_SERVER_URL` env var or use `--server-url` flag.
- **Empty results**: Check the server is running and you have access to the requested resources.
