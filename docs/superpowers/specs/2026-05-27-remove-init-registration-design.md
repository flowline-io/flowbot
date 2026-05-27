# Remove init() Auto-Registration for Testability

## Motivation

Seven files use `func init()` to auto-register providers into global registry maps at import time. This makes testing fragile: simply importing a package for a utility function triggers side effects in the registry. Tests cannot selectively enable or disable providers, and registration order is implicit.

The project already has an established pattern for controlled registration — `export Register() + fx.Invoke` — used by modules (`internal/modules/fx.go`), notify providers (`internal/server/notify.go`), and media handlers (`internal/server/media.go`). The `init()` variants are the exception, not the rule.

## Design Overview

Convert all seven `init()` auto-registration sites to explicit `Register()` functions wired via `fx.Invoke`. No changes to registry maps, provider APIs, or consumer code. Tests gain full control over which providers are registered.

## Architecture

```
Before (import triggers side effect):
  import _ "flowbot/pkg/llm"     // gemini, openai, anthropic
                                  // all auto-registered via init()

After (explicit wiring):
  fx.Invoke(                     // in internal/server/llm.go
      llm.RegisterGemini,
      llm.RegisterOpenAI,
      llm.RegisterAnthropic,
  )

Test control:
  func TestXxx(t *testing.T) {
      llm.RegisterOpenAI()  // only what this test needs
  }
```

Three wiring files added under `internal/server/`, mirroring existing `notify.go`, `media.go`, `modules/fx.go`.

## Files & Changes

### LLM Providers

| File                     | Change                                     |
| ------------------------ | ------------------------------------------ |
| `pkg/llm/gemini.go`     | Remove `init()`, export `RegisterGemini()` |
| `pkg/llm/openai.go`     | Remove `init()`, export `RegisterOpenAI()` |
| `pkg/llm/anthropic.go`  | Remove `init()`, export `RegisterAnthropic()` |
| `internal/server/llm.go` | New: `fx.Invoke(llm.RegisterGemini, llm.RegisterOpenAI, llm.RegisterAnthropic)` |

### OAuth Providers

| File                              | Change                                                 |
| --------------------------------- | ------------------------------------------------------ |
| `pkg/providers/github/github.go` | Remove `init()`, export `Register()`                   |
| `pkg/providers/slack/slack.go`   | Remove `init()`, export `Register()`                   |
| `pkg/providers/dropbox/dropbox.go` | Remove `init()`, export `Register()`                 |
| `internal/server/providers.go`   | New: `fx.Invoke(github.Register, slack.Register, dropbox.Register)` |

### Reexec Handler

| File                                     | Change                               |
| ---------------------------------------- | ------------------------------------ |
| `pkg/executor/runtime/shell/shell.go`   | Remove `init()`, export `Register()` |
| `internal/server/reexec.go`             | New: `fx.Invoke(shell.Register)`     |

### Entry Point Wiring

| File                         | Change                                              |
| ---------------------------- | --------------------------------------------------- |
| `internal/server/server.go`  | Add `LLMModules`, `OAuthModules`, `ReexecModules` to fx App options |

### Unchanged

| Component                                    | Reason                                       |
| -------------------------------------------- | -------------------------------------------- |
| `pkg/llm/provider.go` (`register()` helper)  | Internal helper, no API change               |
| `pkg/providers/providers.go` (registry map)  | Global map stays, only call site changes     |
| `pkg/utils/reexec/rexec.go` (registry map)   | Same as above                                |
| All `GetClient()` / `GetConfig()` functions  | Provider clients unchanged                   |
| Provider interface types                     | No interface changes                         |

### Tests

| Pattern                        | Before                                             | After                                          |
| ------------------------------ | -------------------------------------------------- | ---------------------------------------------- |
| Import triggers registration   | `import "pkg/llm"` auto-registers all three        | Import has no side effect                      |
| Selective per-test registration| Not possible                                       | `llm.RegisterOpenAI()` in test body            |
| Test isolation                 | Order-dependent, tests leak registrations to others| Each test controls its own registrations       |

Existing tests that depend on `init()` side effects need one-line additions (`llm.RegisterXXX()` in `TestMain` or test bodies).

## Error Handling

`Register()` functions are void (matching the project convention: `module.Register`, `notify.Register`). Duplicate registration panics (existing behavior preserved). No new error paths.

## Risk Assessment

| Risk                                    | Mitigation                                            |
| --------------------------------------- | ----------------------------------------------------- |
| Missing `Register()` call in wiring     | Server startup panics on first use, caught in dev/CI  |
| Forgetting `Register()` in existing test| Test fails with "unknown provider", easy to diagnose  |
| Concurrent registration in tests        | Registry RWMutex already in place, no change needed   |
| Auto-generated files using init()       | `internal/store/ent/gen/` and `docs/api/docs.go` not in scope |
