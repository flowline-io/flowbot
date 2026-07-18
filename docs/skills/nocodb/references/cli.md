# NocoDB CLI reference

Capability `nocodb`. Root command: `flowbot nocodb`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

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
