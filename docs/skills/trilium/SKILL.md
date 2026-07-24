---
name: trilium
description: >-
  Create, list, search, update, and delete trilium notes via flowbot trilium. Use when the user mentions trilium, notes, knowledge base, note tree, personal wiki.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: trilium
  cli_root: trilium
---

# Trilium

Use `flowbot trilium` for capability `trilium`.
**CLI root is `trilium`** — do not invent `flowbot trilium` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**JSON fields:** Note ids are strings; create requires an existing `parent_note_id`.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:trilium:read` / `service:trilium:write`

## Workflows

### Create a note under a parent

When a user wants to add a new trilium note:
1. `flowbot trilium list --limit 20`
2. Choose parent_note_id from list/get (ask the user if unclear).
3. `flowbot trilium create -t "<title>" -c "<content>" -p <parent_note_id>`
4. Report back with the note ID.

### Find and read a note

When a user wants to search and open trilium notes:
1. `flowbot trilium search -q "<keywords>"`
2. `flowbot trilium get <id>`
3. `flowbot trilium content get <id>`
4. Present the note content to the user.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:trilium:read` / `service:trilium:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot trilium`, not the capability id as the CLI verb |
