# Karakeep CLI reference

Capability `karakeep`. Root command: `flowbot bookmark`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Toggle archive status of a bookmark

`flowbot bookmark archive <id> [flags]`

Archive or unarchive a bookmark by ID

Flags: `--yes` (`-y`) bool — Skip confirmation

### Check if a URL is already bookmarked

`flowbot bookmark check-url --url <url>`

Check if a URL exists in the bookmark collection

Flags: `--url` (`-u`) string, required — URL to check

### Create a new bookmark

`flowbot bookmark create --url <url>`

Add a new bookmark to the Flowbot server

Flags: `--url` (`-u`) string, required — Bookmark URL

### Delete (archive) a bookmark

`flowbot bookmark delete <id> [flags]`

Archive a bookmark by ID

Flags: `--yes` (`-y`) bool — Skip confirmation

### Get a bookmark by ID

`flowbot bookmark get <id>`

Display details of a specific bookmark

### List all bookmarks

`flowbot bookmark list [flags]`

Display bookmarks from the Flowbot server

Flags: `--cursor` (`-c`) string — Pagination cursor; `--limit` (`-n`) int — Maximum number of bookmarks

### Search bookmarks

`flowbot bookmark search --query <query> [flags]`

Full-text search across all bookmarks

Flags: `--cursor` (`-c`) string — Pagination cursor; `--include-content` (`-i`) bool — Include full content in results; `--limit` (`-n`) int — Maximum number of results; `--query` (`-q`) string, required — Search query; `--sort-order` (`-s`) string — Sort order (asc, desc, relevance)
