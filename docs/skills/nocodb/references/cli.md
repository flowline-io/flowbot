# NocoDB CLI reference

Capability `nocodb`. Root command: `flowbot nocodb`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`, `--idempotency-key`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict`, `already_exists`, or `idempotency_unknown`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. `idempotency_unknown` is exit 5 with `retryable: false`: list or get, and use a new idempotency key only when that read shows the effect did not happen. A write still in progress is `unavailable`, exit 6: retry the same call with the same idempotency key. Mint an idempotency_key before the first write and send that same key on every retry. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### List bases

`flowbot nocodb bases`

List NocoDB bases visible to the configured API token (first page)

### Check NocoDB backend health

`flowbot nocodb health`

Check whether the NocoDB backend is reachable

### Create a record

`flowbot nocodb records create --fields <fields> --table-id <table-id>`

Create a NocoDB record with JSON field values

Flags: `--fields` string, required — JSON object of field values, e.g. {"Name":"Alice"}; `--table-id` string, required — Table ID

### Delete a record

`flowbot nocodb records delete --record-id <record-id> --table-id <table-id>`

Delete a NocoDB record by ID

Flags: `--record-id` string, required — Record ID; `--table-id` string, required — Table ID

### Get a record

`flowbot nocodb records get --record-id <record-id> --table-id <table-id>`

Get a single NocoDB record by ID

Flags: `--record-id` string, required — Record ID; `--table-id` string, required — Table ID

### List records

`flowbot nocodb records list --table-id <table-id> [flags]`

List records in a NocoDB table

Flags: `--fields` string — Comma-separated field names; `--limit` int — Max records to return; `--offset` int — Record offset; `--sort` string — Sort expression; `--table-id` string, required — Table ID; `--where` string — NocoDB where filter

### Update a record

`flowbot nocodb records update --fields <fields> --record-id <record-id> --table-id <table-id>`

Update a NocoDB record with JSON field values

Flags: `--fields` string, required — JSON object of field values, e.g. {"Name":"Bob"}; `--record-id` string, required — Record ID; `--table-id` string, required — Table ID

### Get table metadata

`flowbot nocodb table --table-id <table-id>`

Get NocoDB table metadata including columns

Flags: `--table-id` string, required — Table ID

### List tables in a base

`flowbot nocodb tables --base-id <base-id>`

List tables belonging to a NocoDB base (first page)

Flags: `--base-id` string, required — Base ID
