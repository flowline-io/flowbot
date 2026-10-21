# Trello CLI reference

Capability `trello`. Root command: `flowbot trello`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`, `--idempotency-key`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict`, `already_exists`, or `idempotency_unknown`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. `idempotency_unknown` is exit 5 with `retryable: false`: list or get, and use a new idempotency key only when that read shows the effect did not happen. A write still in progress is `unavailable`, exit 6: retry the same call with the same idempotency key. Mint an idempotency_key before the first write and send that same key on every retry. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### List Trello boards

`flowbot trello board [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page

### Get a Trello board

`flowbot trello board [board_id]`

### Create a Trello card

`flowbot trello card --list-id <list-id> --name <name> [flags]`

Flags: `--desc` string — Card description; `--list-id` string, required — Target list ID; `--name` (`-n`) string, required — Card title

### List cards on a board

`flowbot trello card [board_id] [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page

### Get a Trello card

`flowbot trello card [card_id]`

### Search Trello cards

`flowbot trello card --query <query> [flags]`

Flags: `--limit` int — Maximum results; `--query` (`-q`) string, required — Search query

### Update a Trello card

`flowbot trello card [card_id] [flags]`

Flags: `--desc` string — New description; `--name` (`-n`) string — New title

### Move a card to another list

`flowbot trello card [card_id] --list-id <list-id>`

Flags: `--list-id` string, required — Target list ID

### Delete a Trello card

`flowbot trello card [card_id]`

### Check Trello connectivity

`flowbot trello health`

### List lists on a board

`flowbot trello list [board_id]`

### Register a Trello board webhook

`flowbot trello webhook [flags]`

Flags: `--board-id` string — Board ID; `--callback-url` string — Callback URL; `--description` string — Webhook description

### Delete a Trello webhook

`flowbot trello webhook [webhook_id]`
