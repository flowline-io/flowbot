---
name: memos
description: >-
  Create, list, update, and delete memos via flowbot memo. Use when the user mentions memos, memo notes, scratchpad, quick notes, jotting.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: memos
  cli_root: memo
---

# Memos

Use `flowbot memo` for capability `memos`.
**CLI root is `memo`** — do not invent `flowbot memos` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**JSON fields:** Memo resource name (not a numeric id) is the get/delete argument; see `name` in `-o json`.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:memos:read` / `service:memos:write`

## Workflows

### Capture a quick note

When a user wants to save a short memo:
1. `flowbot memo create -c "<content>"`
2. Report back with the memo name.

### Review recent memos

When a user wants to browse or open memos:
1. `flowbot memo list --limit 20`
2. `flowbot memo get <name>`
3. Present the memo content to the user.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:memos:read` / `service:memos:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot memo`, not the capability id as the CLI verb |
