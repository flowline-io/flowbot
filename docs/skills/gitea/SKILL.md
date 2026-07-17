---
name: gitea
description: >-
  Inspect forge users, repos, issues, diffs, and files via flowbot forge. Use when the user mentions gitea, forge, repositories, issues, commit diffs, source files, code review.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: gitea
  cli_root: forge
---

# Gitea

Use `flowbot forge` for capability `gitea`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

## Workflows

### Inspect a repository issue

When a user asks about a forge issue:
1. `flowbot forge issues <owner> -s open -n 10`
2. `flowbot forge issue <owner> <repo> <index>`
3. Summarize the issue for the user.

### Review a commit change

When a user wants to inspect a commit:
1. `flowbot forge diff <owner> <repo> <commit-id>`
2. `flowbot forge file <owner> <repo> <commit-id> <file-path>`
3. Explain the relevant changes.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| empty results | confirm server health and capability access scopes |
