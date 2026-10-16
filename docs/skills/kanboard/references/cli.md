# Kanboard CLI reference

Capability `kanboard`. Root command: `flowbot kanban`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`, `--yes` / `-y`, `--no-input`. Every leaf accepts `-o table|json` (default `table`, omitted below).

## Output and exit codes

`-o json` writes a two-space indented object to stdout and a trailing newline: `{"data": ...}`. Paginated commands also include `page` (`limit`, `has_more`, and `next_cursor` when another page exists), including an empty page and the last page. Commands that are not paginated omit `page`. `data` is the API object: a list is an array, an id-only delete is `{"id":"..."}`, and an empty success is `{}`. The default stays `table`. A non-terminal stdin does not change the format; it implies `--no-input`. A pseudo-terminal still prompts.

Failures with `-o json` leave stdout empty and write `{"error":{"code","message","retryable","hint"}}` to stderr. Exit codes: 0 success, 1 `internal` / `run_failed` / `cancelled` / non-retryable `provider_error`, 2 `invalid_argument`, 3 `unauthenticated` or `permission_denied`, 4 `not_found`, 5 `conflict` or `already_exists`, 6 retryable transient (`rate_limited`, `unavailable`, `timeout`, retryable `provider_error`), 7 `confirmation_required`, 8 `not_implemented`. Retry the same call only on exit 6. `conflict` is exit 5 with `retryable: true`: re-read, then write. Pass `--yes` to confirm. `--no-input` without `--yes` exits 7 and does not read stdin. A declined prompt is exit 1, code `cancelled`. `--help` stays text. `pipeline export` and `workflow export` take `--file` for a YAML path; `-o` is only the format.

## Commands

### Add a card to a kanban board

`flowbot kanban card add --title <title> [flags]`

Create a new task in the specified column

Flags: `--category` int — Category ID; `--column` (`-c`) int — Column ID; `--description` string — Card description; `--due` string — Due date (YYYY-MM-DD, local midnight); `--project` (`-p`) int — Project ID; `--swimlane` int — Swimlane ID; `--title` (`-t`) string, required — Card title

### Delete a card from a kanban board

`flowbot kanban card delete <card-id>`

Close a task by ID

### Move a card to another column

`flowbot kanban card move <card-id> --column <column> [flags]`

Move a task to a different column

Flags: `--column` (`-c`) int, required — Destination column ID; `--position` (`-p`) int — Position in column (0 = first); `--project` (`-r`) int — Project ID; `--swimlane` int — Swimlane ID

### List categories in a project

`flowbot kanban category list [flags]`

Display all categories in the specified project

Flags: `--project` (`-p`) int — Project ID

### List columns in a project

`flowbot kanban column list [flags]`

Display all columns in the specified project

Flags: `--project` (`-p`) int — Project ID

### Create a new kanban task

`flowbot kanban create --title <title> [flags]`

Add a new task to the kanban board

Flags: `--category` int — Category ID; `--column` (`-c`) int — Column ID; `--description` string — Task description; `--due` string — Due date (YYYY-MM-DD, local midnight); `--project` (`-p`) int — Project ID; `--swimlane` int — Swimlane ID; `--title` (`-t`) string, required — Task title

### Close a kanban task

`flowbot kanban delete <id>`

Close a task by ID

### Get a kanban task by ID

`flowbot kanban get <id>`

Display details of a specific kanban task

### List all kanban tasks

`flowbot kanban list [flags]`

Display kanban tasks from Flowbot server

Flags: `--project` (`-p`) int — Project ID; `--status` (`-s`) string — Status filter (active, inactive, all)

### Delete task metadata

`flowbot kanban metadata delete <task_id> <name>`

Delete a metadata entry from a task

### Get task metadata

`flowbot kanban metadata get <task_id> [name]`

Get all metadata or a specific metadata value by name

### Set task metadata

`flowbot kanban metadata set <task_id> <name=value>...`

Set one or more metadata values for a task

### Move a kanban task to another column

`flowbot kanban move <id> --column <column> [flags]`

Move a task to a different column

Flags: `--column` (`-c`) int, required — Destination column ID; `--position` (`-p`) int — Position in column (0 = first); `--project` (`-r`) int — Project ID; `--swimlane` int — Swimlane ID

### Create a kanban project

`flowbot kanban project create --name <name> [flags]`

Create a new kanban project

Flags: `--description` string — Project description; `--name` (`-n`) string, required — Project name

### List kanban projects

`flowbot kanban project list`

Display all kanban projects

### Search kanban tasks

`flowbot kanban search <query> [flags]`

Search tasks using kanboard search syntax

Flags: `--project` (`-p`) int — Project ID

### Create a new subtask

`flowbot kanban subtask create <task_id> --title <title> [flags]`

Add a subtask to a kanban task

Flags: `--status` int — Status (0=Todo, 1=In progress, 2=Done); `--time-estimated` (`-e`) int — Estimated time (minutes); `--time-spent` (`-s`) int — Time spent (minutes); `--title` (`-t`) string, required — Subtask title; `--user` (`-u`) int — User ID to assign

### Delete a subtask

`flowbot kanban subtask delete <task_id> <subtask_id>`

Remove a subtask by ID

### Get a subtask by ID

`flowbot kanban subtask get <task_id> <subtask_id>`

Display details of a specific subtask

### List subtasks for a task

`flowbot kanban subtask list <task_id>`

Display all subtasks for a given task

### Check if timer is active

`flowbot kanban subtask timer check <task_id> <subtask_id> [flags]`

Check if a timer is started for the given subtask and user

Flags: `--user` (`-u`) int — User ID

### Get time spent

`flowbot kanban subtask timer spent <task_id> <subtask_id> [flags]`

Get time spent on a subtask for a user (in hours)

Flags: `--user` (`-u`) int — User ID

### Start subtask timer

`flowbot kanban subtask timer start <task_id> <subtask_id> [flags]`

Start subtask timer for a user

Flags: `--user` (`-u`) int — User ID

### Stop subtask timer

`flowbot kanban subtask timer stop <task_id> <subtask_id> [flags]`

Stop subtask timer for a user

Flags: `--user` (`-u`) int — User ID

### Update a subtask

`flowbot kanban subtask update <task_id> <subtask_id> [flags]`

Modify an existing subtask

Flags: `--status` int — Status (0=Todo, 1=In progress, 2=Done, -1 to leave unchanged); `--time-estimated` (`-e`) int — Estimated time (minutes, -1 to clear); `--time-spent` (`-s`) int — Time spent (minutes, -1 to clear); `--title` (`-t`) string — New title; `--user` (`-u`) int — User ID to assign (-1 to unassign)

### List swimlanes in a project

`flowbot kanban swimlane list [flags]`

Display all swimlanes in the specified project

Flags: `--project` (`-p`) int — Project ID

### Create a new tag

`flowbot kanban tag create --name <name> [flags]`

Add a new tag to the kanban board

Flags: `--color` (`-c`) string — Color ID; `--name` (`-n`) string, required — Tag name; `--project` (`-p`) int — Project ID

### Delete a tag

`flowbot kanban tag delete <id>`

Remove a tag by ID

### List all tags

`flowbot kanban tag list [flags]`

Display kanban tags

Flags: `--project` (`-p`) int — Project ID (if specified, list tags for this project)

### Get tags for a task

`flowbot kanban tag task get <task_id>`

Display tags assigned to a task

### Set tags for a task

`flowbot kanban tag task set <task_id> --project <project> --tags <tags>`

Assign tags to a task

Flags: `--project` (`-p`) int, required — Project ID; `--tags` (`-t`) stringSlice, required — Tag names (can be specified multiple times)

### Update a tag

`flowbot kanban tag update <id> --name <name> [flags]`

Modify an existing tag

Flags: `--color` (`-c`) string — Color ID; `--name` (`-n`) string, required — New tag name

### Update a kanban task

`flowbot kanban update <id> [flags]`

Modify an existing kanban task

Flags: `--category` int — Category ID; `--description` string — New description; `--due` string — Due date (YYYY-MM-DD, local midnight); `--swimlane` int — Swimlane ID; `--title` (`-t`) string — New title
