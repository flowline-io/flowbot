# Firefly III CLI reference

Capability `fireflyiii`. Root command: `flowbot fireflyiii`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict` or `already_exists`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### Show Firefly III about info

`flowbot fireflyiii about`

Display Firefly III instance metadata

### List accounts

`flowbot fireflyiii accounts [flags]`

List Firefly III accounts

Flags: `--cursor` (`-c`) string — Pagination cursor; `--limit` (`-n`) int — Maximum number of accounts; `--type` (`-t`) string — Account type filter (asset, expense, revenue, ...)

### List bills

`flowbot fireflyiii bills [flags]`

List Firefly III bills

Flags: `--cursor` (`-c`) string — Pagination cursor; `--limit` (`-n`) int — Maximum number of items

### List budgets

`flowbot fireflyiii budgets [flags]`

List Firefly III budgets

Flags: `--cursor` (`-c`) string — Pagination cursor; `--limit` (`-n`) int — Maximum number of items

### List categories

`flowbot fireflyiii categories [flags]`

List Firefly III categories

Flags: `--cursor` (`-c`) string — Pagination cursor; `--limit` (`-n`) int — Maximum number of items

### Create a transaction

`flowbot fireflyiii create --amount <amount> --date <date> --description <description> --type <type> [flags]`

Create a new Firefly III transaction (requires source and destination account id or name)

Flags: `--amount` (`-a`) string, required — Transaction amount; `--bill-id` string — Bill ID; `--bill-name` string — Bill name; `--budget-id` string — Budget ID; `--budget-name` string — Budget name; `--category` string — Category name; `--date` string, required — Transaction date (YYYY-MM-DD); `--description` (`-m`) string, required — Transaction description; `--destination-id` string — Destination account ID (required if --destination-name omitted); `--destination-name` string — Destination account name (required if --destination-id omitted); `--notes` string — Notes; `--source-id` string — Source account ID (required if --source-name omitted); `--source-name` string — Source account name (required if --source-id omitted); `--tag` stringArray — Tag (repeatable); `--type` (`-t`) string, required — Transaction type (withdrawal, deposit, transfer)

### Delete a transaction

`flowbot fireflyiii delete <id>`

Delete a Firefly III transaction group by ID

### Get a transaction

`flowbot fireflyiii get <id>`

Display a Firefly III transaction group by ID

### Check Firefly III backend health

`flowbot fireflyiii health`

Check whether the Firefly III backend is reachable

### List transactions

`flowbot fireflyiii list [flags]`

List Firefly III transaction groups

Flags: `--account-id` string — Filter by account ID; `--cursor` (`-c`) string — Pagination cursor; `--end` string — End date (YYYY-MM-DD); `--limit` (`-n`) int — Maximum number of transactions; `--start` string — Start date (YYYY-MM-DD); `--type` (`-t`) string — Transaction type filter

### List tags

`flowbot fireflyiii tags [flags]`

List Firefly III tags

Flags: `--cursor` (`-c`) string — Pagination cursor; `--limit` (`-n`) int — Maximum number of items

### Update a transaction

`flowbot fireflyiii update <id> [flags]`

Partially update a Firefly III transaction group (first split only)

Flags: `--amount` (`-a`) string — Transaction amount; `--bill-id` string — Bill ID; `--bill-name` string — Bill name; `--budget-id` string — Budget ID; `--budget-name` string — Budget name; `--category` string — Category name; `--date` string — Transaction date (YYYY-MM-DD); `--description` (`-m`) string — Transaction description; `--destination-id` string — Destination account ID (required if --destination-name omitted); `--destination-name` string — Destination account name (required if --destination-id omitted); `--notes` string — Notes; `--source-id` string — Source account ID (required if --source-name omitted); `--source-name` string — Source account name (required if --source-id omitted); `--tag` stringArray — Tag (repeatable); `--type` (`-t`) string — Transaction type (withdrawal, deposit, transfer)

### Show current Firefly III user

`flowbot fireflyiii user`

Display the authenticated Firefly III user
