# Types Package

Core type definitions: rulesets, message payloads, protocol types, KV helpers.

## Structure

```
types/
├── types.go           # Ruler interface, RulesetType
├── kv.go              # KV map with accessor methods (String, Int64, Map)
├── msg.go             # MsgPayload interface
├── errors.go          # Error types, NewError, constant sentinels (ErrNotFound, ...)
├── context.go         # Context type with metadata fields
├── event.go           # DataEvent type definitions
├── task.go            # Task type definitions
├── workflow.go        # Workflow type definitions
├── agent.go           # Agent type definitions
├── uid.go             # Unique ID generation
├── id.go              # Compact ID encoding/decoding
├── filter.go          # Event filter types
├── filter_cache.go    # In-memory filter cache for source/event type dropdowns
├── file.go            # File type definitions
├── pipeline_stats.go  # Pipeline statistics types
├── protocol/          # Platform-agnostic types
│   ├── action.go      # Request/Response, error codes (10xxx–60xxx)
│   ├── message.go     # Message type definition
│   ├── event.go       # Event type definition
│   ├── user.go        # User type definition
│   ├── command.go     # Command type definition
│   └── platform.go    # Driver, Adapter, Action interfaces
├── ruleset/           # Rule implementations
│   ├── command/       # Command rule types
│   ├── form/          # Form rule types
│   └── webservice/    # Webservice rule types
├── audit/             # Audit trail types
└── model/             # AI model type definitions
```

## Rules

- Prefer `KV` type over raw `map[string]any` for structured key-value access. `map[string]any` is acceptable in interface signatures (e.g. `ability.Invoke` params), protocol definitions, and generated code.
- Never define new message types outside this package
- Always implement `MsgPayload.Convert()` for new message types

## Commands

```bash
go test ./pkg/types/...
```
