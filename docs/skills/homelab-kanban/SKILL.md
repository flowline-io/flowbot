---
name: homelab-kanban
description: >
  Manage kanban boards and tasks via the Flowbot CLI. Create, update, move, and search tasks. Manage subtasks with time tracking, tags, columns, and metadata.
  Make sure to use this skill whenever the user mentions kanban, task management, project management, kanban board, todo list, task tracking, issue tracking, subtasks, time tracking, moving cards, board columns, task tags.
---

# Flowbot Kanban

Manage kanban boards and tasks via the Flowbot CLI. Create, update, move, and search tasks. Manage subtasks with time tracking, tags, columns, and metadata.

## Prerequisites

- The `flowbot` CLI must be installed and logged in (`flowbot login`).
- The Flowbot server must be running and reachable.
- Global flags: `--server-url` (server address), `--profile` (config profile), `--debug` (enable debug logging).

## Global Flags Reference

| Flag | Shorthand | Type | Description |
|------|-----------|------|-------------|
| `--server-url` | | string | Flowbot server URL (or set `FLOWBOT_SERVER_URL` env var) |
| `--profile` | | string | Configuration profile name |
| `--debug` | `-d` | bool | Enable debug mode |

## Common Output Options

Most commands support `--output` / `-o` to choose between `table` (default, human-readable) and `json` (structured) output.

## Operations

### List all kanban tasks

**Command:** `flowbot kanban list [flags]`
Display kanban tasks from Flowbot server

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--project` | `-p` | int | no | Project ID |
| `--status` | `-s` | string | no | Status filter (active, inactive, all) |

---

### Search kanban tasks

**Command:** `flowbot kanban search <query> [flags]`
Search tasks using kanboard search syntax

**Positional Arguments:**
- `<query>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--project` | `-p` | int | no | Project ID |

---

### Get a kanban task by ID

**Command:** `flowbot kanban get <id>`
Display details of a specific kanban task

**Positional Arguments:**
- `<id>`

---

### Create a new kanban task

**Command:** `flowbot kanban create --title <title> [flags]`
Add a new task to the kanban board

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--title` | `-t` | string | yes | Task title |
| `--description` | `-d` | string | no | Task description |
| `--project` | `-p` | int | no | Project ID |
| `--column` | `-c` | int | no | Column ID |

---

### Update a kanban task

**Command:** `flowbot kanban update <id> [flags]`
Modify an existing kanban task

**Positional Arguments:**
- `<id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--title` | `-t` | string | no | New title |
| `--description` | `-d` | string | no | New description |

---

### Close a kanban task

**Command:** `flowbot kanban delete <id> [flags]`
Close a task by ID

**Positional Arguments:**
- `<id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--yes` | `-y` | bool | no | Skip confirmation |

---

### Move a kanban task to another column

**Command:** `flowbot kanban move <id> --column <column> [flags]`
Move a task to a different column

**Positional Arguments:**
- `<id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--column` | `-c` | int | yes | Destination column ID |
| `--position` | `-p` | int | no | Position in column (0 = first) |
| `--project` | `-r` | int | no | Project ID |

---

### Add a card to a kanban board

**Command:** `flowbot kanban card add --title <title> [flags]`
Create a new task in the specified column

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--title` | `-t` | string | yes | Card title |
| `--description` | `-d` | string | no | Card description |
| `--project` | `-p` | int | no | Project ID |
| `--column` | `-c` | int | no | Column ID |

---

### Move a card to another column

**Command:** `flowbot kanban card move <card-id> --column <column> [flags]`
Move a task to a different column

**Positional Arguments:**
- `<card-id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--column` | `-c` | int | yes | Destination column ID |
| `--position` | `-p` | int | no | Position in column (0 = first) |
| `--project` | `-r` | int | no | Project ID |

---

### Delete a card from a kanban board

**Command:** `flowbot kanban card delete <card-id> [flags]`
Close a task by ID

**Positional Arguments:**
- `<card-id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--yes` | `-y` | bool | no | Skip confirmation |

---

### List columns in a project

**Command:** `flowbot kanban column list [flags]`
Display all columns in the specified project

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--project` | `-p` | int | no | Project ID |

---

### Get task metadata

**Command:** `flowbot kanban metadata get <task_id> [name]`
Get all metadata or a specific metadata value by name

**Positional Arguments:**
- `<task_id>`
- `<[name]>`

---

### Set task metadata

**Command:** `flowbot kanban metadata set <task_id> <name=value>...`
Set one or more metadata values for a task

**Positional Arguments:**
- `<task_id>`
- `<name=value...>`

---

### Delete task metadata

**Command:** `flowbot kanban metadata delete <task_id> <name> [flags]`
Delete a metadata entry from a task

**Positional Arguments:**
- `<task_id>`
- `<name>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--yes` | `-y` | bool | no | Skip confirmation |

---

### List all tags

**Command:** `flowbot kanban tag list [flags]`
Display kanban tags

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--project` | `-p` | int | no | Project ID (if specified, list tags for this project) |

---

### Create a new tag

**Command:** `flowbot kanban tag create --name <name> [flags]`
Add a new tag to the kanban board

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--name` | `-n` | string | yes | Tag name |
| `--project` | `-p` | int | no | Project ID |
| `--color` | `-c` | string | no | Color ID |

---

### Update a tag

**Command:** `flowbot kanban tag update <id> --name <name> [flags]`
Modify an existing tag

**Positional Arguments:**
- `<id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--name` | `-n` | string | yes | New tag name |
| `--color` | `-c` | string | no | Color ID |

---

### Delete a tag

**Command:** `flowbot kanban tag delete <id> [flags]`
Remove a tag by ID

**Positional Arguments:**
- `<id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--yes` | `-y` | bool | no | Skip confirmation |

---

### Get tags for a task

**Command:** `flowbot kanban tag task get <task_id>`
Display tags assigned to a task

**Positional Arguments:**
- `<task_id>`

---

### Set tags for a task

**Command:** `flowbot kanban tag task set <task_id> --project <project> --tags <tags>`
Assign tags to a task

**Positional Arguments:**
- `<task_id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--project` | `-p` | int | yes | Project ID |
| `--tags` | `-t` | string | yes | Tag names (can be specified multiple times) |

---

### List subtasks for a task

**Command:** `flowbot kanban subtask list <task_id>`
Display all subtasks for a given task

**Positional Arguments:**
- `<task_id>`

---

### Get a subtask by ID

**Command:** `flowbot kanban subtask get <task_id> <subtask_id>`
Display details of a specific subtask

**Positional Arguments:**
- `<task_id>`
- `<subtask_id>`

---

### Create a new subtask

**Command:** `flowbot kanban subtask create <task_id> --title <title> [flags]`
Add a subtask to a kanban task

**Positional Arguments:**
- `<task_id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--title` | `-t` | string | yes | Subtask title |
| `--user` | `-u` | int | no | User ID to assign |
| `--time-estimated` | `-e` | int | no | Estimated time (minutes) |
| `--time-spent` | `-s` | int | no | Time spent (minutes) |
| `--status` | `-S` | int | no | Status (0=Todo, 1=In progress, 2=Done) |

---

### Update a subtask

**Command:** `flowbot kanban subtask update <task_id> <subtask_id> [flags]`
Modify an existing subtask

**Positional Arguments:**
- `<task_id>`
- `<subtask_id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--title` | `-t` | string | no | New title |
| `--user` | `-u` | int | no | User ID to assign (-1 to unassign) |
| `--time-estimated` | `-e` | int | no | Estimated time (minutes, -1 to clear) |
| `--time-spent` | `-s` | int | no | Time spent (minutes, -1 to clear) |
| `--status` | `-S` | int | no | Status (0=Todo, 1=In progress, 2=Done, -1 to leave unchanged) |

---

### Delete a subtask

**Command:** `flowbot kanban subtask delete <task_id> <subtask_id> [flags]`
Remove a subtask by ID

**Positional Arguments:**
- `<task_id>`
- `<subtask_id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--yes` | `-y` | bool | no | Skip confirmation |

---

### Check if timer is active

**Command:** `flowbot kanban subtask timer check <task_id> <subtask_id> [flags]`
Check if a timer is started for the given subtask and user

**Positional Arguments:**
- `<task_id>`
- `<subtask_id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--user` | `-u` | int | no | User ID |

---

### Start subtask timer

**Command:** `flowbot kanban subtask timer start <task_id> <subtask_id> [flags]`
Start subtask timer for a user

**Positional Arguments:**
- `<task_id>`
- `<subtask_id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--user` | `-u` | int | no | User ID |

---

### Stop subtask timer

**Command:** `flowbot kanban subtask timer stop <task_id> <subtask_id> [flags]`
Stop subtask timer for a user

**Positional Arguments:**
- `<task_id>`
- `<subtask_id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--user` | `-u` | int | no | User ID |

---

### Get time spent

**Command:** `flowbot kanban subtask timer spent <task_id> <subtask_id> [flags]`
Get time spent on a subtask for a user (in hours)

**Positional Arguments:**
- `<task_id>`
- `<subtask_id>`

| Flag | Shorthand | Type | Required | Description |
|------|-----------|------|----------|-------------|
| `--user` | `-u` | int | no | User ID |

---

## Common Workflows

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
4. `Summarize task status, subtask completion, and suggest next actions.`


## Troubleshooting

- **"not logged in"**: Run `flowbot login` first.
- **"server URL is required"**: Set `FLOWBOT_SERVER_URL` env var or use `--server-url` flag.
- **Empty results**: Check the server is running and you have access to the requested resources.
