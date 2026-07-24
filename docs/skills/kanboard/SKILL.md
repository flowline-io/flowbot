---
name: kanboard
description: >-
  Manage kanban boards, tasks, subtasks, timers, tags, and metadata via flowbot kanban. Use when the user mentions kanban, kanboard, tasks, todo, subtasks, time tracking, board columns, moving cards.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: kanboard
  cli_root: kanban
---

# Kanboard

Use `flowbot kanban` for capability `kanboard`.
**CLI root is `kanban`** — do not invent `flowbot kanboard` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**CLI limits:** CLI also exposes subtask/tag/timer helpers beyond the core capability CatalogSpec ops; there is no `kanban health` command.

**JSON fields:** Task ids are integers; use `kanban list` / `get` JSON `id`. Column ids come from `kanban column list`.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:kanboard:read` / `service:kanboard:write`

## Workflows

### Create a task with subtasks

When a user wants to create a well-structured task:
1. Ask for or discover project_id (do not assume 1).
2. `flowbot kanban column list -p <project_id>`
3. `flowbot kanban create -t "<task title>" -d "<description>" -p <project_id> -c <column_id>`
4. `flowbot kanban subtask create <task_id> -t "<subtask 1>" -e <minutes>`
5. `flowbot kanban subtask create <task_id> -t "<subtask 2>" -e <minutes>`

### Review and triage tasks

When reviewing the current board state:
1. `flowbot kanban list -s active`
2. `flowbot kanban get <task_id>`
3. `flowbot kanban subtask list <task_id>`
4. Summarize task status, subtask completion, and suggest next actions.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:kanboard:read` / `service:kanboard:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot kanban`, not the capability id as the CLI verb |
