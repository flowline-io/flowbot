# Client i18n templates show `<no value>` in chat agent duration labels

## Decision

`ClientMessages` must export raw `{{.Var}}` templates for client-side interpolation. Use `TClient` (reads `Other` from embedded TOML) instead of `T` (go-template execute without data).

## Verification

```bash
go test ./pkg/i18n -run TestClientJSONPreservesDurationTemplates -count=1
```
