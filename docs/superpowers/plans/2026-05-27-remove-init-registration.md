# Remove init() Auto-Registration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert 7 `init()` auto-registration sites to explicit `Register()` functions wired via `fx.Invoke`, making provider registration testable and controllable.

**Architecture:** Three wiring files (`llm.go`, `providers.go`, `reexec.go`) added under `internal/server/` declare `fx.Options` with `fx.Invoke(...)` to explicitly register providers at server startup. Each provider file removes `init()` and exports a `Register()` function. Existing registry maps (`pkg/llm/provider.go`, `pkg/providers/providers.go`, `pkg/utils/reexec/rexec.go`) are unchanged.

**Tech Stack:** uber-go/fx

---

### Task 1: LLM Providers — Remove init(), export Register functions

**Files:**

- Modify: `pkg/llm/gemini.go`
- Modify: `pkg/llm/openai.go`
- Modify: `pkg/llm/anthropic.go`

- [ ] **Step 1: Edit gemini.go — replace init() with RegisterGemini()**

Replace lines 18-20 in `pkg/llm/gemini.go`:

```go
func init() {
	register(ProviderGemini, newGeminiProvider)
}
```

with:

```go
// RegisterGemini registers the Gemini LLM provider in the global provider registry.
func RegisterGemini() {
	register(ProviderGemini, newGeminiProvider)
}
```

- [ ] **Step 2: Edit openai.go — replace init() with RegisterOpenAI()**

Replace lines 18-21 in `pkg/llm/openai.go`:

```go
func init() {
	register(ProviderOpenAI, newOpenAIProvider)
	register(ProviderOpenAICompatible, newOpenAIProvider)
}
```

with:

```go
// RegisterOpenAI registers the OpenAI and OpenAI-compatible LLM providers.
func RegisterOpenAI() {
	register(ProviderOpenAI, newOpenAIProvider)
	register(ProviderOpenAICompatible, newOpenAIProvider)
}
```

- [ ] **Step 3: Edit anthropic.go — replace init() with RegisterAnthropic()**

Replace lines 18-20 in `pkg/llm/anthropic.go`:

```go
func init() {
	register(ProviderAnthropic, newAnthropicProvider)
}
```

with:

```go
// RegisterAnthropic registers the Anthropic LLM provider in the global provider registry.
func RegisterAnthropic() {
	register(ProviderAnthropic, newAnthropicProvider)
}
```

- [ ] **Step 4: Verify LLM provider files compile**

Run: `go build ./pkg/llm/...`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add pkg/llm/gemini.go pkg/llm/openai.go pkg/llm/anthropic.go
git commit -m "refactor(llm): replace init() with exported Register functions"
```

---

### Task 2: OAuth Providers — Remove init(), export Register functions

**Files:**

- Modify: `pkg/providers/github/github.go`
- Modify: `pkg/providers/slack/slack.go`
- Modify: `pkg/providers/dropbox/dropbox.go`

- [ ] **Step 1: Edit github/github.go — replace init() with Register()**

Replace lines 33-37 in `pkg/providers/github/github.go`:

```go
func init() {
	providers.RegisterOAuthProvider(ID, func() providers.OAuthProvider {
		return GetClient()
	})
}
```

with:

```go
// Register registers the GitHub OAuth provider in the global provider registry.
func Register() {
	providers.RegisterOAuthProvider(ID, func() providers.OAuthProvider {
		return GetClient()
	})
}
```

- [ ] **Step 2: Edit slack/slack.go — replace init() with Register()**

Replace lines 28-32 in `pkg/providers/slack/slack.go`:

```go
func init() {
	providers.RegisterOAuthProvider(ID, func() providers.OAuthProvider {
		return GetClient()
	})
}
```

with:

```go
// Register registers the Slack OAuth provider in the global provider registry.
func Register() {
	providers.RegisterOAuthProvider(ID, func() providers.OAuthProvider {
		return GetClient()
	})
}
```

- [ ] **Step 3: Edit dropbox/dropbox.go — replace init() with Register()**

Replace lines 29-33 in `pkg/providers/dropbox/dropbox.go`:

```go
func init() {
	providers.RegisterOAuthProvider(ID, func() providers.OAuthProvider {
		return GetClient()
	})
}
```

with:

```go
// Register registers the Dropbox OAuth provider in the global provider registry.
func Register() {
	providers.RegisterOAuthProvider(ID, func() providers.OAuthProvider {
		return GetClient()
	})
}
```

- [ ] **Step 4: Verify OAuth provider files compile**

Run: `go build ./pkg/providers/github/... ./pkg/providers/slack/... ./pkg/providers/dropbox/...`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add pkg/providers/github/github.go pkg/providers/slack/slack.go pkg/providers/dropbox/dropbox.go
git commit -m "refactor(providers): replace init() with exported Register functions"
```

---

### Task 3: Reexec Handler — Remove init(), export Register function

**Files:**

- Modify: `pkg/executor/runtime/shell/shell.go`

- [ ] **Step 1: Edit shell.go — replace init() with Register()**

Replace lines 29-31 in `pkg/executor/runtime/shell/shell.go`:

```go
func init() {
	reexec.Register("shell", reexecRun)
}
```

with:

```go
// Register registers the shell reexec handler for self-reexecution.
func Register() {
	reexec.Register("shell", reexecRun)
}
```

- [ ] **Step 2: Verify reexec file compiles**

Run: `go build ./pkg/executor/runtime/shell/...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/executor/runtime/shell/shell.go
git commit -m "refactor(executor): replace init() with exported Register function for shell reexec"
```

---

### Task 4: Add LLM wiring in internal/server/llm.go

**Files:**

- Create: `internal/server/llm.go`

- [ ] **Step 1: Create internal/server/llm.go**

```go
package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/llm"
)

// LLMModules registers all LLM provider backends via fx.
var LLMModules = fx.Options(
	fx.Invoke(
		llm.RegisterGemini,
		llm.RegisterOpenAI,
		llm.RegisterAnthropic,
	),
)
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/server/...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/server/llm.go
git commit -m "feat(server): add LLM provider wiring module"
```

---

### Task 5: Add OAuth wiring in internal/server/providers.go

**Files:**

- Create: `internal/server/providers.go`

- [ ] **Step 1: Create internal/server/providers.go**

```go
package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/providers/dropbox"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/providers/slack"
)

// OAuthModules registers all OAuth provider factories via fx.
var OAuthModules = fx.Options(
	fx.Invoke(
		github.Register,
		slack.Register,
		dropbox.Register,
	),
)
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/server/...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/server/providers.go
git commit -m "feat(server): add OAuth provider wiring module"
```

---

### Task 6: Add Reexec wiring in internal/server/reexec.go

**Files:**

- Create: `internal/server/reexec.go`

- [ ] **Step 1: Create internal/server/reexec.go**

```go
package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/executor/runtime/shell"
)

// ReexecModules registers all reexec handlers via fx.
var ReexecModules = fx.Options(
	fx.Invoke(
		shell.Register,
	),
)
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/server/...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/server/reexec.go
git commit -m "feat(server): add reexec handler wiring module"
```

---

### Task 7: Wire new modules into internal/server/fx.go

**Files:**

- Modify: `internal/server/fx.go`

- [ ] **Step 1: Edit fx.go — add LLMModules, OAuthModules, ReexecModules to Modules**

In `internal/server/fx.go`, line 20, the `Modules` var currently is:

```go
var Modules = fx.Options(
	metrics.Module(),
	modules.Modules,
	NotifyModules,
	MediaModules,
	fx.Provide(
```

Add the three new modules after `MediaModules`:

```go
var Modules = fx.Options(
	metrics.Module(),
	modules.Modules,
	NotifyModules,
	MediaModules,
	LLMModules,
	OAuthModules,
	ReexecModules,
	fx.Provide(
```

- [ ] **Step 2: Verify server compiles**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/server/fx.go
git commit -m "feat(server): wire LLM, OAuth, and reexec modules into fx"
```

---

### Task 8: Fix LLM tests — add Register calls in TestMain

**Files:**

- Modify: `pkg/llm/llm_test.go`

- [ ] **Step 1: Edit llm_test.go — add Register calls before m.Run()**

In `pkg/llm/llm_test.go`, the `TestMain` currently is:

```go
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
```

Add Register calls before `m.Run()`. Also add the import for the `llm` package (the test is in `package llm_test`, so it imports `pkg/llm` as `llm` — check the existing import).

Replace the `TestMain` function with:

```go
func TestMain(m *testing.M) {
	llm.RegisterGemini()
	llm.RegisterOpenAI()
	llm.RegisterAnthropic()
	os.Exit(m.Run())
}
```

And add the `llm` import if not already present:

```go
import (
	"os"
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/llm"
)
```

- [ ] **Step 2: Run LLM tests to verify they pass**

Run: `go test ./pkg/llm/... -v -count=1`
Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add pkg/llm/llm_test.go
git commit -m "test(llm): add explicit Register calls in TestMain"
```

---

### Task 9: Final verification

- [ ] **Step 1: Run full lint**

Run: `go tool task lint`
Expected: No lint errors.

- [ ] **Step 2: Run full test suite**

Run: `go tool task test`
Expected: All tests pass.

- [ ] **Step 3: Verify no remaining init() registration functions**

Run: `rg "func init\(\)" pkg/llm/ pkg/providers/github/ pkg/providers/slack/ pkg/providers/dropbox/ pkg/executor/runtime/shell/`
Expected: No output (no init() functions remain in those packages).

- [ ] **Step 4: Commit any fixes (if needed)**

```bash
git add -A
git commit -m "chore: final lint and test fixes for init() removal"
```
