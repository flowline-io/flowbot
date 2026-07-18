# Trilium CLI reference

Capability `trilium`. Root command: `flowbot trilium`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Get note content

`flowbot trilium content get <id>`

Display the full content of a note

### Set note content

`flowbot trilium content set <id> --content <content>`

Replace the full content of a note

Flags: `--content` (`-c`) string, required — New note content

### Create a new note

`flowbot trilium create --title <title> [flags]`

Add a new note to the trilium backend

Flags: `--content` (`-c`) string — Note content; `--parent` (`-p`) string — Parent note ID; `--title` (`-t`) string, required — Note title; `--type` string — Note type (default: text)

### Delete a note

`flowbot trilium delete <id> [flags]`

Delete a note by its ID

Flags: `--yes` (`-y`) bool — Skip confirmation

### Get a note by ID

`flowbot trilium get <id>`

Display details of a specific note

### Check trilium backend health

`flowbot trilium health`

Check whether the trilium backend is reachable

### List notes

`flowbot trilium list [flags]`

Display notes from the trilium backend

Flags: `--limit` (`-n`) int — Maximum number of notes; `--query` (`-q`) string — Optional search filter

### Search notes

`flowbot trilium search --query <query>`

Full-text search across trilium notes

Flags: `--query` (`-q`) string, required — Search query

### Update a note

`flowbot trilium update <id> [flags]`

Update title and/or content of a note

Flags: `--content` (`-c`) string — New content; `--title` (`-t`) string — New title
