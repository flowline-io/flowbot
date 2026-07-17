# Memos CLI reference

Capability `memos`. Root command: `flowbot memo`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Create a new memo

`flowbot memo create --content <content> [flags]`

Add a new memo to the Flowbot server

Flags: `--content` (`-c`) string, required — Memo content; `--visibility` (`-v`) string — Memo visibility (PRIVATE, PROTECTED, PUBLIC)

### Delete a memo

`flowbot memo delete <name> [flags]`

Delete a memo by its resource name

Flags: `--yes` (`-y`) bool — Skip confirmation

### Get a memo by resource name

`flowbot memo get <name>`

Display details of a specific memo

### Check memo backend health

`flowbot memo health`

Check whether the memo backend is reachable

### List all memos

`flowbot memo list [flags]`

Display memos from the Flowbot server

Flags: `--limit` (`-n`) int — Maximum number of memos

### Update a memo

`flowbot memo update <name> [flags]`

Update content, visibility, or pinned status of a memo

Flags: `--content` (`-c`) string — New content; `--pinned` (`-p`) bool — Set pinned status; `--visibility` (`-v`) string — New visibility (PRIVATE, PROTECTED, PUBLIC)
