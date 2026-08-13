# Auth Package

AuthContext and scopes for Flowbot call paths.

## Subjects

`pkg/auth` subjects are `user` / `token` / `cron` / `pipeline` / `workflow` / `agent`.

## Call paths

AuthContext spans REST, CLI, Chat, Webhook, Cron, Pipeline, and Workflow. REST / CLI / Chat / Webhook are call paths, not subject types. Cron / Pipeline / Workflow appear as both a subject and a call path.

## Entry points

- `context.go` — AuthContext
- `scope.go` — scope constants and checks
- `token.go` — token helpers
