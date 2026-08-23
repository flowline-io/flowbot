---
name: confluence
description: >-
  Manage Confluence Cloud spaces and pages via flowbot confluence. Use when the user mentions confluence, atlassian, wiki, knowledge base, pages, spaces, cql.
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: confluence
  cli_root: confluence
---

# Confluence

Use `flowbot confluence` for capability `confluence`.
**CLI root is `confluence`** — do not invent `flowbot confluence` unless cli.md lists it as an alias.
Prefer the workflows below; load [references/cli.md](references/cli.md) only when you need a flag or subcommand not covered here.

**JSON fields:** Page and space ids are strings; page content uses Confluence storage XHTML.

## Setup

1. Ensure CLI auth: `flowbot login`
2. Set server via `FLOWBOT_SERVER_URL` or `--server-url`; optional `--profile`, `--debug` / `-d`
3. Prefer `-o json` when parsing results programmatically
4. Destructive commands often need `-y` / `--yes` in non-interactive sessions — check cli.md
5. Token scopes: `service:confluence:read` / `service:confluence:write`

## Workflows

### Create a wiki page

When a user wants to add a Confluence page:
1. `flowbot confluence space list`
2. `flowbot confluence page create --space-key <KEY> -t "<title>" -c "<p>content</p>"`

### Find and read a page

When a user wants to open Confluence content:
1. `flowbot confluence page search --cql "text ~ '<keywords>'"`
2. `flowbot confluence page content <page_id>`

## Troubleshooting

| Error | Fix |
|-------|-----|
| not logged in | `flowbot login` |
| server URL is required | set `FLOWBOT_SERVER_URL` or pass `--server-url` |
| permission denied / 403 | token missing service scopes (`service:confluence:read` / `service:confluence:write`) |
| hung waiting for confirm | pass `-y` when the command supports it (see cli.md) |
| empty results | provider not configured, wrong id/name, or empty dataset |
| unknown command | use `flowbot confluence`, not the capability id as the CLI verb |
