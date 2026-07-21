# Workflow CLI reference

Platform skill (not a hub capability). Root command: `flowbot workflow`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Apply a workflow YAML definition

`flowbot workflow apply --file <file>`

Flags: `--file` string, required — Path to workflow YAML file

### Delete a workflow definition

`flowbot workflow delete <name>`

### Export a workflow as YAML

`flowbot workflow export <name>`

### Get a workflow definition

`flowbot workflow get <name>`

### List workflows

`flowbot workflow list`

### Run a stored workflow asynchronously

`flowbot workflow run <name> [flags]`

Flags: `--input` string — JSON object of workflow inputs

### List runs for a workflow

`flowbot workflow runs <name>`
