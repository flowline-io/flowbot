---
name: github
description: >-
  Inspect GitHub users, repos, issues, notifications, releases, diffs, and files via flowbot github. Use when the user mentions github, repositories, issues, notifications, releases, pull requests, commit diffs.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: github
  cli_root: github
---

# GitHub

Use `flowbot github` for capability `github`. Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically

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
| empty results | confirm server health and capability access scopes |
