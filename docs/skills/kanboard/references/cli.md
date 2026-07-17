# Kanboard CLI reference

Capability `kanboard`. Root command: `flowbot kanban`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Add a card to a kanban board

`flowbot kanban card add --title <title> [flags]`

Create a new task in the specified column

Flags: `--column` (`-c`) int — Column ID; `--description` (`-d`) string — Card description; `--project` (`-p`) int — Project ID; `--title` (`-t`) string, required — Card title

### Delete a card from a kanban board

`flowbot kanban card delete <card-id> [flags]`

Close a task by ID

Flags: `--yes` (`-y`) bool — Skip confirmation

### Move a card to another column

`flowbot kanban card move <card-id> --column <column> [flags]`

Move a task to a different column

Flags: `--column` (`-c`) int, required — Destination column ID; `--position` (`-p`) int — Position in column (0 = first); `--project` (`-r`) int — Project ID

### List columns in a project

`flowbot kanban column list [flags]`

Display all columns in the specified project

Flags: `--project` (`-p`) int — Project ID

### Create a new kanban task

`flowbot kanban create --title <title> [flags]`

Add a new task to the kanban board

Flags: `--column` (`-c`) int — Column ID; `--description` (`-d`) string — Task description; `--project` (`-p`) int — Project ID; `--title` (`-t`) string, required — Task title

### Close a kanban task

`flowbot kanban delete <id> [flags]`

Close a task by ID

Flags: `--yes` (`-y`) bool — Skip confirmation

### Get a kanban task by ID

`flowbot kanban get <id>`

Display details of a specific kanban task

### List all kanban tasks

`flowbot kanban list [flags]`

Display kanban tasks from Flowbot server

Flags: `--project` (`-p`) int — Project ID; `--status` (`-s`) string — Status filter (active, inactive, all)

### Delete task metadata

`flowbot kanban metadata delete <task_id> <name> [flags]`

Delete a metadata entry from a task

Flags: `--yes` (`-y`) bool — Skip confirmation

### Get task metadata

`flowbot kanban metadata get <task_id> [name]`

Get all metadata or a specific metadata value by name

### Set task metadata

`flowbot kanban metadata set <task_id> <name=value>...`

Set one or more metadata values for a task

### Move a kanban task to another column

`flowbot kanban move <id> --column <column> [flags]`

Move a task to a different column

Flags: `--column` (`-c`) int, required — Destination column ID; `--position` (`-p`) int — Position in column (0 = first); `--project` (`-r`) int — Project ID

### Search kanban tasks

`flowbot kanban search <query> [flags]`

Search tasks using kanboard search syntax

Flags: `--project` (`-p`) int — Project ID

### Create a new subtask

`flowbot kanban subtask create <task_id> --title <title> [flags]`

Add a subtask to a kanban task

Flags: `--status` int — Status (0=Todo, 1=In progress, 2=Done); `--time-estimated` (`-e`) int — Estimated time (minutes); `--time-spent` (`-s`) int — Time spent (minutes); `--title` (`-t`) string, required — Subtask title; `--user` (`-u`) int — User ID to assign

### Delete a subtask

`flowbot kanban subtask delete <task_id> <subtask_id> [flags]`

Remove a subtask by ID

Flags: `--yes` (`-y`) bool — Skip confirmation

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

### Create a new tag

`flowbot kanban tag create --name <name> [flags]`

Add a new tag to the kanban board

Flags: `--color` (`-c`) string — Color ID; `--name` (`-n`) string, required — Tag name; `--project` (`-p`) int — Project ID

### Delete a tag

`flowbot kanban tag delete <id> [flags]`

Remove a tag by ID

Flags: `--yes` (`-y`) bool — Skip confirmation

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

Flags: `--description` (`-d`) string — New description; `--title` (`-t`) string — New title
