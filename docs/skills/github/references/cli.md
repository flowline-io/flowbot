# GitHub CLI reference

Capability `github`. Root command: `flowbot github`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Get GitHub commit diff

`flowbot github diff <owner> <repo> <commit-id>`

Display the diff for a specific GitHub commit

### Get GitHub file content

`flowbot github file <owner> <repo> <commit-id> <file-path> [flags]`

Display the content of a file at a specific GitHub commit

Flags: `--line-count` int — Number of lines to return; `--line-start` int — Starting line number

### Get a GitHub issue

`flowbot github issue <owner> <repo> <number>`

Display a single GitHub issue by owner, repo, and issue number

### List GitHub issues

`flowbot github issues <owner> [flags]`

List issues for an owner from GitHub

Flags: `--cursor` string — Pagination cursor; `--limit` (`-n`) int — Maximum number of issues; `--state` (`-s`) string — Issue state filter (open, closed)

### List GitHub notifications

`flowbot github notifications [flags]`

Display the authenticated user's GitHub notifications

Flags: `--cursor` string — Pagination cursor; `--limit` (`-n`) int — Maximum number of notifications

### List GitHub releases

`flowbot github releases <owner> <repo> [flags]`

Display releases for a GitHub repository

Flags: `--cursor` string — Pagination cursor; `--limit` (`-n`) int — Maximum number of releases

### Get a GitHub repository

`flowbot github repo <owner> <repo>`

Display repository details from GitHub

### Get authenticated GitHub user

`flowbot github user`

Display the authenticated GitHub user

### Get GitHub user by login

`flowbot github user-by-login <login>`

Display a GitHub user by their login name
