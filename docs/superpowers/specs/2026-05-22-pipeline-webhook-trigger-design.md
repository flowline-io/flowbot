# Pipeline Webhook Trigger

Date: 2026-05-22

## Summary

Add HTTP webhook-based trigger to the pipeline engine so external systems can
invoke pipelines via HTTP POST requests to per-pipeline URLs.

## Motivation

Pipeline currently supports event-based triggers (`Trigger.Event`) and
cron-based triggers (`Trigger.Cron`). External systems need a direct HTTP
callback mechanism to trigger pipelines without going through the DataEvent
infrastructure (PostgreSQL event store + Redis Stream). Webhooks provide a
synchronous request-to-trigger path with immediate HTTP acknowledgement.

## Design

### Route model

Each webhook-enabled pipeline gets its own URL path, configured via
`trigger.webhook.path`. The server registers routes under `/webhook/{path}`.

A webhook trigger is mutually exclusive with event and cron triggers — a
pipeline cannot combine webhook with event or cron in the same trigger block.

### Config layer (`pkg/config/config.go`)

`PipelineTrigger` gains a `Webhook` field pointing to `*WebhookTrigger`:

```go
type PipelineTrigger struct {
    Event       string          `json:"event" yaml:"event" mapstructure:"event"`
    Cron        string          `json:"cron" yaml:"cron" mapstructure:"cron"`
    CronTimeout string          `json:"cron_timeout" yaml:"cron_timeout" mapstructure:"cron_timeout"`
    Webhook     *WebhookTrigger `json:"webhook" yaml:"webhook" mapstructure:"webhook"`
}

type WebhookTrigger struct {
    Path      string             `json:"path" yaml:"path" mapstructure:"path"`
    Method    string             `json:"method" yaml:"method" mapstructure:"method"`
    Auth      *WebhookAuth       `json:"auth" yaml:"auth" mapstructure:"auth"`
    Payload   WebhookPayloadMode `json:"payload" yaml:"payload" mapstructure:"payload"`
    EventType string             `json:"event_type" yaml:"event_type" mapstructure:"event_type"`
}

type WebhookAuth struct {
    Token      string `json:"token" yaml:"token" mapstructure:"token"`
    HMACSecret string `json:"hmac_secret" yaml:"hmac_secret" mapstructure:"hmac_secret"`
    HMACHeader string `json:"hmac_header" yaml:"hmac_header" mapstructure:"hmac_header"`
    TokenHeader string `json:"token_header" yaml:"token_header" mapstructure:"token_header"`
}

type WebhookPayloadMode string

const (
    WebhookPayloadRaw    WebhookPayloadMode = "raw"
    WebhookPayloadMapped WebhookPayloadMode = "mapped"
)
```

**Defaults**:
- `Method`: `"POST"` when empty
- `Payload`: `WebhookPayloadRaw` when empty
- `EventType`: `"webhook.{path}"` when empty
- `HMACHeader`: `"X-Hub-Signature-256"` when empty
- `TokenHeader`: `"X-Webhook-Token"` when empty

**Validation at LoadConfig time**:
- `Path` must be non-empty
- `Method` must be one of GET, POST, PUT (all uppercase)
- At least one of `Auth.Token` or `Auth.HMACSecret` must be set (no unauthenticated webhooks)
- `EventType` defaults to `"webhook.{path}"` if empty
- Webhook must not coexist with `event` or `cron` on the same trigger

**Enabled semantics**: `Pipeline.Enabled` controls ALL trigger types. An
`enabled: false` pipeline is never loaded by `LoadConfig` and is never
registered as a webhook route.

Example `pipelines.yaml`:

```yaml
- name: github_push_handler
  description: "Process GitHub push events"
  enabled: true
  resumable: true
  trigger:
    webhook:
      path: "github-push"
      method: POST
      auth:
        hmac_secret: "${GITHUB_WEBHOOK_SECRET}"
      payload: raw
  steps:
    - name: process_push
      capability: github
      operation: handle_push
      params:
        payload: "{{event.payload._webhook_body}}"

- name: n8n_callback
  description: "Handle n8n workflow callback"
  enabled: true
  trigger:
    webhook:
      path: "n8n-callback"
      method: POST
      auth:
        token: "${N8N_WEBHOOK_TOKEN}"
      payload: mapped
  steps:
    - name: handle_callback
      capability: n8n
      operation: callback
      params:
        workflow_id: "{{event.payload.workflow_id}}"
```

### Trigger model (`pkg/pipeline/loader.go`)

`Trigger` struct gains a `Webhook` field:

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

`convertTrigger()` maps all fields from config to runtime struct. Validation
is performed at this stage (path non-empty, method valid, auth present, no
mixed trigger).

### Engine layer (`pkg/pipeline/engine.go`)

**RegisterWebhooks** builds a path-to-definition map:

```go
func (e *Engine) RegisterWebhooks() (map[string]*Definition, error)
```

- Iterates all definitions, collects those with `Trigger.Webhook != nil`
- Validates path uniqueness across all webhook definitions — duplicate paths error
- Returns `map[string]*Definition` keyed by path

**ExecuteWebhook** is the direct dispatch entry point called by the HTTP handler:

```go
func (e *Engine) ExecuteWebhook(ctx context.Context, def *Definition, event types.DataEvent) error
```

Internal logic:
1. Acquires per-pipeline mutex (same mutex map shared with cron/event handlers)
2. Generates synthetic event ID: `"webhook:{path}:{unix-nano}-{randomHex(8)}"`
3. Calls `executePipeline(ctx, def, event)` — reuses the existing execution
   path including run records, checkpoints, metrics, and audit
4. Audit records tag source as `"webhook"`

Webhook does NOT go through Watermill/Redis Stream. It calls
`executePipeline` directly with a fabricated event, matching the cron
trigger pattern.

### Concurrency model

Webhook uses the same per-pipeline `sync.Mutex` map as event and cron
triggers. `ExecuteWebhook` calls `Lock()` (blocking) so webhook requests
queue up and execute sequentially for the same pipeline. A webhook arriving
while a cron run is in-flight for the same pipeline blocks until the cron
run completes, and vice versa.

### HTTP handler (`internal/server/webhook.go`)

New file implementing the webhook HTTP handler, registered at server startup:

```
{Method} /webhook/{path}
```

Handler flow:
1. Extract `{path}` from URL params
2. Lookup `path` in engine's webhook map — 404 if not found
3. Validate method matches configured method — 405 if mismatch (Allow header set)
4. Authenticate the request:
   - If `Auth.Token` is configured: compare against configured token.
     Token is read from header `X-Webhook-Token` (configurable via `Auth.TokenHeader`).
   - If `Auth.HMACSecret` is configured: compute HMAC-SHA256 of request body,
     compare against header `X-Hub-Signature-256` (configurable via
     `Auth.HMACHeader`). Expected format: `sha256=<hex>`.
   - If both are configured: either passing grants access (OR logic).
   - If neither is configured: return 401 (no unauthenticated webhooks).
5. Build `types.DataEvent`:
   - `EventID`: `"webhook:{path}:{nano}-{hex}"` (deferred to ExecuteWebhook)
   - `EventType`: configured `EventType` (default `"webhook.{path}"`)
   - `Source`: `"webhook"` (set by ExecuteWebhook)
   - **`raw` payload mode**: request body stored as string in
     `Data["_webhook_body"]`. All request headers stored as
     `map[string]string` in `Data["_webhook_headers"]`.
   - **`mapped` payload mode**: request body parsed as JSON into
     `map[string]any` and merged into `Data`. Headers still injected as
     `_webhook_headers`. If body is not valid JSON, return 400.
6. Call `engine.ExecuteWebhook(ctx, def, event)`
7. Return 202 Accepted (pipeline runs asynchronously)

### Headers injection

Regardless of payload mode, all incoming HTTP request headers are stored in
`DataEvent.Data["_webhook_headers"]` as `map[string]string`. This allows
pipeline steps to access headers like `{{event.payload._webhook_headers.X-GitHub-Event}}`.

### Error responses

| Scenario | Status | Detail |
|----------|--------|--------|
| Path not registered | 404 | No pipeline has the requested webhook path |
| Method mismatch | 405 | Allow header lists the configured method |
| Auth not configured on pipeline | 401 | Pipeline has no token or HMAC secret |
| Token mismatch | 401 | Header token does not match configured value |
| HMAC signature mismatch | 401 | Computed signature does not match header value |
| Payload=mapped, body not valid JSON | 400 | Failed to parse request body as JSON |
| Pipeline execution fails | 202 | Accepted; error is recorded in pipeline run, not in HTTP response |

### Auth verification details

**Token verification**:
```
configured_token == request_header_value
```
Constant-time comparison is not required (webhook tokens are not passwords;
the primary threat model is service-to-service authentication, not side-channel).

**HMAC verification**:
```
expected = hmacSHA256(configured_secret, request_body)
actual   = parse_header("sha256=<hex>")
expected_hex == actual
```
The header value must start with `sha256=` prefix. The hex string is
compared case-insensitively.

## Key decisions

- **Per-pipeline URL**: Each webhook pipeline gets a unique path under
  `/webhook/`. Path-to-pipeline mapping is 1:1.
- **Direct dispatch**: Webhook triggers call `ExecuteWebhook` directly on
  the engine without going through the DataEvent infrastructure. Matches
  the cron trigger pattern.
- **No mixed triggers**: Webhook cannot coexist with `event` or `cron` on
  the same trigger block. Validation at LoadConfig time enforces this.
- **Auth required**: At least one of token or HMAC secret must be
  configured. No unauthenticated webhooks.
- **Unified concurrency**: Per-pipeline mutex in `executePipeline` protects
  ALL trigger sources (event, cron, webhook).
- **Raw vs mapped payload**: Configurable per pipeline. Raw passes the body
  as a string; mapped parses it as JSON.
- **All headers injected**: `_webhook_headers` is always available in
  DataEvent.Data, enabling steps to inspect HTTP metadata.
- **202 Accepted**: HTTP response is sent immediately after queueing the
  pipeline. Execution is asynchronous.
- **No HTTP response body passthrough**: The HTTP response is always a
  simple 202; pipeline output is not returned in the webhook response.

## Known limitations

- Pipelines are static after `NewEngine`. Adding/removing webhook pipelines
  requires a server restart.
- No request size limit is enforced beyond Fiber's default body limit.
- No retry from the webhook side — if a webhook is accepted (202) but the
  pipeline fails, the external system must retry the webhook call.
- No request validation schema per pipeline. Validation (beyond JSON parse
  in mapped mode) is left to pipeline steps.

## Testing

### Unit tests (TDD, table-driven)

All test functions use `for _, tt := range tests { t.Run(tt.name, ...) }`
with at least 3 cases per table. Happy path first, error cases required.

**`pkg/config/config_test.go`** — parse webhook fields from YAML:

- `tt.name`: "webhook complete config", "webhook token auth only", "webhook HMAC auth only", "webhook both auth", "webhook defaults only path", "webhook missing path", "webhook invalid method"
- verify `WebhookTrigger.Path`, `Method`, `Auth.Token`, `Auth.HMACSecret`, `Payload`, `EventType` parsed correctly
- verify defaults applied for empty fields

**`pkg/pipeline/pipeline_test.go`** — `LoadConfig` maps webhook and validates:

- `tt.name`: "webhook valid definition", "webhook with cron errors", "webhook with event errors", "webhook duplicate paths", "webhook empty path", "webhook defaults applied", "disabled webhook pipeline"
- verify `WebhookConfig` populated correctly
- verify mixed trigger validation rejects webhook+cron and webhook+event
- verify duplicate paths are rejected at RegisterWebhooks
- verify disabled pipelines are not loaded

**`pkg/pipeline/engine_test.go`** — webhook engine behavior:

- `tt.name`: "RegisterWebhooks returns paths", "RegisterWebhooks duplicate path errors", "ExecuteWebhook success", "ExecuteWebhook mutex serialization", "ExecuteWebhook event ID format", "ExecuteWebhook with raw payload", "ExecuteWebhook with mapped payload", "ExecuteWebhook audit source"
- verify `RegisterWebhooks` returns correct path→def mapping
- verify `ExecuteWebhook` runs pipeline and generates correct event structure
- verify raw mode stores body in `_webhook_body`
- verify mapped mode merges JSON fields into Data
- verify event ID format `webhook:{path}:{nano}-{hex}`
- verify audit source is `"webhook"`
- verify per-pipeline mutex serializes concurrent webhook calls

**`internal/server/webhook_test.go`** — HTTP integration tests (Fiber test package):

- `tt.name`: "valid webhook token auth returns 202", "valid webhook HMAC auth returns 202", "unknown path returns 404", "wrong method returns 405", "token mismatch returns 401", "HMAC mismatch returns 401", "no auth configured returns 401", "invalid JSON in mapped mode returns 400", "valid JSON in mapped mode returns 202", "raw mode preserves body", "headers injected into DataEvent", "cron-only pipeline not in webhook map"

### BDD specs (Ginkgo v2 + Gomega)

`tests/specs/pipeline_spec_test.go` — `Describe("Webhook trigger")`:

- `It("executes pipeline on webhook POST")`: send POST with valid token, assert pipeline runs complete, assert 202 response
- `It("rejects unauthenticated webhook when no auth configured")`: pipeline without token/hmac, assert 401
- `It("rejects request with wrong HMAC signature")`: send with incorrect HMAC, assert 401
- `It("blocks concurrent webhook runs for same pipeline")`: send two requests rapidly, assert second blocks and executes after first
- `It("passes raw body to pipeline steps")`: send raw text body, assert `{{event.payload._webhook_body}}` equals sent body
- `It("parses JSON body in mapped mode")`: send `{"key":"val"}`, assert `{{event.payload.key}}` equals `"val"`
- `It("injects HTTP headers into DataEvent")`: send with custom header `X-Custom: test`, assert `_webhook_headers.X-Custom` is `"test"`
- `It("records DataEvent with correct webhook source")`: assert run record has `source` = `"webhook"`

## Files affected

| File | Change |
|------|--------|
| `pkg/config/config.go` | Add `WebhookTrigger`, `WebhookAuth`, `WebhookPayloadMode` types; add `Webhook` field to `PipelineTrigger` |
| `pkg/pipeline/loader.go` | Add `WebhookConfig`, `WebhookAuthConfig` types to `Trigger`; extend `convertTrigger` with webhook validation |
| `pkg/pipeline/engine.go` | Add `RegisterWebhooks()` and `ExecuteWebhook()` methods |
| `internal/server/webhook.go` | New file: HTTP handler for webhook requests |
| `internal/server/pipeline.go` | Register webhook routes at startup via `RegisterWebhooks` |
| `pkg/config/config_test.go` | Webhook YAML parse tests |
| `pkg/pipeline/pipeline_test.go` | Webhook LoadConfig and RegisterWebhooks tests |
| `pkg/pipeline/engine_test.go` | ExecuteWebhook behavior tests |
| `internal/server/webhook_test.go` | New file: HTTP handler integration tests |
| `tests/specs/pipeline_spec_test.go` | BDD webhook trigger specs |
| `docs/reference/pipelines.yaml` | Webhook trigger example |
