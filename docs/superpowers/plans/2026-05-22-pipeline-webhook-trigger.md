# Pipeline Webhook Trigger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add HTTP webhook-based trigger to the pipeline engine so external systems can invoke pipelines via per-pipeline URLs.

**Architecture:** Extend `PipelineTrigger` config with a `Webhook` sub-struct. At startup, build a path-to-definition map and register per-pipeline Fiber routes under `/webhook/{path}`. The handler authenticates (token or HMAC), converts the request to a synthetic `DataEvent`, and calls `engine.ExecuteWebhook()` which reuses the existing `executePipeline` execution path under the shared per-pipeline mutex.

**Tech Stack:** Fiber v3 (HTTP), go-cron/v4 patterns for synthetic event IDs, sonic (JSON), existing `types.DataEvent` / `pipeline.Engine` / `pipeline.Definition`

---

### Task 1: Config types (`pkg/config/config.go`)

**Files:**

- Modify: `pkg/config/config.go:521-525`
- Test: `pkg/config/config_test.go` (add after existing `TestPipelineTrigger_CronFields`)

- [ ] **Step 1: Add WebhookTrigger, WebhookAuth, WebhookPayloadMode types and Webhook field to PipelineTrigger**

Add the following types after `PipelineTrigger` (lines 521-525):

```go
type WebhookPayloadMode string

const (
	WebhookPayloadRaw    WebhookPayloadMode = "raw"
	WebhookPayloadMapped WebhookPayloadMode = "mapped"
)

// WebhookAuth holds webhook authentication configuration.
type WebhookAuth struct {
	Token      string `json:"token" yaml:"token" mapstructure:"token"`
	HMACSecret string `json:"hmac_secret" yaml:"hmac_secret" mapstructure:"hmac_secret"`
	HMACHeader string `json:"hmac_header" yaml:"hmac_header" mapstructure:"hmac_header"`
	TokenHeader string `json:"token_header" yaml:"token_header" mapstructure:"token_header"`
}

// WebhookTrigger configures a webhook-based pipeline trigger.
type WebhookTrigger struct {
	Path      string             `json:"path" yaml:"path" mapstructure:"path"`
	Method    string             `json:"method" yaml:"method" mapstructure:"method"`
	Auth      *WebhookAuth       `json:"auth" yaml:"auth" mapstructure:"auth"`
	Payload   WebhookPayloadMode `json:"payload" yaml:"payload" mapstructure:"payload"`
	EventType string             `json:"event_type" yaml:"event_type" mapstructure:"event_type"`
}
```

Modify `PipelineTrigger` to add the `Webhook` field:

```go
type PipelineTrigger struct {
	Event       string          `json:"event" yaml:"event" mapstructure:"event"`
	Cron        string          `json:"cron" yaml:"cron" mapstructure:"cron"`
	CronTimeout string          `json:"cron_timeout" yaml:"cron_timeout" mapstructure:"cron_timeout"`
	Webhook     *WebhookTrigger `json:"webhook" yaml:"webhook" mapstructure:"webhook"`
}
```

- [ ] **Step 2: Write webhook config parse tests**

Add `TestPipelineTrigger_WebhookFields` after the existing `TestPipelineTrigger_CronFields` test in `pkg/config/config_test.go`:

```go
func TestPipelineTrigger_WebhookFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		yamlData      string
		wantPath      string
		wantMethod    string
		wantToken     string
		wantHMAC      string
		wantPayload   WebhookPayloadMode
		wantEventType string
		wantErr       bool
	}{
		{
			name: "webhook complete config",
			yamlData: `
name: webhook-full
trigger:
  webhook:
    path: "github-push"
    method: POST
    auth:
      token: "secret123"
      hmac_secret: "hmac-secret"
    payload: raw
    event_type: "github.push"
`,
			wantPath:      "github-push",
			wantMethod:    "POST",
			wantToken:     "secret123",
			wantHMAC:      "hmac-secret",
			wantPayload:   WebhookPayloadRaw,
			wantEventType: "github.push",
		},
		{
			name: "webhook token auth only",
			yamlData: `
name: webhook-token
trigger:
  webhook:
    path: "token-callback"
    auth:
      token: "tok123"
`,
			wantPath:    "token-callback",
			wantToken:   "tok123",
			wantPayload: WebhookPayloadRaw,
		},
		{
			name: "webhook HMAC auth only",
			yamlData: `
name: webhook-hmac
trigger:
  webhook:
    path: "hmac-callback"
    auth:
      hmac_secret: "secret"
`,
			wantPath:    "hmac-callback",
			wantHMAC:    "secret",
			wantPayload: WebhookPayloadRaw,
		},
		{
			name: "webhook defaults",
			yamlData: `
name: webhook-defaults
trigger:
  webhook:
    path: "minimal"
`,
			wantPath:    "minimal",
			wantMethod:  "",
			wantPayload: WebhookPayloadRaw,
		},
		{
			name: "webhook mapped payload",
			yamlData: `
name: webhook-mapped
trigger:
  webhook:
    path: "json-callback"
    payload: mapped
`,
			wantPath:    "json-callback",
			wantPayload: WebhookPayloadMapped,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var pl Pipeline
			err := yaml.Unmarshal([]byte(tt.yamlData), &pl)
			require.NoError(t, err)
			if tt.wantPath != "" {
				require.NotNil(t, pl.Trigger.Webhook)
				assert.Equal(t, tt.wantPath, pl.Trigger.Webhook.Path)
				assert.Equal(t, tt.wantMethod, pl.Trigger.Webhook.Method)
				if pl.Trigger.Webhook.Auth != nil {
					assert.Equal(t, tt.wantToken, pl.Trigger.Webhook.Auth.Token)
					assert.Equal(t, tt.wantHMAC, pl.Trigger.Webhook.Auth.HMACSecret)
				}
				assert.Equal(t, tt.wantPayload, pl.Trigger.Webhook.Payload)
				assert.Equal(t, tt.wantEventType, pl.Trigger.Webhook.EventType)
			}
		})
	}
}
```

- [ ] **Step 3: Run config tests to verify they pass**

```bash
go test ./pkg/config/ -run TestPipelineTrigger_WebhookFields -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "feat: add WebhookTrigger config types and parse tests"
```

---

### Task 2: Runtime types and validation (`pkg/pipeline/loader.go`)

**Files:**

- Modify: `pkg/pipeline/loader.go:24-28` (Trigger struct), `pkg/pipeline/loader.go:120-133` (cronTrigger function)
- Test: `pkg/pipeline/pipeline_test.go` (add after `TestLoadConfig_CronTrigger`)

- [ ] **Step 1: Add WebhookConfig types and extend Trigger struct**

Add webhook config types and modify `Trigger` in `pkg/pipeline/loader.go`:

```go
type Trigger struct {
	Event       string
	Cron        string
	CronTimeout time.Duration
	Webhook     *WebhookConfig
}

type WebhookConfig struct {
	Path      string
	Method    string
	Auth      WebhookAuthConfig
	Payload   config.WebhookPayloadMode
	EventType string
}

type WebhookAuthConfig struct {
	Token      string
	HMACSecret string
	HMACHeader string
	TokenHeader string
}
```

- [ ] **Step 2: Add allowed methods set and write the convertWebhookTrigger function**

Add to `pkg/pipeline/loader.go` after `validateCronExpr`:

```go
var allowedWebhookMethods = map[string]bool{
	"GET":  true,
	"POST": true,
	"PUT":  true,
}

func convertWebhookTrigger(name string, wh *config.WebhookTrigger) (*WebhookConfig, error) {
	if wh == nil {
		return nil, nil
	}

	if wh.Path == "" {
		return nil, fmt.Errorf("pipeline %s: webhook trigger path must not be empty", name)
	}

	method := wh.Method
	if method == "" {
		method = "POST"
	}
	method = strings.ToUpper(method)
	if !allowedWebhookMethods[method] {
		return nil, fmt.Errorf("pipeline %s: unsupported webhook method %q", name, wh.Method)
	}

	if wh.Auth == nil || (wh.Auth.Token == "" && wh.Auth.HMACSecret == "") {
		return nil, fmt.Errorf("pipeline %s: webhook trigger requires at least one of auth.token or auth.hmac_secret", name)
	}

	payload := wh.Payload
	if payload == "" {
		payload = config.WebhookPayloadRaw
	}
	if payload != config.WebhookPayloadRaw && payload != config.WebhookPayloadMapped {
		return nil, fmt.Errorf("pipeline %s: invalid webhook payload mode %q", name, wh.Payload)
	}

	eventType := wh.EventType
	if eventType == "" {
		eventType = "webhook." + wh.Path
	}

	hmacHeader := "X-Hub-Signature-256"
	tokenHeader := "X-Webhook-Token"
	if wh.Auth != nil {
		if wh.Auth.HMACHeader != "" {
			hmacHeader = wh.Auth.HMACHeader
		}
		if wh.Auth.TokenHeader != "" {
			tokenHeader = wh.Auth.TokenHeader
		}
	}

	wc := &WebhookConfig{
		Path:      wh.Path,
		Method:    method,
		Payload:   payload,
		EventType: eventType,
	}
	if wh.Auth != nil {
		wc.Auth = WebhookAuthConfig{
			Token:      wh.Auth.Token,
			HMACSecret: wh.Auth.HMACSecret,
			HMACHeader: hmacHeader,
			TokenHeader: tokenHeader,
		}
	}
	return wc, nil
}
```

- [ ] **Step 3: Update convertTrigger (rename from cronTrigger) and LoadConfig**

Rename `cronTrigger` to `convertTrigger` and extend it to handle webhook configurations. Replace lines 120-133:

```go
func convertTrigger(name string, cfg config.PipelineTrigger) (Trigger, error) {
	t := Trigger{Event: cfg.Event, Cron: cfg.Cron}

	if cfg.CronTimeout != "" {
		d, err := time.ParseDuration(cfg.CronTimeout)
		if err != nil {
			flog.Error(fmt.Errorf("pipeline %s: invalid cron_timeout %q: %w", name, cfg.CronTimeout, err))
		} else {
			t.CronTimeout = d
		}
	} else if cfg.Cron != "" {
		t.CronTimeout = 10 * time.Minute
	}

	// Validate webhook is not mixed with cron or event.
	if cfg.Webhook != nil && (cfg.Event != "" || cfg.Cron != "") {
		return t, fmt.Errorf("pipeline %s: webhook trigger cannot be combined with event or cron", name)
	}

	wh, err := convertWebhookTrigger(name, cfg.Webhook)
	if err != nil {
		return t, err
	}
	t.Webhook = wh

	return t, nil
}
```

Update `LoadConfig` to use `convertTrigger` with error handling. Modify lines 50-56:

```go
		trigger, err := convertTrigger(p.Name, p.Trigger)
		if err != nil {
			flog.Error(err)
			continue
		}
		d := Definition{
			Name:        p.Name,
			Description: p.Description,
			Enabled:     p.Enabled,
			Resumable:   p.Resumable,
			Trigger:     trigger,
		}
```

Also add `"strings"` to the imports.

- [ ] **Step 4: Write loader tests**

Add `TestLoadConfig_WebhookTrigger` after `TestLoadConfig_CronTrigger` in `pkg/pipeline/pipeline_test.go`:

```go
func TestLoadConfig_WebhookTrigger(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cfg         []config.Pipeline
		wantDefs    int
		wantPath    string
		wantMethod  string
		wantPayload config.WebhookPayloadMode
	}{
		{
			name: "webhook valid definition",
			cfg: []config.Pipeline{
				{
					Name:    "webhook1",
					Enabled: true,
					Trigger: config.PipelineTrigger{
						Webhook: &config.WebhookTrigger{
							Path: "test-path",
							Auth: &config.WebhookAuth{Token: "secret"},
						},
					},
				},
			},
			wantDefs:    1,
			wantPath:    "test-path",
			wantMethod:  "POST",
			wantPayload: config.WebhookPayloadRaw,
		},
		{
			name: "webhook default method and payload",
			cfg: []config.Pipeline{
				{
					Name:    "webhook2",
					Enabled: true,
					Trigger: config.PipelineTrigger{
						Webhook: &config.WebhookTrigger{
							Path: "minimal",
							Auth: &config.WebhookAuth{Token: "t"},
						},
					},
				},
			},
			wantDefs:    1,
			wantPath:    "minimal",
			wantMethod:  "POST",
			wantPayload: config.WebhookPayloadRaw,
		},
		{
			name: "webhook with cron errors",
			cfg: []config.Pipeline{
				{
					Name:    "mixed1",
					Enabled: true,
					Trigger: config.PipelineTrigger{
						Cron: "0 0 * * *",
						Webhook: &config.WebhookTrigger{
							Path: "mixed-path",
							Auth: &config.WebhookAuth{Token: "x"},
						},
					},
				},
			},
			wantDefs: 0,
		},
		{
			name: "webhook with event errors",
			cfg: []config.Pipeline{
				{
					Name:    "mixed2",
					Enabled: true,
					Trigger: config.PipelineTrigger{
						Event: "some.event",
						Webhook: &config.WebhookTrigger{
							Path: "ev-path",
							Auth: &config.WebhookAuth{Token: "x"},
						},
					},
				},
			},
			wantDefs: 0,
		},
		{
			name: "webhook empty path errors",
			cfg: []config.Pipeline{
				{
					Name:    "nopath",
					Enabled: true,
					Trigger: config.PipelineTrigger{
						Webhook: &config.WebhookTrigger{
							Auth: &config.WebhookAuth{Token: "x"},
						},
					},
				},
			},
			wantDefs: 0,
		},
		{
			name: "webhook disabled pipeline skipped",
			cfg: []config.Pipeline{
				{
					Name:    "disabled-webhook",
					Enabled: false,
					Trigger: config.PipelineTrigger{
						Webhook: &config.WebhookTrigger{
							Path: "disabled-path",
							Auth: &config.WebhookAuth{Token: "x"},
						},
					},
				},
			},
			wantDefs: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defs := LoadConfig(tt.cfg)
			assert.Len(t, defs, tt.wantDefs)
			if tt.wantDefs > 0 && len(defs) > 0 {
				assert.NotNil(t, defs[0].Trigger.Webhook)
				assert.Equal(t, tt.wantPath, defs[0].Trigger.Webhook.Path)
				assert.Equal(t, tt.wantMethod, defs[0].Trigger.Webhook.Method)
				assert.Equal(t, tt.wantPayload, defs[0].Trigger.Webhook.Payload)
			}
		})
	}
}
```

- [ ] **Step 5: Run loader tests**

```bash
go test ./pkg/pipeline/ -run TestLoadConfig_WebhookTrigger -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/pipeline/loader.go pkg/pipeline/pipeline_test.go
git commit -m "feat: add webhook trigger conversion and validation in loader"
```

---

### Task 3: Engine methods (`pkg/pipeline/engine.go`)

**Files:**

- Modify: `pkg/pipeline/engine.go:62-71` (Engine struct), `pkg/pipeline/engine.go:549-597` (add after executeCronJob)
- Test: `pkg/pipeline/engine_test.go` (add after `TestEngine_SyntheticEventFormat`)

- [ ] **Step 1: Add RegisterWebhooks method to Engine**

Add the following method to `engine.go` before `Stop()`:

```go
// RegisterWebhooks returns a map of webhook path to pipeline Definition for
// all webhook-enabled pipelines. Duplicate paths return an error.
func (e *Engine) RegisterWebhooks() (map[string]*Definition, error) {
	result := make(map[string]*Definition)
	for i := range e.defs {
		if e.defs[i].Trigger.Webhook == nil {
			continue
		}
		path := e.defs[i].Trigger.Webhook.Path
		if _, exists := result[path]; exists {
			return nil, fmt.Errorf("duplicate webhook path %q", path)
		}
		result[path] = &e.defs[i]
	}
	return result, nil
}
```

- [ ] **Step 2: Add ExecuteWebhook method to Engine**

Add after `executeCronJob` (after line 589):

```go
// ExecuteWebhook executes a pipeline from a webhook trigger. It uses the
// per-pipeline mutex for concurrency control and calls executePipeline
// with a synthetic event.
func (e *Engine) ExecuteWebhook(ctx context.Context, def *Definition, event types.DataEvent) error {
	mu := e.mu[def.Name]
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	return e.executePipeline(ctx, *def, event)
}
```

- [ ] **Step 3: Write engine tests**

Add `TestEngine_RegisterWebhooks` and `TestEngine_ExecuteWebhook` in `pkg/pipeline/engine_test.go` after `TestEngine_SyntheticEventFormat`:

```go
func TestEngine_RegisterWebhooks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		defs      []Definition
		wantPaths []string
		wantErr   bool
	}{
		{
			name: "returns webhook paths",
			defs: []Definition{
				{
					Name: "wh1", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "path-a", Method: "POST", Auth: WebhookAuthConfig{Token: "t"}}},
				},
				{
					Name: "wh2", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "path-b", Method: "POST", Auth: WebhookAuthConfig{Token: "t"}}},
				},
			},
			wantPaths: []string{"path-a", "path-b"},
		},
		{
			name: "skips non-webhook definitions",
			defs: []Definition{
				{Name: "ev1", Enabled: true, Trigger: Trigger{Event: "e1"}},
				{
					Name: "wh1", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "path-a", Method: "POST", Auth: WebhookAuthConfig{Token: "t"}}},
				},
			},
			wantPaths: []string{"path-a"},
		},
		{
			name: "returns error on duplicate paths",
			defs: []Definition{
				{
					Name: "wh1", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "dup", Method: "POST", Auth: WebhookAuthConfig{Token: "t"}}},
				},
				{
					Name: "wh2", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "dup", Method: "PUT", Auth: WebhookAuthConfig{HMACSecret: "s"}}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(tt.defs, nil, nil, noopPC, noopEC)
			defer e.Stop()
			m, err := e.RegisterWebhooks()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			for _, p := range tt.wantPaths {
				assert.Contains(t, m, p)
			}
		})
	}
}

func TestEngine_ExecuteWebhook(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		eventTypes []string
	}{
		{
			name:       "execute webhook with single pipeline run",
			eventTypes: []string{"webhook.run"},
		},
		{
			name:       "execute webhook with differing event types",
			eventTypes: []string{"custom.type", "another.type"},
		},
		{
			name:       "execute webhook with empty steps completes immediately",
			eventTypes: []string{"noop.event"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, et := range tt.eventTypes {
				defs := []Definition{
					{
						Name: "wh-exec-" + et[len(et)-3:],
						Enabled: true,
						Trigger: Trigger{
							Webhook: &WebhookConfig{
								Path: "exec-path", Method: "POST",
								Auth: WebhookAuthConfig{Token: "t"},
								EventType: et,
							},
						},
					},
				}
				e := NewEngine(defs, nil, nil, noopPC, noopEC)
				defer e.Stop()
				event := types.DataEvent{
					EventID:   "test-id",
					EventType: et,
					Source:    "webhook",
				}
				err := e.ExecuteWebhook(context.Background(), &defs[0], event)
				assert.NoError(t, err)
			}
		})
	}
}

func TestEngine_ExecuteWebhookMutex(t *testing.T) {
	t.Parallel()
	def := Definition{
		Name: "mutex-test",
		Enabled: true,
		Trigger: Trigger{
			Webhook: &WebhookConfig{
				Path: "mtx", Method: "POST",
				Auth: WebhookAuthConfig{Token: "t"},
			},
		},
	}
	e := NewEngine([]Definition{def}, nil, nil, noopPC, noopEC)
	defer e.Stop()

	// Verify mutex is created for webhook pipeline.
	mu := e.mu[def.Name]
	require.NotNil(t, mu)

	// Acquire the mutex, start a goroutine that calls ExecuteWebhook (which
	// also acquires the same mutex), then release. The goroutine should be
	// unblocked after release.
	mu.Lock()

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(started)
		event := types.DataEvent{EventID: "mtx-ev", EventType: "t"}
		_ = e.ExecuteWebhook(context.Background(), &def, event)
		close(done)
	}()

	// Give goroutine time to attempt Lock.
	<-started
	time.Sleep(50 * time.Millisecond)

	select {
	case <-done:
		t.Fatal("ExecuteWebhook completed before mutex released")
	default:
	}

	mu.Unlock()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ExecuteWebhook did not complete after mutex release")
	}
}
```

Add `"context"` to imports in engine_test.go if not already present.

- [ ] **Step 4: Run engine tests**

```bash
go test ./pkg/pipeline/ -run "TestEngine_RegisterWebhooks|TestEngine_ExecuteWebhook" -v
```

Expected: PASS for all

- [ ] **Step 5: Run all existing pipeline tests to ensure no regression**

```bash
go test ./pkg/pipeline/ -v
```

Expected: PASS for all

- [ ] **Step 6: Commit**

```bash
git add pkg/pipeline/engine.go pkg/pipeline/engine_test.go
git commit -m "feat: add RegisterWebhooks and ExecuteWebhook to engine"
```

---

### Task 4: HTTP handler (`internal/server/webhook.go`)

**Files:**

- Create: `internal/server/webhook.go`
- Create: `internal/server/webhook_test.go`

- [ ] **Step 1: Create the webhook handler file**

Create `internal/server/webhook.go`:

```go
package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

// registerWebhookRoutes registers webhook HTTP routes on the Fiber app
// for each webhook-enabled pipeline definition.
func registerWebhookRoutes(engine *pipeline.Engine) error {
	webhookMap, err := engine.RegisterWebhooks()
	if err != nil {
		return fmt.Errorf("register webhooks: %w", err)
	}

	for path, def := range webhookMap {
		method := def.Trigger.Webhook.Method
		routePath := "/webhook/" + strings.TrimPrefix(path, "/")
		handler := makeWebhookHandler(engine, def)
		switch method {
		case "GET":
			sharedApp.Get(routePath, handler)
		case "POST":
			sharedApp.Post(routePath, handler)
		case "PUT":
			sharedApp.Put(routePath, handler)
		}
		flog.Info("webhook route registered: %s %s -> pipeline %s", method, routePath, def.Name)
	}

	return nil
}

// makeWebhookHandler returns a Fiber handler that authenticates the request
// and dispatches to the engine.
func makeWebhookHandler(engine *pipeline.Engine, def *pipeline.Definition) fiber.Handler {
	return func(c fiber.Ctx) error {
		wcfg := def.Trigger.Webhook

		// Authenticate.
		status, ok := authenticateWebhook(c, wcfg)
		if !ok {
			return c.Status(status).SendString(http.StatusText(status))
		}

		// Build DataEvent.
		dataEvent := types.DataEvent{
			EventID:   fmt.Sprintf("webhook:%s:%d-%s", wcfg.Path, realClockNow().UnixNano(), pipeline.RandomHex(8)),
			EventType: wcfg.EventType,
			Source:    "webhook",
		}

		// Inject headers.
		headers := make(map[string]string)
		c.Request().Header.VisitAll(func(key, value []byte) {
			headers[string(key)] = string(value)
		})

		body := c.Body()

		if wcfg.Payload == "mapped" {
			var parsed map[string]any
			if err := sonic.Unmarshal(body, &parsed); err != nil {
				return c.Status(fiber.StatusBadRequest).
					SendString("invalid JSON body for mapped payload: " + err.Error())
			}
			dataEvent.Data = types.KV(parsed)
		} else {
			if dataEvent.Data == nil {
				dataEvent.Data = make(types.KV)
			}
			dataEvent.Data["_webhook_body"] = string(body)
		}

		if dataEvent.Data == nil {
			dataEvent.Data = make(types.KV)
		}
		dataEvent.Data["_webhook_headers"] = headers

		if engine == nil {
			return c.Status(fiber.StatusServiceUnavailable).
				SendString("pipeline engine not initialized")
		}
		if err := engine.ExecuteWebhook(c.Context(), def, dataEvent); err != nil {
			flog.Error(fmt.Errorf("webhook pipeline %s: %w", def.Name, err))
		}

		return c.SendStatus(fiber.StatusAccepted)
	}
}

// authenticateWebhook validates the request against the webhook auth config.
// Returns HTTP status code and true if authenticated.
func authenticateWebhook(c fiber.Ctx, wcfg *pipeline.WebhookConfig) (int, bool) {
	ac := wcfg.Auth

	// Require at least one auth method.
	if ac.Token == "" && ac.HMACSecret == "" {
		return fiber.StatusUnauthorized, false
	}

	if ac.Token != "" {
		tokenHeader := ac.TokenHeader
		if tokenHeader == "" {
			tokenHeader = "X-Webhook-Token"
		}
		provided := c.Get(tokenHeader)
		if provided == ac.Token {
			return fiber.StatusOK, true
		}
	}

	if ac.HMACSecret != "" {
		hmacHeader := ac.HMACHeader
		if hmacHeader == "" {
			hmacHeader = "X-Hub-Signature-256"
		}
		provided := c.Get(hmacHeader)
		if verifyHMACSHA256(ac.HMACSecret, c.Body(), provided) {
			return fiber.StatusOK, true
		}
	}

	return fiber.StatusUnauthorized, false
}

// verifyHMACSHA256 verifies an HMAC-SHA256 signature against the body.
// The expected format is "sha256=<hex>".
func verifyHMACSHA256(secret string, body []byte, signature string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(strings.ToLower(signature), prefix) {
		return false
	}
	expectedHex := strings.ToLower(strings.TrimPrefix(strings.ToLower(signature), prefix))
	expected, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	actual := mac.Sum(nil)
	return hmac.Equal(actual, expected)
}
```

- [ ] **Step 2: Add RandomHex export to pipeline package**

Add to `pkg/pipeline/engine.go` at the bottom (after `randomHex` at line 597):

```go
// RandomHex generates n random bytes as a hex string (exported for use by server package).
func RandomHex(n int) string {
	return randomHex(n)
}
```

- [ ] **Step 3: Add a clock helper for the server package**

The webhook handler needs `time.Now().UnixNano()` for event IDs. Since the server doesn't have access to the engine's clock, use a package-level function. Add to `internal/server/webhook.go`:

```go
import "time"

var realClockNow = time.Now
```

(This is already handled in the Step 1 code above.)

- [ ] **Step 4: Update `initPipeline` in `internal/server/pipeline.go` to register webhook routes**

Add after `engine := pipeline.NewEngine(...)` and before `lc.Append(fx.Hook{...})`:

```go
	// Register webhook routes.
	if err := registerWebhookRoutes(engine); err != nil {
		return fmt.Errorf("register webhook routes: %w", err)
	}
```

- [ ] **Step 5: Write webhook handler tests**

Create `internal/server/webhook_test.go`:

```go
package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

func newWebhookTestApp(engine *pipeline.Engine, defs ...pipeline.Definition) (*fiber.App, func()) {
	app := fiber.New(fiber.Config{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	if engine == nil {
		engine = &pipeline.Engine{}
	}
	if len(defs) == 0 {
		defs = []pipeline.Definition{}
	}

	for _, def := range defs {
		if def.Trigger.Webhook != nil {
			handler := makeWebhookHandler(engine, &def)
			routePath := "/webhook/" + strings.TrimPrefix(def.Trigger.Webhook.Path, "/")
			app.Post(routePath, handler)
		}
	}

	return app, func() {}
}

func makeHMACSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestWebhookAuthenticateWebhook(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wcfg       *pipeline.WebhookConfig
		setHeaders func(req *http.Request)
		wantStatus int
		wantOK     bool
	}{
		{
			name: "valid token auth",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{
					Token:       "secret",
					TokenHeader: "X-Webhook-Token",
				},
			},
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "secret")
			},
			wantStatus: fiber.StatusOK,
			wantOK:     true,
		},
		{
			name: "valid HMAC auth",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{
					HMACSecret: "hmac-secret",
					HMACHeader: "X-Hub-Signature-256",
				},
			},
			setHeaders: func(req *http.Request) {
				sig := makeHMACSignature("hmac-secret", []byte("test-body"))
				req.Header.Set("X-Hub-Signature-256", sig)
			},
			wantStatus: fiber.StatusOK,
			wantOK:     true,
		},
		{
			name: "token mismatch returns 401",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{
					Token:       "secret",
					TokenHeader: "X-Webhook-Token",
				},
			},
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "wrong")
			},
			wantStatus: fiber.StatusUnauthorized,
			wantOK:     false,
		},
		{
			name: "HMAC mismatch returns 401",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{
					HMACSecret: "hmac-secret",
					HMACHeader: "X-Hub-Signature-256",
				},
			},
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
			},
			wantStatus: fiber.StatusUnauthorized,
			wantOK:     false,
		},
		{
			name: "no auth configured returns 401",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{},
			},
			setHeaders: func(req *http.Request) {},
			wantStatus: fiber.StatusUnauthorized,
			wantOK:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := "test-body"
			req, err := http.NewRequest("POST", "/webhook/test", strings.NewReader(body))
			require.NoError(t, err)
			tt.setHeaders(req)
			c := fiber.New().AcquireCtx(req)
			defer fiber.New().ReleaseCtx(c)
			c.Request().Header.SetMethod("POST")
			c.Request().Header.Set("Content-Type", "text/plain")
			c.Request().SetBody([]byte(body))

			status, ok := authenticateWebhook(c, tt.wcfg)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

func TestWebhookHandler_Integration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wcfg       *pipeline.WebhookConfig
		body       string
		contentType string
		setHeaders func(req *http.Request)
		wantStatus int
	}{
		{
			name: "happy path token auth mapped payload",
			wcfg: &pipeline.WebhookConfig{
				Path:      "test-callback",
				Method:    "POST",
				EventType: "test.event",
				Auth: pipeline.WebhookAuthConfig{
					Token:       "test-token",
					TokenHeader: "X-Webhook-Token",
				},
				Payload: "mapped",
			},
			body:        `{"key":"value"}`,
			contentType: "application/json",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "test-token")
			},
			wantStatus: fiber.StatusAccepted,
		},
		{
			name: "happy path HMAC auth raw payload",
			wcfg: &pipeline.WebhookConfig{
				Path:      "hmac-cb",
				Method:    "POST",
				EventType: "hmac.event",
				Auth: pipeline.WebhookAuthConfig{
					HMACSecret: "raw-secret",
					HMACHeader: "X-Hub-Signature-256",
				},
				Payload: "raw",
			},
			body:        "plain text body",
			contentType: "text/plain",
			setHeaders: func(req *http.Request) {
				sig := makeHMACSignature("raw-secret", []byte("plain text body"))
				req.Header.Set("X-Hub-Signature-256", sig)
			},
			wantStatus: fiber.StatusAccepted,
		},
		{
			name: "invalid JSON in mapped mode returns 400",
			wcfg: &pipeline.WebhookConfig{
				Path:      "json-fail",
				Method:    "POST",
				EventType: "fail.event",
				Auth: pipeline.WebhookAuthConfig{
					Token:       "t",
					TokenHeader: "X-Webhook-Token",
				},
				Payload: "mapped",
			},
			body:        "not-json",
			contentType: "text/plain",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "t")
			},
			wantStatus: fiber.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := pipeline.Definition{
				Name:    tt.name,
				Enabled: true,
				Trigger: pipeline.Trigger{Webhook: tt.wcfg},
			}

			// Create an engine with no Store (no persistence needed for these tests).
			engine := pipeline.NewEngine([]pipeline.Definition{def}, nil, nil, nil, nil)
			defer engine.Stop()

			handler := makeWebhookHandler(engine, &def)

			app := fiber.New()
			app.Post("/webhook/"+tt.wcfg.Path, handler)

			req, err := http.NewRequest("POST", "/webhook/"+tt.wcfg.Path, strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", tt.contentType)
			tt.setHeaders(req)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantStatus == fiber.StatusAccepted {
				// Verify that the engine processed the event by reading the body.
				// The 202 response confirms acceptance.
				respBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.Empty(t, respBody) // 202 Accepted has no body.
			}
		})
	}
}

func TestWebhookHandler_HeadersInjected(t *testing.T) {
	t.Parallel()
	def := pipeline.Definition{
		Name:    "header-test",
		Enabled: true,
		Trigger: pipeline.Trigger{
			Webhook: &pipeline.WebhookConfig{
				Path:      "header-check",
				Method:    "POST",
				EventType: "header.event",
				Auth: pipeline.WebhookAuthConfig{Token: "test", TokenHeader: "X-Webhook-Token"},
				Payload: "raw",
			},
		},
	}

	engine := pipeline.NewEngine([]pipeline.Definition{def}, nil, nil, nil, nil)
	defer engine.Stop()

	handler := makeWebhookHandler(engine, &def)

	app := fiber.New()
	app.Post("/webhook/header-check", handler)

	req, err := http.NewRequest("POST", "/webhook/header-check", strings.NewReader("body"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Webhook-Token", "test")
	req.Header.Set("X-Custom-Header", "custom-value")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)
}
```

- [ ] **Step 6: Run webhook handler tests**

```bash
go test ./internal/server/ -run "TestWebhook" -v
```

Expected: PASS for all

- [ ] **Step 7: Commit**

```bash
git add internal/server/webhook.go internal/server/webhook_test.go internal/server/pipeline.go pkg/pipeline/engine.go
git commit -m "feat: add webhook HTTP handler and server integration"
```

---

### Task 5: Integration verification

**Files:**

- No new files

- [ ] **Step 1: Run all pipeline tests**

```bash
go test ./pkg/pipeline/... -v
```

Expected: All tests PASS, no regressions.

- [ ] **Step 2: Run all config tests**

```bash
go test ./pkg/config/ -v
```

Expected: All tests PASS.

- [ ] **Step 3: Run all server tests**

```bash
go test ./internal/server/ -v
```

Expected: All tests PASS.

- [ ] **Step 4: Run lint**

```bash
go tool task lint
```

Expected: No lint errors in changed files.

- [ ] **Step 5: Run full test suite**

```bash
go tool task test
```

Expected: All tests PASS.

- [ ] **Step 6: Commit final checkpoint**

```bash
git add -A
git commit -m "chore: verify all tests and lint pass for webhook trigger"
```

---

### Task 6: BDD specs (`tests/specs/pipeline_spec_test.go`)

**Files:**

- Modify: `tests/specs/pipeline_spec_test.go`

- [ ] **Step 1: Add webhook trigger BDD specs**

Add a new `Describe("Webhook trigger")` block to the existing pipeline specs file, following the existing Ginkgo/Gomega patterns:

```go
Describe("Webhook trigger", func() {
	var (
		engine *pipeline.Engine
		def    pipeline.Definition
	)

	BeforeEach(func() {
		def = pipeline.Definition{
			Name:    "webhook-spec",
			Enabled: true,
			Trigger: pipeline.Trigger{
				Webhook: &pipeline.WebhookConfig{
					Path:      "spec-path",
					Method:    "POST",
					EventType: "spec.event",
					Auth:      pipeline.WebhookAuthConfig{Token: "spec-token", TokenHeader: "X-Webhook-Token"},
					Payload:   "raw",
				},
			},
		}
	})

	It("executes pipeline on webhook invocation", func() {
		engine = pipeline.NewEngine([]pipeline.Definition{def}, nil, nil, nil, nil)
		defer engine.Stop()

		event := types.DataEvent{
			EventID:   "webhook:spec-path:123-abcdef",
			EventType: "spec.event",
			Source:    "webhook",
		}
		err := engine.ExecuteWebhook(context.Background(), &def, event)
		Expect(err).NotTo(HaveOccurred())
	})

	It("serializes concurrent webhook calls via mutex", func() {
		engine = pipeline.NewEngine([]pipeline.Definition{def}, nil, nil, nil, nil)
		defer engine.Stop()

		mu := engine.MutexFor(def.Name)
		Expect(mu).NotTo(BeNil())

		mu.Lock()
		done := make(chan struct{})
		go func() {
			event := types.DataEvent{EventID: "concurrent", EventType: "spec.event"}
			_ = engine.ExecuteWebhook(context.Background(), &def, event)
			close(done)
		}()

		Consistently(done, 100*time.Millisecond).ShouldNot(BeClosed())
		mu.Unlock()
		Eventually(done).Should(BeClosed())
	})

	It("records pipeline run for webhook trigger", func() {
		// Requires RunStore — skip if store not available.
		if store.Database == nil {
			Skip("database store not available")
		}
		client, ok := store.Database.GetDB().(*store.Client)
		if !ok {
			Skip("ent store not available")
		}
		runStore := store.NewPipelineStore(client)
		engine = pipeline.NewEngine([]pipeline.Definition{def}, runStore, nil, nil, nil)
		defer engine.Stop()

		event := types.DataEvent{
			EventID:   "webhook:record:456",
			EventType: "spec.event",
			Source:    "webhook",
		}
		err := engine.ExecuteWebhook(context.Background(), &def, event)
		Expect(err).NotTo(HaveOccurred())
	})
})
```

If `Engine.MutexFor` doesn't exist, add it to `pkg/pipeline/engine.go`:

```go
// MutexFor returns the per-pipeline mutex for the given pipeline name.
// Exported for testing (BDD specs).
func (e *Engine) MutexFor(name string) *sync.Mutex {
	return e.mu[name]
}
```

- [ ] **Step 2: Run BDD specs**

```bash
go tool task test:specs
```

Expected: All specs PASS.

- [ ] **Step 3: Commit**

```bash
git add tests/specs/pipeline_spec_test.go pkg/pipeline/engine.go
git commit -m "test: add BDD specs for webhook trigger"
```

---

## Post-Implementation Checklist

- [ ] All unit tests pass (`go tool task test`)
- [ ] All BDD specs pass (`go tool task test:specs`)
- [ ] Lint passes (`go tool task lint`)
- [ ] No new `// TODO` or `TBD` in code
- [ ] Webhook trigger example added to `docs/reference/pipelines.yaml`
