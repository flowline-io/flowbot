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

Use `flowbot forge` for capability `gitea`.
**CLI root is `forge`** — do not invent `flowbot gitea` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**CLI limits:** CLI root is `forge` (alias may include gitea). No `forge health` command.

**JSON fields:** Issue index is the forge issue number argument; not the same as GitHub `number` naming in docs.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:gitea:read` / `service:gitea:write`

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
| permission denied / 403 | token missing service scopes (`service:gitea:read` / `service:gitea:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot forge`, not the capability id as the CLI verb |
