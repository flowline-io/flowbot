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

Use `flowbot trilium` for capability `trilium`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

## Workflows

### Create a note under a parent

When a user wants to add a new trilium note:
1. `flowbot trilium create -t "<title>" -c "<content>" -p <parent_note_id>`
2. Report back with the note ID.

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
| empty results | confirm server health and capability access scopes |
