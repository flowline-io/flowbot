# Pipeline CLI reference

Platform skill (not a hub capability). Root command: `flowbot pipeline`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict` or `already_exists`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### Apply a pipeline YAML definition

`flowbot pipeline apply --file <file>`

Flags: `--file` string, required — Path to pipeline YAML file

### Delete a pipeline definition (also deletes run history)

`flowbot pipeline delete <name>`

### Export a pipeline as YAML

`flowbot pipeline export <name> [flags]`

Flags: `--file` string — Optional file path to write YAML

### Get a pipeline definition

`flowbot pipeline get <name>`

### List pipelines

`flowbot pipeline list`

Display published pipelines.

### Run a stored pipeline asynchronously

`flowbot pipeline run <name> [flags]`

Flags: `--event` string — JSON object used as synthetic DataEvent payload

### List runs for a pipeline

`flowbot pipeline runs <name>`
