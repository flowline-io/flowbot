# Confluence CLI reference

Capability `confluence`. Root command: `flowbot confluence`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Check Confluence connectivity

`flowbot confluence health`

### List pages in a space

`flowbot confluence page [space_key] [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page

### Get a Confluence page

`flowbot confluence page [page_id]`

### Get page storage content

`flowbot confluence page [page_id]`

### Search pages with CQL

`flowbot confluence page --cql <cql> [flags]`

Flags: `--cql` string, required — CQL query; `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page

### Create a Confluence page

`flowbot confluence page --title <title> [flags]`

Flags: `--content` (`-c`) string — Storage-format XHTML content; `--space-key` string — Space key; `--title` (`-t`) string, required — Page title

### Update a Confluence page

`flowbot confluence page [page_id] [flags]`

Flags: `--content` (`-c`) string — Storage-format XHTML content; `--title` (`-t`) string — New title

### Delete a Confluence page

`flowbot confluence page [page_id]`

### List Confluence spaces

`flowbot confluence space [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Maximum items per page
