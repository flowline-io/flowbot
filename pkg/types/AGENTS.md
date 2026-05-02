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
└── ruleset/        # Rule implementations (7 types)
    ├── command/    # Slash commands
    ├── cron/       # Scheduled tasks
    ├── event/      # Event handlers
    ├── form/       # Interactive forms
    ├── page/       # UI pages
    ├── webhook/    # HTTP webhooks
    └── webservice/ # HTTP endpoints
```

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
