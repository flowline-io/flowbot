# Email CLI reference

Capability `email`. Root command: `flowbot email`.

Global flags: `--server-url`, `--profile`, `--debug` / `-d`. Most commands accept `-o table|json` (omitted below).

## Commands

### Get a message by id

`flowbot email get <id>`

### Check email backend health

`flowbot email health`

### List messages

`flowbot email list [flags]`

Flags: `--cursor` string — Pagination cursor; `--limit` int — Page size; `--mailbox` string — Mailbox name; `--unseen-only` bool — Only unseen messages

### Mark a message as read

`flowbot email mark-read <id>`

### Mark a message as unread

`flowbot email mark-unread <id>`

### Search messages

`flowbot email search [flags]`

Flags: `--before` string — RFC3339 before; `--cursor` string — Pagination cursor; `--from` string — From filter; `--limit` int — Page size; `--mailbox` string — Mailbox name; `--since` string — RFC3339 since; `--subject` string — Subject filter; `--to` string — To filter; `--unseen` bool — Unseen only

### Send an email

`flowbot email send --subject <subject> --to <to> [flags]`

Flags: `--bcc` stringSlice — Bcc addresses; `--cc` stringSlice — Cc addresses; `--from-name` string — From display name; `--html` string — HTML body; `--subject` string, required — Subject; `--text` string — Plain text body; `--to` stringSlice, required — Recipient addresses
