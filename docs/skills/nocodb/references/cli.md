# NocoDB CLI reference

Capability `nocodb`. Root command: `flowbot nocodb`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### List bases

`flowbot nocodb bases`

List NocoDB bases visible to the configured API token (first page)

### List tables

`flowbot nocodb tables --base-id <base-id>`

List tables belonging to a NocoDB base (first page)

Flags: `--base-id` string, required — Base ID

### Get table metadata

`flowbot nocodb table --table-id <table-id>`

Get NocoDB table metadata including columns

Flags: `--table-id` string, required — Table ID

### List records

`flowbot nocodb records list --table-id <table-id>`

List records in a NocoDB table (first page; use `--limit` / `--offset` for paging)

Flags:

- `--table-id` string, required — Table ID
- `--limit` int — Max records to return (max 1000)
- `--offset` int — Record offset
- `--where` string — NocoDB where filter
- `--sort` string — Sort expression
- `--fields` string — Comma-separated field names

Record IDs use the default NocoDB `Id` primary key.

### Get a record

`flowbot nocodb records get --table-id <table-id> --record-id <record-id>`

Get a single NocoDB record by ID

Flags: `--table-id` string, required; `--record-id` string, required

### Create a record

`flowbot nocodb records create --table-id <table-id> --fields '{"Name":"Alice"}'`

Create a NocoDB record with JSON field values

Flags: `--table-id` string, required; `--fields` string, required — JSON object

### Update a record

`flowbot nocodb records update --table-id <table-id> --record-id <record-id> --fields '{"Name":"Bob"}'`

Update a NocoDB record with JSON field values

Flags: `--table-id` string, required; `--record-id` string, required; `--fields` string, required

### Delete a record

`flowbot nocodb records delete --table-id <table-id> --record-id <record-id>`

Delete a NocoDB record by ID

Flags: `--table-id` string, required; `--record-id` string, required

### Check NocoDB backend health

`flowbot nocodb health`

Check whether the NocoDB backend is reachable
