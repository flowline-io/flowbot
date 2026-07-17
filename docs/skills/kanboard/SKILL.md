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

Use `flowbot kanban` for capability `kanboard`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

## Workflows

### Create a task with subtasks

When a user wants to create a well-structured task:
1. `flowbot kanban column list -p 1`
2. `flowbot kanban create -t "<task title>" -d "<description>" -p 1 -c <column_id>`
3. `flowbot kanban subtask create <task_id> -t "<subtask 1>" -e <minutes>`
4. `flowbot kanban subtask create <task_id> -t "<subtask 2>" -e <minutes>`

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
| empty results | confirm server health and capability access scopes |
