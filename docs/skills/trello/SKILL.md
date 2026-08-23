---
name: trello
description: >-
  Manage Trello boards, lists, and cards via flowbot trello. Use when the user mentions trello, boards, lists, cards, kanban cloud, project management.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: trello
  cli_root: trello
---

# Trello

Use `flowbot trello` for capability `trello`.
**CLI root is `trello`** — do not invent `flowbot trello` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**JSON fields:** Board, list, and card ids are strings; use `-o json` `id` fields.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:trello:read` / `service:trello:write`

## Workflows

### Create a card on a board

When a user wants to add a task to Trello:
1. `flowbot trello board list`
2. `flowbot trello list list <board_id>`
3. `flowbot trello card create --list-id <list_id> -n "<title>"`

### Review board cards

When a user wants to see cards on a board:
1. `flowbot trello card list <board_id>`
2. `flowbot trello card get <card_id>`

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:trello:read` / `service:trello:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot trello`, not the capability id as the CLI verb |
