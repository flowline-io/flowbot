---
name: email
description: >-
  Send and read email via SMTP/IMAP with flowbot email. Use when the user mentions email, mail, smtp, imap, inbox, send, unread.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: email
  cli_root: email
---

# Email

Use `flowbot email` for capability `email`.
**CLI root is `email`** — do not invent `flowbot email` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**CLI limits:** No attachment download/upload in CLI; get returns text/html bodies and attachment metadata only.

**JSON fields:** Message ids are opaque strings from `list`/`search`; use `-o json` for next_cursor.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:email:read` / `service:email:write`

## Workflows

### Send an email

When a user wants to send a message:
1. `flowbot email send --to "user@example.com" --subject "Hello" --text "Body"`
2. Prefer --text for plain mail; use --html when HTML is required. Confirm recipients before send.

### Find and read messages

When a user asks to inspect inbox mail:
1. `flowbot email list --unseen-only`
2. `flowbot email get <id>`
3. Mark processed mail with `flowbot email mark-read <id>` when appropriate.

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:email:read` / `service:email:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot email`, not the capability id as the CLI verb |
