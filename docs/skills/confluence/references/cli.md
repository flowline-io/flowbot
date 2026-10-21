# Confluence CLI reference

Capability `confluence`. Root command: `flowbot confluence`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`, `--idempotency-key`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict`, `already_exists`, or `idempotency_unknown`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. `idempotency_unknown` is exit 5 with `retryable: false`: list or get, and use a new idempotency key only when that read shows the effect did not happen. A write still in progress is `unavailable`, exit 6: retry the same call with the same idempotency key. Mint an idempotency_key before the first write and send that same key on every retry. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### Check Confluence connectivity

`flowbot confluence health`

### List pages in a space

`flowbot confluence page [space_key] [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page

### Get a Confluence page

`flowbot confluence page [page_id]`

### Get page storage content

`flowbot confluence page [page_id]`

### Search pages with CQL

`flowbot confluence page --cql <cql> [flags]`

Flags: `--cql` string, required — CQL query; `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page

### Create a Confluence page

`flowbot confluence page --title <title> [flags]`

Flags: `--content` (`-c`) string — Storage-format XHTML content; `--space-key` string — Space key; `--title` (`-t`) string, required — Page title

### Update a Confluence page

`flowbot confluence page [page_id] [flags]`

Flags: `--content` (`-c`) string — Storage-format XHTML content; `--title` (`-t`) string — New title

### Delete a Confluence page

`flowbot confluence page [page_id]`

### List Confluence spaces

`flowbot confluence space [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page
