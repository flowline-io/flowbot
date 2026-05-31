# Types Package

Core type definitions: rulesets, message payloads, protocol types, KV helpers.

## Structure

```
types/
├── types.go           # Ruler interface, RulesetType
├── kv.go              # KV map with accessor methods (String, Int64, Map)
├── msg.go             # MsgPayload interface
├── errors.go          # Error types, NewError, constant sentinels (ErrNotFound, ...)
├── context.go, event.go, task.go, workflow.go, agent.go
├── protocol/          # Platform-agnostic types
│   ├── action.go      # Request/Response, error codes (10xxx–60xxx)
│   ├── message.go, event.go, user.go
└── ruleset/           # Rule implementations
    ├── command/ form/ webservice/
```

## Rules

- Prefer `KV` type over raw `map[string]any` for structured key-value access. `map[string]any` is acceptable in interface signatures (e.g. `ability.Invoke` params), protocol definitions, and generated code.
- Never define new message types outside this package
- Always implement `MsgPayload.Convert()` for new message types

## Commands

```bash
go test ./pkg/types/...
```
