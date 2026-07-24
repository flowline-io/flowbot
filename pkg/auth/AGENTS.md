# Auth Package

AuthContext and scopes for Flowbot call paths.

## Subjects

`pkg/auth` subjects are `user` / `token` / `cron` / `pipeline` / `workflow` / `agent`.

REST / CLI / Chat are call paths, not subject types.

## Entry points

- `context.go` — AuthContext
- `scope.go` — scope constants and checks
- `token.go` — token helpers
