# Gitea CLI reference

Capability `gitea`. Root command: `flowbot forge`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`, `--idempotency-key`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict`, `already_exists`, or `idempotency_unknown`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. `idempotency_unknown` is exit 5 with `retryable: false`: list or get, and use a new idempotency key only when that read shows the effect did not happen. A write still in progress is `unavailable`, exit 6: retry the same call with the same idempotency key. Mint an idempotency_key before the first write and send that same key on every retry. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### Get commit diff

`flowbot forge diff <owner> <repo> <commit-id>`

Display the diff for a specific commit from the forge

### Get file content

`flowbot forge file <owner> <repo> <commit-id> <file-path> [flags]`

Display the content of a file at a specific commit from the forge

Flags: `--line-count` int — Number of lines to return; `--line-start` int — Starting line number

### Get an issue

`flowbot forge issue <owner> <repo> <index>`

Display a single issue from the forge by owner, repo, and index

### List issues

`flowbot forge issues <owner> [flags]`

List issues for an owner from the forge

Flags: `--cursor` string — Pagination cursor; `--limit` (`-n`) int — Maximum number of issues; `--state` (`-s`) string — Issue state filter (open, closed)

### Get a repository

`flowbot forge repo <owner> <repo>`

Display repository details from the forge

### Get authenticated forge user

`flowbot forge user`

Display the authenticated user from the configured forge
