# Types Package Guide

Core type definitions for Flowbot: rulesets, message payloads, protocol types, and KV helpers.

## Structure

```
types/
├── types.go        # Ruler interface, RulesetType enum
├── kv.go           # KV map type with accessor methods
├── msg.go          # MsgPayload interface, message types
├── context.go      # Context helpers
├── event.go        # Event types
├── task.go         # Task definitions
├── workflow.go     # Workflow types
├── agent.go        # Agent types
├── protocol/       # Platform protocol types
│   ├── action.go   # Request/Response, error codes
│   ├── message.go  # Message types
│   ├── event.go    # Platform events
│   └── user.go     # User info
└── ruleset/        # Rule implementations (14 types)
    ├── command/    # Slash commands
    ├── form/       # Interactive forms
    ├── cron/       # Scheduled tasks
    ├── event/      # Event handlers
    ├── webhook/    # HTTP webhooks
    ├── webservice/ # HTTP endpoints
    ├── tool/       # Tools
    ├── instruct/   # LLM instructions
    ├── page/       # UI pages
    ├── setting/    # Bot settings
    ├── collect/    # Data collectors
    └── ...
```

## Ruleset Types

14 rule types defined in `types.go`:

| Type           | Purpose           |
| -------------- | ----------------- |
| ActionRule     | Generic actions   |
| CommandRule    | Slash commands    |
| FormRule       | Interactive forms |
| CronRule       | Scheduled tasks   |
| EventRule      | Event handlers    |
| WebhookRule    | HTTP webhooks     |
| WebserviceRule | HTTP endpoints    |
| ToolRule       | Tools             |
| InstructRule   | LLM instructions  |
| PageRule       | UI pages          |
| SettingRule    | Bot configuration |
| CollectRule    | Data collection   |
| TriggerRule    | Workflow triggers |
| WorkflowRule   | Workflow actions  |

## Message Types

All implement `MsgPayload` interface:

| Type        | Usage             |
| ----------- | ----------------- |
| TextMsg     | Plain text        |
| FormMsg     | Interactive forms |
| LinkMsg     | URL previews      |
| TableMsg    | Tabular data      |
| InfoMsg     | Info display      |
| ChartMsg    | Chart data        |
| MarkdownMsg | Markdown content  |
| HtmlMsg     | Raw HTML          |
| InstructMsg | LLM instructions  |
| KVMsg       | Key-value data    |
| EmptyMsg    | No content        |

## KV Type

`type KV map[string]any` with helpers:

```go
kv := types.KV{"key": "value"}
str, ok := kv.String("key")
num, ok := kv.Int64("count")
m, ok := kv.Map("nested")
```

## Protocol

`pkg/types/protocol/` defines platform-agnostic types:

- `Request` / `Response` - Action request/response
- Error codes (10xxx request, 20xxx handler, 30xxx execution, 60xxx business)
- `Driver` interface for platform adapters

## Anti-Patterns

- **Never** use `map[string]any` directly — use `KV` type
- **Never** define new message types outside this package
- **Always** implement `MsgPayload.Convert()` for new message types

## Commands

```bash
go test ./pkg/types/...   # Test all types
```
