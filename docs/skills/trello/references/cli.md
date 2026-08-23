# Trello CLI reference

Capability `trello`. Root command: `flowbot trello`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### List Trello boards

`flowbot trello board [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page

### Get a Trello board

`flowbot trello board [board_id]`

### Create a Trello card

`flowbot trello card --list-id <list-id> --name <name> [flags]`

Flags: `--desc` (`-d`) string — Card description; `--list-id` string, required — Target list ID; `--name` (`-n`) string, required — Card title

### List cards on a board

`flowbot trello card [board_id] [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page

### Get a Trello card

`flowbot trello card [card_id]`

### Search Trello cards

`flowbot trello card --query <query> [flags]`

Flags: `--limit` int — Maximum results; `--query` (`-q`) string, required — Search query

### Update a Trello card

`flowbot trello card [card_id] [flags]`

Flags: `--desc` (`-d`) string — New description; `--name` (`-n`) string — New title

### Move a card to another list

`flowbot trello card [card_id] --list-id <list-id>`

Flags: `--list-id` string, required — Target list ID

### Delete a Trello card

`flowbot trello card [card_id]`

### Check Trello connectivity

`flowbot trello health`

### List lists on a board

`flowbot trello list [board_id]`

### Register a Trello board webhook

`flowbot trello webhook [flags]`

Flags: `--board-id` string — Board ID; `--callback-url` string — Callback URL; `--description` string — Webhook description

### Delete a Trello webhook

`flowbot trello webhook [webhook_id]`
