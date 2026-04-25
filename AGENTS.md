# Agents Guide for Flowbot

Multi-platform chatbot framework with 18 bot modules, workflow engine, and LLM agents.

**Generated:** 2026-04-25
**Commit:** 954313f9
**Branch:** master
**Go Version:** 1.26

## Quick Reference

| Task          | Location           | Notes                   |
| ------------- | ------------------ | ----------------------- |
| Add new bot   | `internal/bots/`   | See `AGENTS.md` there   |
| Database work | `internal/store/`  | DAO pattern, migrations |
| New provider  | `pkg/providers/`   | OAuth + API clients     |
| Core types    | `pkg/types/`       | Rulesets, protocol, KV  |
| API routes    | `internal/server/` | Fiber v3 handlers       |
| Entry points  | `cmd/`             | 4 binaries              |
| Frontend/PWA  | `pkg/page/`        | go-app WASM components  |
| Utilities     | `pkg/utils/`       | Must have unit tests    |

## Structure

```
flowbot/
├── cmd/                  # Entry points
│   ├── main.go          # HTTP server (Fiber)
│   ├── agent/           # Background agent daemon
│   ├── composer/        # CLI: code gen, migration
│   └── cli/             # CLI: admin commands
├── internal/
│   ├── bots/            # 18 bot modules
│   ├── server/          # Fiber v3 HTTP layer
│   ├── store/           # GORM DAO/models
│   └── platforms/       # Discord, Slack, Tailchat
├── pkg/
│   ├── types/           # Core type system
│   ├── providers/       # 17 third-party integrations
│   ├── page/            # PWA frontend (go-app/WASM)
│   ├── utils/           # Common utilities
│   ├── event/           # Redis Stream pub/sub
│   ├── executor/        # Workflow runtime (Docker)
│   ├── llm/             # LLM agent system
│   ├── chatbot/         # Platform chat interface
│   ├── migrate/         # Migration runner
│   └── ...              # config, flog, media, etc.
```

## Build Commands

```bash
go tool task default       # tidy → swagger → format → lint → test
go tool task build         # Main server
go tool task build:agent   # Agent daemon
go tool task test          # All tests
go tool task lint          # revive + actionlint
```

## Code Style

- **Format**: `go fmt` + `npx prettier`
- **Lint**: `revive` (strict, see `revive.toml`)
- **Imports**: stdlib → third-party → internal
- **Naming**: packages lowercase, types CamelCase
- **Errors**: Wrap with `%w`, use `errors.New` for sentinels

### Lint Rules (Key)

```toml
severity = "error"
enabled: blank-imports, dot-imports, error-naming, import-shadowing
```

## Key Patterns

### Bot Module

```go
type Bot struct{}
func (b *Bot) Info() types.BotInfo
func (b *Bot) Rules() []types.Rule
func (b *Bot) HandleEvent(evt types.Event) error
```

### Error Handling

```go
if err != nil {
    return fmt.Errorf("context: %w", err)
}
```

### Testing

```bash
go test ./pkg/utils
go test -run ^TestFoo$ ./pkg/utils
```

## Generated Code

| Type       | Command          | Location                      |
| ---------- | ---------------- | ----------------------------- |
| DAO        | `task dao`       | `internal/store/dao/*.gen.go` |
| Swagger    | `task swagger`   | `docs/api/`                   |
| Migrations | `task migration` | `pkg/migrate/migrations/`     |

**Never** edit `.gen.go` files directly.

## Testing

- Tests live next to code: `*_test.go`
- Use `require`/`assert` from testify
- Table-driven tests preferred
- Run with `gotestsum` (installed as Go tool)

## Anti-Patterns

- **Never** use `panic` outside initialization
- **Never** ignore errors (assign to `_` or handle)
- **Never** edit generated code directly
- **Never** block in event handlers
- **Always** check `err != nil` immediately
- **Always** wrap errors when propagating

## CI/Quality

```bash
go tool task link      # code lint
go tool task check     # lint + secure + leak + gosec
go tool task secure    # govulncheck
go tool task leak      # gitleaks
go tool task gosec     # security scan
```

## Configuration

- Runtime: `flowbot.yaml` (copy from `docs/config/config.yaml`)
- Build: `taskfile.yaml`
- Lint: `revive.toml`
- CI: `.github/workflows/build.yml`

## Notes

- Go 1.26+ required
- MySQL + Redis required
- Uses Fiber v3 for HTTP
- MCP protocol support per bot
- Do not use emojis
- You must run lint after modifying the code.
- Code comments, documentation, Git commit messages, and other text should all be written in English.
- The code in the utils directory must have unit tests.
