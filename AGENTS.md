# Agents Guide for Flowbot

Multi-platform chatbot framework with 18 bot modules, workflow engine, and LLM agents.

## Quick Reference

| Task | Location | Notes |
|------|----------|-------|
| Add new bot | `internal/bots/` | See `AGENTS.md` there |
| Database work | `internal/store/` | DAO pattern, migrations |
| New provider | `pkg/providers/` | OAuth + API clients |
| API routes | `internal/server/` | Fiber v3 handlers |
| Entry points | `cmd/` | 4 binaries |

## Structure

```
flowbot/
├── cmd/                  # Entry points
│   ├── main.go          # Server
│   ├── agent/           # Background agent
│   ├── app/             # Admin PWA
│   └── composer/        # CLI tool
├── internal/
│   ├── bots/            # 18 bot modules
│   ├── store/           # Database layer
│   ├── server/          # HTTP server
│   └── platforms/       # Discord, Slack, Tailchat
├── pkg/
│   ├── providers/       # 17 third-party APIs
│   ├── utils/           # Shared utilities
│   └── types/           # Common types
└── app/                 # WebAssembly admin UI
```

## Build Commands

```bash
task default       # tidy → swagger → format → lint → test
task build         # Main server
task build:agent   # Agent daemon
task build:app     # Admin PWA
task test          # All tests
task lint          # revive + actionlint
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

| Type | Command | Location |
|------|---------|----------|
| DAO | `task dao` | `internal/store/dao/*.gen.go` |
| Swagger | `task swagger` | `docs/api/` |
| Migrations | `task migration` | `internal/store/migrate/` |

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
task check     # lint + secure + leak + gosec
task secure    # govulncheck
task leak      # gitleaks
task gosec     # security scan
```

## Configuration

- Runtime: `flowbot.yaml` (copy from `docs/config/config.yaml`)
- Build: `taskfile.yaml`
- Lint: `revive.toml`
- CI: `.github/workflows/build.yml`

## Notes

- Go 1.24+ required
- MySQL + Redis required
- Uses Fiber v3 for HTTP
- WebAssembly for admin PWA
- MCP protocol support per bot
