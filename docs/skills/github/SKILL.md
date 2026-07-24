---
name: github
description: >-
  Inspect GitHub users, repos, issues, notifications, releases, diffs, and files via flowbot github. Use when the user mentions github, repositories, issues, notifications, releases, commit diffs, source files.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: github
  cli_root: github
---

# GitHub

Use `flowbot github` for capability `github`.
**CLI root is `github`** — do not invent `flowbot github` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**CLI limits:** No pull-request commands and no `github health` CLI.

**JSON fields:** Issue argument is `<number>`; notification and release ids appear in `-o json`.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:github:read` / `service:github:write`

## Workflows

### Triage open issues

When a user wants to review GitHub issues:
1. `flowbot github issues <owner> -s open -n 10`
2. `flowbot github issue <owner> <repo> <number>`
3. Summarize the issue for the user.

### Check notifications and releases

When a user wants an activity overview:
1. `flowbot github notifications -n 20`
2. `flowbot github releases <owner> <repo> -n 5`
3. Present a concise summary.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:github:read` / `service:github:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot github`, not the capability id as the CLI verb |
