# Karakeep CLI reference

Capability `karakeep`. Root command: `flowbot bookmark`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict` or `already_exists`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### Toggle archive status of a bookmark

`flowbot bookmark archive <id>`

Archive or unarchive a bookmark by ID

### Check if a URL is already bookmarked

`flowbot bookmark check-url --url <url>`

Check if a URL exists in the bookmark collection

Flags: `--url` (`-u`) string, required — URL to check

### Create a new bookmark

`flowbot bookmark create --url <url>`

Add a new bookmark to the Flowbot server

Flags: `--url` (`-u`) string, required — Bookmark URL

### Delete (archive) a bookmark

`flowbot bookmark delete <id>`

Archive a bookmark by ID

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
