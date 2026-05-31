# Notify Guide

Multi-channel notification gateway with rule evaluation, template rendering, and provider dispatch.

## Architecture

```
                      Callers
      ┌────────────┬───────┼───────────────┐
      │ Modules    │ Agent │ Pipelines/Cron │
      │ (hub/cmd)  │ Ops   │ (via ability)  │
      └─────┬──────┴───┬───┴──────┬────────┘
            │          │          │
    notify.ChannelSend │  ability/notify/send.go
            │          │          │
            └──────────┼──────────┘
                       │
               notify.GatewaySend()
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
    Rules Engine   Template     Channel Send
    (throttle/     Engine       (via Notifyer
     aggregate/    (Sprig)       registry)
     mute/drop)
          │                        │
    Redis Store              ┌─────┼─────┐
    (rate counters,          Slack ntfy  Pushover  MessagePusher
     aggregate lists)
```

## Structure

```
pkg/notify/
├── types.go             # Notifyer interface, Message, Priority types
├── notify.go            # Core engine: Register, Send, GatewaySend
├── notify_test.go       # Unit tests for core package
├── AGENTS.md            # This file
├── template/            # Template rendering engine (Go text/template + Sprig)
│   ├── engine.go        # Template compile + Render with per-channel overrides
│   ├── loader.go        # Global singleton Init() / GetEngine()
│   └── engine_test.go   # Template engine tests
├── rules/               # Notification rule engine
│   ├── engine.go        # Rule evaluation, pattern matching, time conditions
│   ├── throttle.go      # Redis-based rate limiting
│   ├── aggregate.go     # Redis List-based aggregation buffers
│   ├── worker.go        # Background worker for flushing expired aggregates
│   └── engine_test.go   # Rule engine tests
└── <provider>/          # One directory per notification channel
    ├── provider.go      # Notifyer implementation
    └── provider_test.go # Provider tests (table-driven, httptest mock)
```

## Notifyer Interface

```go
type Notifyer interface {
    Protocol() string                // protocol scheme (e.g. "slack", "ntfy")
    Templates() []string             // URI templates with {placeholders}
    Send(tokens types.KV, message Message) error  // dispatch a notification
}
```

See `types.go` for the full interface definition.

## How to Add a New Provider

1. Create `pkg/notify/<name>/provider.go`
2. Implement the `Notifyer` interface on a package-level `plugin struct{}`
3. Add a `Register()` function that calls `notify.Register(ID, &handler)`
4. Define URI templates in `Templates()` using `{placeholder}` syntax
5. Implement `Send()` with an internal `doSend(tokens, msg, client, baseURL)` for testability:

   ```go
   func (*plugin) Send(tokens types.KV, message notify.Message) error {
       return doSend(tokens, message, resty.New(), defaultBaseURL)
   }

   func doSend(tokens types.KV, message notify.Message, client *resty.Client, baseURL string) error {
       // actual HTTP implementation
   }
   ```

6. Register the provider in `internal/server/notify.go` via `fx.Invoke`
7. Write tests in `provider_test.go` using `httptest.NewServer`

## Provider Conventions

- Package name matches directory name (lowercase, no hyphens or underscores)
- File name: `provider.go` for logic, `provider_test.go` for tests
- Use `resty.dev/v3` for HTTP client
- Error wrapping: `fmt.Errorf("<protocol>: %w", err)` or `fmt.Errorf("<protocol>: non-200 response %d", ...)`
- Log success at `flog.Debug` or `flog.Info` level, failures at `flog.Error` level
- ID constant uses the protocol scheme string (may contain hyphens for backward compatibility)
- The `doSend` pattern enables testing via HTTP mock server injection

## Template Engine

- Compiles `config.NotifyTemplate` entries using Go `text/template` + Sprig functions
- Supports per-channel template overrides (e.g. `telegram` gets HTML, `slack` gets Markdown)
- Title auto-extracted from first line of body (strips Markdown headings)
- Custom functions: `eventTime(t)`, `shorten(s, maxLen)`
- See `template/` sub-package for full details

## Rule Engine

- Evaluates `config.NotifyRule` entries in priority order (highest first)
- Pattern matching: `*` (all), exact match, prefix (`infra.*`), suffix (`*.created`)
- Time conditions: `time.hour >= 23`, with `||` and `&&` boolean operators
- Actions: `drop`, `mute`, `throttle` (Redis INCR+TTL), `aggregate` (Redis List buffer + timer)
- See `rules/` sub-package for full details

## Testing

- Provider tests: `httptest.NewServer` + `doSend` injection, table-driven with `for _, tt := range tests { t.Run(tt.name, ...) }`
- Core tests: `ParseSchema`, `ParseTemplate`, `Priority` constants, `Message` zero values
- Template engine: compile + render with various inputs and channel overrides
- Rule engine: pattern matching, rule evaluation, time condition parsing
- BDD: `tests/specs/notify_spec_test.go` (Ginkgo v2 + Gomega)

## Anti-Patterns

- Never import `pkg/providers/*` from a notify provider implementation
- Never call hub/pipeline/emit DataEvent from inside a provider's `Send()`
- Never hardcode credentials in provider code — read from `tokens types.KV`
- Never use `encoding/json` — use `sonic` or `resty` JSON helpers
- Never return 500 for all errors — distinguish connection errors from downstream API errors
- Never block event handlers — `Send()` should be non-blocking
