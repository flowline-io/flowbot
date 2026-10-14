# Firefly III CLI reference

Capability `fireflyiii`. Root command: `flowbot fireflyiii`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

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

`flowbot fireflyiii delete <id> [flags]`

Delete a Firefly III transaction group by ID

Flags: `--yes` (`-y`) bool — Skip confirmation

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
