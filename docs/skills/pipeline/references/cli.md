# Pipeline CLI reference

Platform skill (not a hub capability). Root command: `flowbot pipeline`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Apply a pipeline YAML definition

`flowbot pipeline apply --file <file>`

Flags: `--file` string, required — Path to pipeline YAML file

### Delete a pipeline definition (also deletes run history)

`flowbot pipeline delete <name>`

### Export a pipeline as YAML

`flowbot pipeline export <name>`

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
