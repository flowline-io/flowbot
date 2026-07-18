# Firefly III CLI reference

Capability `fireflyiii`. Root command: `flowbot fireflyiii`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Show Firefly III about info

`flowbot fireflyiii about`

Display Firefly III instance metadata

### Create a transaction

`flowbot fireflyiii create --amount <amount> --date <date> --description <description> --type <type> [flags]`

Create a new Firefly III transaction (requires source and destination account id or name)

Flags: `--amount` (`-a`) string, required — Transaction amount; `--category` string — Category name; `--date` string, required — Transaction date (YYYY-MM-DD); `--description` (`-m`) string, required — Transaction description; `--destination-id` string — Destination account ID (required if --destination-name omitted); `--destination-name` string — Destination account name (required if --destination-id omitted); `--notes` string — Notes; `--source-id` string — Source account ID (required if --source-name omitted); `--source-name` string — Source account name (required if --source-id omitted); `--type` (`-t`) string, required — Transaction type (withdrawal, deposit, transfer)

### Check Firefly III backend health

`flowbot fireflyiii health`

Check whether the Firefly III backend is reachable

### Show current Firefly III user

`flowbot fireflyiii user`

Display the authenticated Firefly III user
