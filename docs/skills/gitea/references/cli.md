# Gitea CLI reference

Capability `gitea`. Root command: `flowbot forge`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Get commit diff

`flowbot forge diff <owner> <repo> <commit-id>`

Display the diff for a specific commit from the forge

### Get file content

`flowbot forge file <owner> <repo> <commit-id> <file-path> [flags]`

Display the content of a file at a specific commit from the forge

Flags: `--line-count` int — Number of lines to return; `--line-start` int — Starting line number

### Get an issue

`flowbot forge issue <owner> <repo> <index>`

Display a single issue from the forge by owner, repo, and index

### List issues

`flowbot forge issues <owner> [flags]`

List issues for an owner from the forge

Flags: `--cursor` string — Pagination cursor; `--limit` (`-n`) int — Maximum number of issues; `--state` (`-s`) string — Issue state filter (open, closed)

### Get a repository

`flowbot forge repo <owner> <repo>`

Display repository details from the forge

### Get authenticated forge user

`flowbot forge user`

Display the authenticated user from the configured forge
