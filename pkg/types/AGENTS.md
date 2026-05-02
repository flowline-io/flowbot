# Types Package

Core type definitions: rulesets, message payloads, protocol types, KV helpers.

## Structure

```
types/
├── types.go           # Ruler interface, RulesetType
├── kv.go              # KV map with accessor methods (String, Int64, Map)
├── msg.go             # MsgPayload interface
├── context.go, event.go, task.go, workflow.go, agent.go
├── protocol/          # Platform-agnostic types
│   ├── action.go      # Request/Response, error codes (10xxx–60xxx)
│   ├── message.go, event.go, user.go
└── ruleset/           # Rule implementations
    ├── command/ cron/ event/ form/ page/ webhook/ webservice/
```

## Rules

- Never use `map[string]any` directly — use `KV` type
- Never define new message types outside this package
- Always implement `MsgPayload.Convert()` for new message types

## Commands

```bash
go test ./pkg/types/...
```
