# Bot Modules Guide

18 specialized bot handlers implementing the `Bot` interface for different domains.

## Structure

Each bot module follows a consistent pattern:

```
bots/<name>/
├── bot.go              # Bot struct + interface implementation
├── command.go          # Slash commands (optional)
├── form.go             # Interactive forms (optional)
├── cron.go             # Scheduled tasks (optional)
├── event.go            # Event handlers (optional)
├── webhook.go          # HTTP webhooks (optional)
├── webservice.go       # HTTP handlers (optional)
├── page.go             # UI pages (optional)
├── tool.go             # MCP tools (optional)
├── instruct.go         # LLM instructions (optional)
├── setting.go          # Bot settings (optional)
├── collect.go          # Data collectors (optional)
├── *_test.go           # Tests for each component
└── static/             # Static assets (optional)
```

## Required Implementation

Every bot must implement in `bot.go`:

```go
type Bot struct {
    // Bot state
}

func (b *Bot) Info() types.BotInfo  // Bot metadata
func (b *Bot) Rules() []types.Rule  // Command/form handlers
func (b *Bot) HandleEvent(evt types.Event) error  // Event processing
```

## Bot Types

| Bot      | Domain     | Key Features                    |
| -------- | ---------- | ------------------------------- |
| agent    | LLM AI     | Multi-model, context management |
| workflow | Automation | DAG execution, 8+ actions       |
| finance  | Bills      | Firefly III integration         |
| kanban   | Tasks      | Kanboard sync                   |
| reader   | RSS        | Miniflux integration            |
| github   | Dev        | Issues, PRs                     |
| gitea    | Dev        | Repos, issues                   |
| dev      | Debug      | Testing utilities               |

## Registration

Bots auto-register via `modules.go` init pattern:

```go
func init() {
    Register(&Bot{})
}
```

## Testing

- Each component has `*_test.go` counterpart
- Use table-driven tests with `require`/`assert`
- Mock external dependencies

## Anti-Patterns

- **Never** import bot packages directly (use registration)
- **Never** block in event handlers (use goroutines for long ops)
- **Always** validate inputs before processing
- **Always** log errors with `flog.Error()`

## Commands

```bash
task generator:bot NAME=mybot RULE=command,form  # Generate new bot
go test ./internal/bots/dev/...                  # Test specific bot
```
