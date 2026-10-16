# Email CLI reference

Capability `email`. Root command: `flowbot email`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict` or `already_exists`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### Get a message by id

`flowbot email get <id>`

### Check email backend health

`flowbot email health`

### List messages

`flowbot email list [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Page size; `--mailbox` string — Mailbox name; `--unseen-only` bool — Only unseen messages

### Mark a message as read

`flowbot email mark-read <id>`

### Mark a message as unread

`flowbot email mark-unread <id>`

### Search messages

`flowbot email search [flags]`

Flags: `--before` string — RFC3339 before; `--cursor` string — Pagination cursor; `--from` string — From filter; `--limit` int — Page size; `--mailbox` string — Mailbox name; `--since` string — RFC3339 since; `--subject` string — Subject filter; `--to` string — To filter; `--unseen` bool — Unseen only

### Send an email

`flowbot email send --subject <subject> --to <to> [flags]`

Flags: `--bcc` stringSlice — Bcc addresses; `--cc` stringSlice — Cc addresses; `--from-name` string — From display name; `--html` string — HTML body; `--subject` string, required — Subject; `--text` string — Plain text body; `--to` stringSlice, required — Recipient addresses
