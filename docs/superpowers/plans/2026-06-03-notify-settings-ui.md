# Notify Settings UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a database-backed notification settings UI with channel management and rule editing, replacing `flowbot.yaml` notify rules as the sole source of truth.

**Architecture:** Two new Ent schemas (`notify_channel`, `notify_rule`) with store CRUD. Rules engine gains `Reload()` for hot-reload and validation helpers. Templ-based UI follows existing Configs page pattern with tabbed layout, inline form rows, and HTMX partial swaps.

**Tech Stack:** Ent ORM, Go 1.26+, templ v0.3, DaisyUI v5 + Tailwind v4 (CDN), HTMX 2.x, Alpine.js 3.x

---

## File Structure Map

```
internal/store/ent/schema/
  notify_channel.go       [CREATE] - Ent schema for notification channels
  notify_rule.go          [CREATE] - Ent schema for notification rules

pkg/types/model/
  notify_channel.go       [CREATE] - UI model types for NotifyChannel
  notify_rule.go          [CREATE] - UI model types for NotifyRule

internal/store/
  store.go                [MODIFY] - Add 10 adapter interface methods
  postgres/adapter.go     [MODIFY] - Implement 10 adapter methods + masking helpers

pkg/notify/rules/
  engine.go               [MODIFY] - Add Reload(), ValidateCondition()
pkg/notify/template/
  engine.go               [MODIFY] - Add ListTemplateIDs(), HasTemplate()
internal/server/
  notify.go               [MODIFY] - Load rules from DB on startup

internal/modules/web/
  notify_settings_webservice.go  [CREATE] - Route rules + handler functions
  module.go                [MODIFY] - Register new webservice rules

pkg/views/pages/
  notify_settings.templ    [CREATE] - Tabbed page wrapper
pkg/views/partials/
  notify_settings_helpers.go     [CREATE] - URL builders, maskURI helper
  notify_channels_table.templ    [CREATE] - Channel table partial
  notify_channel_row.templ       [CREATE] - Channel row partial
  notify_channel_form.templ      [CREATE] - Channel form partial
  notify_rules_table.templ       [CREATE] - Rule table partial
  notify_rule_row.templ          [CREATE] - Rule row partial
  notify_rule_form.templ         [CREATE] - Rule form partial
pkg/views/layout/
  base.templ               [MODIFY] - Add navbar link
```

---

### Task 1: Create Ent schema for notify_channel

**Files:**
- Create: `internal/store/ent/schema/notify_channel.go`

- [ ] **Step 1: Write the schema file**

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NotifyChannel struct {
	ent.Schema
}

func (NotifyChannel) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").Unique().NotEmpty(),
		field.String("protocol").NotEmpty(),
		field.String("uri").NotEmpty(),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NotifyChannel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("protocol"),
		index.Fields("enabled"),
	}
}

func (NotifyChannel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("notify_channels"),
	}
}
```

- [ ] **Step 2: Verify file exists**

Run: `ls -la internal/store/ent/schema/notify_channel.go`

---

### Task 2: Create Ent schema for notify_rule

**Files:**
- Create: `internal/store/ent/schema/notify_rule.go`

- [ ] **Step 1: Write the schema file**

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NotifyRule struct {
	ent.Schema
}

func (NotifyRule) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("rule_id").Unique().NotEmpty(),
		field.String("name").NotEmpty(),
		field.Enum("action").Values("throttle", "aggregate", "mute", "drop"),
		field.String("event_pattern").Default("*").NotEmpty(),
		field.String("channel_pattern").Default("*").NotEmpty(),
		field.String("condition").Optional(),
		field.Int("priority").Default(0),
		field.JSON("params", map[string]any{}),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NotifyRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("priority"),
		index.Fields("enabled"),
	}
}

func (NotifyRule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("notify_rules"),
	}
}
```

- [ ] **Step 2: Verify file exists**

Run: `ls -la internal/store/ent/schema/notify_rule.go`

---

### Task 3: Run ent code generation

**Files:**
- Generate: `internal/store/ent/gen/` (auto-generated, do not edit)

- [ ] **Step 1: Run ent generate**

Run: `go tool task ent`

Expected: No errors. New files under `internal/store/ent/gen/` include `notifychannel.go`, `notifyrule.go`, and related query/builders.

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`

Expected: No errors (unused import/symbol warnings are OK here since we haven't consumed the types yet).

- [ ] **Step 3: Commit**

```bash
git add internal/store/ent/schema/notify_channel.go internal/store/ent/schema/notify_rule.go internal/store/ent/gen/
git commit -m "feat: add notify_channel and notify_rule Ent schemas"
```

---

### Task 4: Create model types for UI

**Files:**
- Create: `pkg/types/model/notify_channel.go`
- Create: `pkg/types/model/notify_rule.go`

- [ ] **Step 1: Write NotifyChannel model**

```go
// Package model provides shared data types for UI views and transport.
package model

import "time"

// NotifyChannel represents a configured notification channel for UI display.
// The URI is masked for display; raw URI is never exposed to the client.
type NotifyChannel struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Protocol  string    `json:"protocol"`
	URI       string    `json:"uri"` // masked for display
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Write NotifyRule model**

```go
package model

import "time"

// NotifyRule represents a notification routing rule for UI display and editing.
type NotifyRule struct {
	ID             int64     `json:"id"`
	RuleID         string    `json:"rule_id"`
	Name           string    `json:"name"`
	Action         string    `json:"action"`
	EventPattern   string    `json:"event_pattern"`
	ChannelPattern string    `json:"channel_pattern"`
	Condition      string    `json:"condition"`
	Priority       int       `json:"priority"`
	ParamsJSON     string    `json:"params_json"` // JSON string for form display
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/types/model/...`

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/types/model/notify_channel.go pkg/types/model/notify_rule.go
git commit -m "feat: add NotifyChannel and NotifyRule UI model types"
```

---

### Task 5: Add store interface methods

**Files:**
- Modify: `internal/store/store.go`

- [ ] **Step 1: Add ListNotifyChannelOptions and ListNotifyRuleOptions types**

Find the existing option structs section (near `ListConfigOptions`), add after them:

```go
// ListNotifyChannelOptions holds filtering options for listing notification channels.
type ListNotifyChannelOptions struct {
	Protocol string
	Enabled  *bool // nil = all, true = enabled only, false = disabled only
}

// ListNotifyRuleOptions holds filtering and sorting options for listing notification rules.
type ListNotifyRuleOptions struct {
	Enabled *bool // nil = all, true = enabled only, false = disabled only
}
```

- [ ] **Step 2: Add interface methods**

Find the `Adapter` interface, add after the config section or at the end before closing `}`:

```go
	// NotifyChannel CRUD
	CreateNotifyChannel(ctx context.Context, name, protocol, uri string) (int64, error)
	GetNotifyChannel(ctx context.Context, id int64) (model.NotifyChannel, error)     // returns masked URI
	GetNotifyChannelRaw(ctx context.Context, id int64) (model.NotifyChannel, error)   // returns raw URI (internal use only)
	ListNotifyChannels(ctx context.Context, opts ListNotifyChannelOptions) ([]model.NotifyChannel, error)
	UpdateNotifyChannel(ctx context.Context, id int64, name, protocol, uri string, enabled bool) error
	DeleteNotifyChannel(ctx context.Context, id int64) error

	// NotifyRule CRUD
	CreateNotifyRule(ctx context.Context, rule model.NotifyRule) (int64, error)
	GetNotifyRule(ctx context.Context, id int64) (model.NotifyRule, error)
	ListNotifyRules(ctx context.Context, opts ListNotifyRuleOptions) ([]model.NotifyRule, error)
	UpdateNotifyRule(ctx context.Context, id int64, rule model.NotifyRule) error
	DeleteNotifyRule(ctx context.Context, id int64) error

	// Notify URI masking
	MaskNotifyURI(protocol, uri string) string
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./...`

Expected: Errors in `postgres/adapter.go` since methods aren't implemented yet. That's expected.

- [ ] **Step 4: Commit**

```bash
git add internal/store/store.go
git commit -m "feat: add NotifyChannel and NotifyRule store interface methods"
```

---

### Task 6: Implement channel CRUD in postgres adapter

**Files:**
- Modify: `internal/store/postgres/adapter.go`

- [ ] **Step 1: Add imports at top of file**

Add these imports to the existing import block:
```go
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/notifychannel"
	"github.com/flowline-io/flowbot/pkg/types/model"
```

(Make sure `fmt`, `strings`, `time` are already imported; if not add them. Check existing imports first.)

- [ ] **Step 2: Add channel CRUD implementations**

Add the following block near the bottom of the file (before the closing of any type or at the end of the adapter methods):

```go
// ---------------------------------------------------------------------------
// NotifyChannel CRUD
// ---------------------------------------------------------------------------

func (a *adapter) CreateNotifyChannel(ctx context.Context, name, protocol, uri string) (int64, error) {
	ch, err := a.client.NotifyChannel.Create().
		SetName(name).
		SetProtocol(protocol).
		SetURI(uri).
		SetEnabled(true).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create notify channel: %w", err)
	}
	return ch.ID, nil
}

func (a *adapter) GetNotifyChannel(ctx context.Context, id int64) (model.NotifyChannel, error) {
	ch, err := a.client.NotifyChannel.Query().Where(notifychannel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyChannel{}, types.ErrNotFound
		}
		return model.NotifyChannel{}, fmt.Errorf("postgres: get notify channel: %w", err)
	}
	return model.NotifyChannel{
		ID:        ch.ID,
		Name:      ch.Name,
		Protocol:  ch.Protocol,
		URI:       a.MaskNotifyURI(ch.Protocol, ch.URI),
		Enabled:   ch.Enabled,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}, nil
}

func (a *adapter) GetNotifyChannelRaw(ctx context.Context, id int64) (model.NotifyChannel, error) {
	ch, err := a.client.NotifyChannel.Query().Where(notifychannel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyChannel{}, types.ErrNotFound
		}
		return model.NotifyChannel{}, fmt.Errorf("postgres: get notify channel raw: %w", err)
	}
	return model.NotifyChannel{
		ID:        ch.ID,
		Name:      ch.Name,
		Protocol:  ch.Protocol,
		URI:       ch.URI, // raw, unmasked
		Enabled:   ch.Enabled,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}, nil
}

func (a *adapter) ListNotifyChannels(ctx context.Context, opts ListNotifyChannelOptions) ([]model.NotifyChannel, error) {
	q := a.client.NotifyChannel.Query()
	if opts.Protocol != "" {
		q = q.Where(notifychannel.Protocol(opts.Protocol))
	}
	if opts.Enabled != nil {
		q = q.Where(notifychannel.Enabled(*opts.Enabled))
	}
	chs, err := q.Order(gen.Asc(notifychannel.FieldName)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list notify channels: %w", err)
	}
	result := make([]model.NotifyChannel, len(chs))
	for i, ch := range chs {
		result[i] = model.NotifyChannel{
			ID:        ch.ID,
			Name:      ch.Name,
			Protocol:  ch.Protocol,
			URI:       a.MaskNotifyURI(ch.Protocol, ch.URI),
			Enabled:   ch.Enabled,
			CreatedAt: ch.CreatedAt,
			UpdatedAt: ch.UpdatedAt,
		}
	}
	return result, nil
}

func (a *adapter) UpdateNotifyChannel(ctx context.Context, id int64, name, protocol, uri string, enabled bool) error {
	upd := a.client.NotifyChannel.Update().Where(notifychannel.IDEQ(id)).
		SetName(name).
		SetProtocol(protocol).
		SetEnabled(enabled).
		SetUpdatedAt(time.Now())
	if uri != "" {
		upd = upd.SetURI(uri)
	}
	n, err := upd.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update notify channel: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) DeleteNotifyChannel(ctx context.Context, id int64) error {
	_, err := a.client.NotifyChannel.Delete().Where(notifychannel.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete notify channel: %w", err)
	}
	return nil
}

// MaskNotifyURI produces a display-safe masked form of a notification URI.
// For slack: slack://hooks.slack.com/services/T******/B******/C******
// For ntfy: http://ntfy.example.com/******
// For pushover: pushover://U******@A******
// For message-pusher: message-pusher://user@domain/******/******
func (a *adapter) MaskNotifyURI(protocol, uri string) string {
	switch protocol {
	case "slack":
		return maskSlackURI(uri)
	case "ntfy":
		return maskNtfyURI(uri)
	case "pushover":
		return maskPushoverURI(uri)
	case "message-pusher":
		return maskMessagePusherURI(uri)
	default:
		if len(uri) > 30 {
			return uri[:27] + "..."
		}
		return uri
	}
}

func maskSlackURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "slack://******"
	}
	pathParts := strings.Split(parts[1], "/")
	if len(pathParts) > 3 {
		pathParts[len(pathParts)-3] = "T******"
	}
	if len(pathParts) > 2 {
		pathParts[len(pathParts)-2] = "B******"
	}
	if len(pathParts) > 1 {
		pathParts[len(pathParts)-1] = "C******"
	}
	return parts[0] + "://" + strings.Join(pathParts, "/")
}

func maskNtfyURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "ntfy://******"
	}
	hostParts := strings.SplitN(parts[1], "/", 2)
	if len(hostParts) < 2 {
		return parts[0] + "://" + hostParts[0] + "/******"
	}
	return parts[0] + "://" + hostParts[0] + "/******"
}

func maskPushoverURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "pushover://******"
	}
	userIdx := strings.Index(parts[1], "@")
	if userIdx < 0 {
		return parts[0] + "://U******@" + maskEnd(parts[1])
	}
	return parts[0] + "://U******@A******"
}

func maskMessagePusherURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "message-pusher://******"
	}
	atIdx := strings.Index(parts[1], "@")
	if atIdx < 0 {
		return parts[0] + "://******"
	}
	finalSlash := strings.LastIndex(parts[1], "/")
	if finalSlash < 0 {
		return parts[0] + "://" + parts[1][:atIdx+1] + "domain/******/******"
	}
	secondLast := strings.LastIndex(parts[1][:finalSlash], "/")
	if secondLast < 0 {
		return parts[0] + "://" + parts[1][:finalSlash+1] + "******"
	}
	return parts[0] + "://" + parts[1][:secondLast+1] + "******/******"
}

func maskEnd(s string) string {
	if len(s) > 8 {
		return s[:4] + "******"
	}
	return "******"
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/store/...`

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add internal/store/postgres/adapter.go
git commit -m "feat: implement NotifyChannel CRUD and URI masking in postgres adapter"
```

---

### Task 7: Implement rule CRUD in postgres adapter

**Files:**
- Modify: `internal/store/postgres/adapter.go`

- [ ] **Step 1: Add rule CRUD implementations**

Add after the channel CRUD block:

```go
// ---------------------------------------------------------------------------
// NotifyRule CRUD
// ---------------------------------------------------------------------------

func (a *adapter) CreateNotifyRule(ctx context.Context, rule model.NotifyRule) (int64, error) {
	var params map[string]any
	if rule.ParamsJSON != "" {
		if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
			return 0, fmt.Errorf("postgres: create notify rule params parse: %w", err)
		}
	} else {
		params = map[string]any{}
	}
	r, err := a.client.NotifyRule.Create().
		SetRuleID(rule.RuleID).
		SetName(rule.Name).
		SetAction(notifyrule.Action(rule.Action)).
		SetEventPattern(rule.EventPattern).
		SetChannelPattern(rule.ChannelPattern).
		SetNillableCondition(nilString(rule.Condition)).
		SetPriority(rule.Priority).
		SetParams(params).
		SetEnabled(rule.Enabled).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create notify rule: %w", err)
	}
	return r.ID, nil
}

func (a *adapter) GetNotifyRule(ctx context.Context, id int64) (model.NotifyRule, error) {
	r, err := a.client.NotifyRule.Query().Where(notifyrule.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyRule{}, types.ErrNotFound
		}
		return model.NotifyRule{}, fmt.Errorf("postgres: get notify rule: %w", err)
	}
	return notifyRuleToModel(r), nil
}

func (a *adapter) ListNotifyRules(ctx context.Context, opts ListNotifyRuleOptions) ([]model.NotifyRule, error) {
	q := a.client.NotifyRule.Query()
	if opts.Enabled != nil {
		q = q.Where(notifyrule.Enabled(*opts.Enabled))
	}
	rules, err := q.Order(gen.Desc(notifyrule.FieldPriority)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list notify rules: %w", err)
	}
	result := make([]model.NotifyRule, len(rules))
	for i, r := range rules {
		result[i] = notifyRuleToModel(r)
	}
	return result, nil
}

func (a *adapter) UpdateNotifyRule(ctx context.Context, id int64, rule model.NotifyRule) error {
	var params map[string]any
	if rule.ParamsJSON != "" {
		if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
			return fmt.Errorf("postgres: update notify rule params parse: %w", err)
		}
	} else {
		params = map[string]any{}
	}
	n, err := a.client.NotifyRule.Update().Where(notifyrule.IDEQ(id)).
		SetRuleID(rule.RuleID).
		SetName(rule.Name).
		SetAction(notifyrule.Action(rule.Action)).
		SetEventPattern(rule.EventPattern).
		SetChannelPattern(rule.ChannelPattern).
		SetNillableCondition(nilString(rule.Condition)).
		SetPriority(rule.Priority).
		SetParams(params).
		SetEnabled(rule.Enabled).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update notify rule: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *adapter) DeleteNotifyRule(ctx context.Context, id int64) error {
	_, err := a.client.NotifyRule.Delete().Where(notifyrule.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete notify rule: %w", err)
	}
	return nil
}

func notifyRuleToModel(r *gen.NotifyRule) model.NotifyRule {
	var cond string
	if r.Condition != nil {
		cond = *r.Condition
	}
	paramsJSON, _ := sonic.MarshalString(r.Params)
	return model.NotifyRule{
		ID:             r.ID,
		RuleID:         r.RuleID,
		Name:           r.Name,
		Action:         string(r.Action),
		EventPattern:   r.EventPattern,
		ChannelPattern: r.ChannelPattern,
		Condition:      cond,
		Priority:       r.Priority,
		ParamsJSON:     paramsJSON,
		Enabled:        r.Enabled,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
```

- [ ] **Step 2: Ensure required imports exist**

The import block needs:
```go
	"github.com/bytedance/sonic"
	gennotifyrule "github.com/flowline-io/flowbot/internal/store/ent/gen/notifyrule"
```

Add `sonic` if not already imported. Use a different alias for `notifyrule` if it conflicts; check existing import aliases in the file. If the variable `notifyrule` in the generated code is a package qualifier, use `notifyrule` directly without gen prefix.

Actually, the generated ent code uses `notifychannel` and `notifyrule` as package qualifiers. Import them as:
```go
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notifychannel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notifyrule"
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/store/...`

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add internal/store/postgres/adapter.go
git commit -m "feat: implement NotifyRule CRUD in postgres adapter"
```

---

### Task 8: Add Reload() and ValidateCondition to rules engine

**Files:**
- Modify: `pkg/notify/rules/engine.go`

- [ ] **Step 1: Check that sync.RWMutex is already present**

The engine struct already has `mu sync.RWMutex` and `LoadConfig` already uses `e.mu.Lock()`. No change needed for concurrency — it's already safe.

- [ ] **Step 2: Add Reload method**

Add after the `LoadConfig` method:

```go
// Reload refreshes the rule list from the database.
// Called after rule CRUD operations to enable hot-reload without restart.
func (e *Engine) Reload(ctx context.Context, loader func(context.Context) ([]config.NotifyRule, error)) error {
	rules, err := loader(ctx)
	if err != nil {
		return err
	}
	return e.LoadConfig(rules)
}
```

- [ ] **Step 3: Add ValidateCondition function**

Add after `evalTimeCondition`:

```go
// ValidateCondition checks whether a condition expression string is syntactically valid.
// It uses the same parsing logic as evalCondition but without evaluating time values.
func ValidateCondition(condition string) error {
	if condition == "" {
		return nil
	}
	// validate each || part
	parts := strings.SplitSeq(condition, "||")
	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("rules: empty expression after ||")
		}
		andParts := strings.SplitSeq(part, "&&")
		for ap := range andParts {
			ap = strings.TrimSpace(ap)
			if ap == "" {
				return fmt.Errorf("rules: empty expression after &&")
			}
			if err := validateTimeExpression(ap); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTimeExpression(expr string) error {
	expr = strings.TrimSpace(expr)
	if !strings.HasPrefix(expr, "time.hour ") {
		return fmt.Errorf("rules: expected 'time.hour <op> N', got %q", expr)
	}
	for _, op := range []string{">=", "<=", "==", ">", "<"} {
		if strings.Contains(expr, " "+op+" ") || strings.HasPrefix(expr[len("time.hour "):], op+" ") {
			return nil
		}
	}
	return fmt.Errorf("rules: unknown operator in %q", expr)
}
```

- [ ] **Step 4: Add `fmt` and `strings` to imports if missing**

Check imports block for `"fmt"` and `"strings"`. Add if not present.

- [ ] **Step 5: Verify compilation and run existing tests**

Run: `go build ./pkg/notify/rules/... && go test ./pkg/notify/rules/... -v`
Expected: Existing tests pass, no compilation errors.

- [ ] **Step 6: Commit**

```bash
git add pkg/notify/rules/engine.go
git commit -m "feat: add Reload() and ValidateCondition to rules engine"
```

---

### Task 9: Add ListTemplateIDs() to template engine

**Files:**
- Modify: `pkg/notify/template/engine.go`

- [ ] **Step 1: Add ListTemplateIDs and HasTemplate methods**

Add after `GetTemplateID`:

```go
// ListTemplateIDs returns all registered template IDs in the engine.
func (e *Engine) ListTemplateIDs() []string {
	ids := make([]string, 0, len(e.templates))
	for id := range e.templates {
		ids = append(ids, id)
	}
	return ids
}

// HasTemplate returns true if the given template ID exists in the engine.
func (e *Engine) HasTemplate(id string) bool {
	_, ok := e.templates[id]
	return ok
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/notify/template/...`

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/notify/template/engine.go
git commit -m "feat: add ListTemplateIDs and HasTemplate to template engine"
```

---

### Task 10: Update server notify.go to load rules from DB

**Files:**
- Modify: `internal/server/notify.go`

- [ ] **Step 1: Read the current file to understand existing wiring**

The current `initNotificationGateway` calls `notifytmpl.Init()` then `notifyrules.Init(store)`. We need to replace the `notifyrules.Init(store)` call with loading rules from DB.

Check what `notifyrules.Init` does first:

```go
// Look at pkg/notify/rules/engine.go or a loader file for Init function
```

- [ ] **Step 2: Check if rules package has an Init function**

Run: `grep -n "func Init" pkg/notify/rules/*.go`

If it exists, read its implementation to understand how to adapt it.

- [ ] **Step 3: Update initNotificationGateway**

Replace the current function to load rules from DB instead of YAML config:

```go
func initNotificationGateway(lc fx.Lifecycle, store *cache.RedisStore) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := notifytmpl.Init(); err != nil {
				return err
			}

			engine := notifyrules.GetEngine()
			if engine == nil {
				engine = notifyrules.New(store)
			}

			enabled := true
			rules, err := storedb.Database.ListNotifyRules(ctx, store.ListNotifyRuleOptions{Enabled: &enabled})
			if err != nil {
				flog.Error("failed to load notify rules from DB: %v", err)
			} else {
				configRules := make([]config.NotifyRule, len(rules))
				for i, r := range rules {
					var cond string
					if r.Condition != "" {
						cond = r.Condition
					}
					var params config.NotifyRuleParams
					if r.ParamsJSON != "" {
						if err := sonic.Unmarshal([]byte(r.ParamsJSON), &params); err != nil {
							flog.Error("skipping notify rule %s: invalid params JSON: %v", r.RuleID, err)
							continue
						}
					}
					configRules[i] = config.NotifyRule{
						ID:     r.RuleID,
						Action: config.NotifyRuleAction(r.Action),
						Match: config.NotifyRuleMatch{
							Event:   r.EventPattern,
							Channel: r.ChannelPattern,
						},
						Condition: cond,
						Priority:  r.Priority,
						Params:    params,
					}
				}
				if err := engine.LoadConfig(configRules); err != nil {
					return err
				}
			}

			return abilitynotify.Register()
		},
		OnStop: func(_ context.Context) error {
			return nil
		},
	})
}
```

- [ ] **Step 4: Add required imports**

```go
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	storedb "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
```

Note: Check existing import aliases and adjust accordingly. `store` package is likely already imported. If a naming conflict exists, use an alias.

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/server/...`

Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add internal/server/notify.go
git commit -m "feat: load notify rules from database on startup"
```

---

### Task 11: Create notify settings webservice handlers

**Files:**
- Create: `internal/modules/web/notify_settings_webservice.go`

- [ ] **Step 1: Write the full webservice file**

```go
package web

import (
	"context"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var notifySettingsWebserviceRules = []webservice.Rule{
	webservice.Get("/notify-settings", notifySettingsPage, route.WithNotAuth()),
	webservice.Get("/notify-settings/channels/list", notifyChannelsTable, route.WithNotAuth()),
	webservice.Get("/notify-settings/channels/new", notifyChannelNewForm, route.WithNotAuth()),
	webservice.Post("/notify-settings/channels", notifyChannelCreate, route.WithNotAuth()),
	webservice.Get("/notify-settings/channels/:id/edit", notifyChannelEditForm, route.WithNotAuth()),
	webservice.Put("/notify-settings/channels/:id", notifyChannelUpdate, route.WithNotAuth()),
	webservice.Delete("/notify-settings/channels/:id", notifyChannelDelete, route.WithNotAuth()),
	webservice.Post("/notify-settings/channels/:id/test", notifyChannelTest, route.WithNotAuth()),
	webservice.Get("/notify-settings/rules/list", notifyRulesTable, route.WithNotAuth()),
	webservice.Get("/notify-settings/rules/new", notifyRuleNewForm, route.WithNotAuth()),
	webservice.Post("/notify-settings/rules", notifyRuleCreate, route.WithNotAuth()),
	webservice.Get("/notify-settings/rules/:id/edit", notifyRuleEditForm, route.WithNotAuth()),
	webservice.Put("/notify-settings/rules/:id", notifyRuleUpdate, route.WithNotAuth()),
	webservice.Delete("/notify-settings/rules/:id", notifyRuleDelete, route.WithNotAuth()),
}

func notifySettingsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return pages.NotifySettingsPage().Render(ctx.Context(), ctx.Response().BodyWriter())
}

// ---------------------------------------------------------------------------
// Channel handlers
// ---------------------------------------------------------------------------

func notifyChannelsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	channels, err := store.Database.ListNotifyChannels(ctx.Context(), store.ListNotifyChannelOptions{})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load channels").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyChannelsTable(channels).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	protocols := []string{}
	for id := range notifypkg.List() {
		protocols = append(protocols, id)
	}
	ctx.Type("html")
	return partials.NotifyChannelForm(model.NotifyChannel{}, true, nil, protocols).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	name := ctx.FormValue("name")
	protocol := ctx.FormValue("protocol")
	uri := ctx.FormValue("uri")
	errors := validateChannelForm(name, protocol, uri)
	if len(errors) > 0 {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(model.NotifyChannel{Name: name, Protocol: protocol}, true, errors, protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := store.Database.CreateNotifyChannel(ctx.Context(), name, protocol, uri)
	if err != nil {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(model.NotifyChannel{Name: name, Protocol: protocol}, true,
			map[string]string{"_save": err.Error()}, protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ch, err := store.Database.GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Channel created but failed to load").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyChannelRow(ch).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelEditForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	ch, err := store.Database.GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	protocols := getProtocolsList()
	ctx.Type("html")
	return partials.NotifyChannelForm(ch, false, nil, protocols).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelUpdate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	name := ctx.FormValue("name")
	protocol := ctx.FormValue("protocol")
	uri := ctx.FormValue("uri")
	enabled := ctx.FormValue("enabled") == "on"
	errors := validateChannelForm(name, protocol, "")
	if len(errors) > 0 {
		protocols := getProtocolsList()
		ctx.Type("html")
		return partials.NotifyChannelForm(model.NotifyChannel{ID: id, Name: name, Protocol: protocol}, false, errors, protocols).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := store.Database.UpdateNotifyChannel(ctx.Context(), id, name, protocol, uri, enabled); err != nil {
		return storeError(ctx, err)
	}
	ch, err := store.Database.GetNotifyChannel(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	return partials.NotifyChannelRow(ch).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyChannelDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	if err := store.Database.DeleteNotifyChannel(ctx.Context(), id); err != nil {
		return storeError(ctx, err)
	}
	return ctx.SendString("")
}

func notifyChannelTest(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	ch, err := store.Database.GetNotifyChannelRaw(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	uid := getUID(ctx)
	if uid == "" {
		uid = "system"
	}
	notifyMsg := notifypkg.Message{
		Title:    "Test Notification",
		Body:     "Connectivity test from Flowbot",
		Priority: notifypkg.Low,
	}
	notifyURI := ch.Protocol + "://" + ch.URI
	// Strip protocol prefix if URI already has it
	if !strings.Contains(ch.URI, "://") {
		notifyURI = ch.Protocol + "://" + ch.URI
	} else {
		notifyURI = ch.URI
	}
	if err := notifypkg.Send(notifyURI, notifyMsg); err != nil {
		ctx.Set("HX-Trigger", `{"showToast": {"type": "error", "message": "Connection failed: `+err.Error()+`"}}`)
		ns := notifypkg.GetNotifyStore()
		if ns != nil {
			_, _ = ns.Record(ctx.Context(), uid, ch.Name, "test", "failed", err.Error(), nil)
		}
		return ctx.SendString("")
	}
	ctx.Set("HX-Trigger", `{"showToast": {"type": "success", "message": "Connection successful"}}`)
	ns := notifypkg.GetNotifyStore()
	if ns != nil {
		_, _ = ns.Record(ctx.Context(), uid, ch.Name, "test", "success", "", nil)
	}
	return ctx.SendString("")
}

// ---------------------------------------------------------------------------
// Rule handlers
// ---------------------------------------------------------------------------

func notifyRulesTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rules, err := store.Database.ListNotifyRules(ctx.Context(), store.ListNotifyRuleOptions{})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load rules").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	templateIDs := []string{}
	if eng := notifytmpl.GetEngine(); eng != nil {
		templateIDs = eng.ListTemplateIDs()
	}
	ctx.Type("html")
	return partials.NotifyRulesTable(rules, templateIDs).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleNewForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	templateIDs := []string{}
	if eng := notifytmpl.GetEngine(); eng != nil {
		templateIDs = eng.ListTemplateIDs()
	}
	ctx.Type("html")
	return partials.NotifyRuleForm(model.NotifyRule{}, true, nil, templateIDs).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleCreate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rule := parseRuleForm(ctx)
	templateIDs := getTemplateIDs()
	errors := validateRuleForm(rule)
	if len(errors) > 0 {
		ctx.Type("html")
		return partials.NotifyRuleForm(rule, true, errors, templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	id, err := store.Database.CreateNotifyRule(ctx.Context(), rule)
	if err != nil {
		ctx.Type("html")
		return partials.NotifyRuleForm(rule, true, map[string]string{"_save": err.Error()}, templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	reloadRulesEngine(ctx.Context())
	r, err := store.Database.GetNotifyRule(ctx.Context(), id)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Rule created but failed to load").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	ctx.Type("html")
	return partials.NotifyRuleRow(r, templateIDs).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleEditForm(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	rule, err := store.Database.GetNotifyRule(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	templateIDs := getTemplateIDs()
	ctx.Type("html")
	return partials.NotifyRuleForm(rule, false, nil, templateIDs).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleUpdate(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	rule := parseRuleForm(ctx)
	templateIDs := getTemplateIDs()
	errors := validateRuleForm(rule)
	if len(errors) > 0 {
		ctx.Type("html")
		return partials.NotifyRuleForm(rule, false, errors, templateIDs).
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if err := store.Database.UpdateNotifyRule(ctx.Context(), id, rule); err != nil {
		return storeError(ctx, err)
	}
	reloadRulesEngine(ctx.Context())
	r, err := store.Database.GetNotifyRule(ctx.Context(), id)
	if err != nil {
		return notFound(ctx)
	}
	ctx.Type("html")
	return partials.NotifyRuleRow(r, templateIDs).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notifyRuleDelete(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	id, err := parseID(ctx)
	if err != nil {
		return err
	}
	if err := store.Database.DeleteNotifyRule(ctx.Context(), id); err != nil {
		return storeError(ctx, err)
	}
	reloadRulesEngine(ctx.Context())
	return ctx.SendString("")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseID(ctx fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return 0, err
	}
	return id, nil
}

func notFound(ctx fiber.Ctx) error {
	ctx.Type("html")
	return partials.EmptyState("Not found").Render(ctx.Context(), ctx.Response().BodyWriter())
}

func storeError(ctx fiber.Ctx, err error) error {
	ctx.Type("html")
	return partials.EmptyState(err.Error()).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func getProtocolsList() []string {
	protocols := []string{}
	for id := range notifypkg.List() {
		protocols = append(protocols, id)
	}
	return protocols
}

func getTemplateIDs() []string {
	if eng := notifytmpl.GetEngine(); eng != nil {
		return eng.ListTemplateIDs()
	}
	return []string{}
}

func parseRuleForm(ctx fiber.Ctx) model.NotifyRule {
	prio, _ := strconv.Atoi(ctx.FormValue("priority"))
	enabled := ctx.FormValue("enabled") == "on"
	return model.NotifyRule{
		RuleID:         ctx.FormValue("rule_id"),
		Name:           ctx.FormValue("name"),
		Action:         ctx.FormValue("action"),
		EventPattern:   ctx.FormValue("event_pattern"),
		ChannelPattern: ctx.FormValue("channel_pattern"),
		Condition:      ctx.FormValue("condition"),
		Priority:       prio,
		ParamsJSON:     ctx.FormValue("params_json"),
		Enabled:        enabled,
	}
}

func validateChannelForm(name, protocol, uri string) map[string]string {
	errs := map[string]string{}
	if name == "" {
		errs["name"] = "Name is required"
	}
	if protocol == "" {
		errs["protocol"] = "Protocol is required"
	}
	if uri == "" {
		errs["uri"] = "URI is required"
	}
	return errs
}

func validateRuleForm(rule model.NotifyRule) map[string]string {
	errs := map[string]string{}
	if rule.Name == "" {
		errs["name"] = "Name is required"
	}
	if rule.RuleID == "" {
		errs["rule_id"] = "Rule ID is required"
	}
	if rule.EventPattern == "" {
		errs["event_pattern"] = "Event pattern is required"
	}
	if rule.ChannelPattern == "" {
		errs["channel_pattern"] = "Channel pattern is required"
	}
	if rule.Action == "" {
		errs["action"] = "Action is required"
	}
	if rule.Condition != "" {
		if err := notifyrules.ValidateCondition(rule.Condition); err != nil {
			errs["condition"] = err.Error()
		}
	}
	if rule.ParamsJSON != "" {
		var params map[string]any
		if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
			errs["params_json"] = "Invalid JSON: " + err.Error()
		} else {
			switch rule.Action {
			case "throttle", "aggregate":
				if w, ok := params["window"].(string); !ok || w == "" {
					errs["params_json"] = "Window is required"
				}
				if rule.Action == "throttle" {
					if l, ok := params["limit"]; !ok || (func() bool {
						switch v := l.(type) {
						case float64:
							return v <= 0
						default:
							return true
						}
					}()) {
						errs["params_json"] = "Limit must be > 0"
					}
				}
				if rule.Action == "aggregate" {
					if tid, ok := params["digest_tpl_id"].(string); ok && tid != "" {
						if eng := notifytmpl.GetEngine(); eng != nil && !eng.HasTemplate(tid) {
							errs["params_json"] = "Unknown template: " + tid
						}
					}
				}
			}
		}
	}
	return errs
}

func reloadRulesEngine(ctx context.Context) {
	enabled := true
	rules, err := store.Database.ListNotifyRules(ctx, store.ListNotifyRuleOptions{Enabled: &enabled})
	if err != nil {
		return
	}
	configRules := make([]config.NotifyRule, 0, len(rules))
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		var cond string
		if r.Condition != "" {
			cond = r.Condition
		}
		var params config.NotifyRuleParams
		if r.ParamsJSON != "" {
			_ = sonic.Unmarshal([]byte(r.ParamsJSON), &params)
		}
		configRules = append(configRules, config.NotifyRule{
			ID:     r.RuleID,
			Action: config.NotifyRuleAction(r.Action),
			Match: config.NotifyRuleMatch{
				Event:   r.EventPattern,
				Channel: r.ChannelPattern,
			},
			Condition: cond,
			Priority:  r.Priority,
			Params:    params,
		})
	}
	_ = notifyrules.GetEngine().LoadConfig(configRules)
}
```

- [ ] **Step 2: Fix the test connectivity issue**

The masked URI problem: `GetNotifyChannel` returns masked URIs. We need a way to get the raw URI for testing. Add a method `GetNotifyChannelRaw` to the store interface and adapter, or use `GetNotifyChannel` from the ent client directly.

Better approach: Update the store to NOT mask URIs in `GetNotifyChannel` (masking happens in `ListNotifyChannels`). Or add an additional store method.

Simplest: Add `GetNotifyChannelRaw` in Task 5/6. But since we're past those tasks, let's modify the test handler to get the raw URI directly from the ent client.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/modules/web/...`

Expected: Some errors because templates don't exist yet. That's OK for now.

- [ ] **Step 4: Commit**

```bash
git add internal/modules/web/notify_settings_webservice.go
git commit -m "feat: add notify settings webservice handlers"
```

---

### Task 12: Create notify settings helpers for templates

**Files:**
- Create: `pkg/views/partials/notify_settings_helpers.go`

- [ ] **Step 1: Write the helpers file**

```go
package partials

import (
	"fmt"
	"net/url"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func notifyChannelRowID(item model.NotifyChannel) string {
	return fmt.Sprintf("notify-channel-%d", item.ID)
}

func notifyChannelEditURL(item model.NotifyChannel) string {
	return fmt.Sprintf("/service/web/notify-settings/channels/%d/edit", item.ID)
}

func notifyChannelDeleteURL(item model.NotifyChannel) string {
	return fmt.Sprintf("/service/web/notify-settings/channels/%d", item.ID)
}

func notifyChannelTestURL(item model.NotifyChannel) string {
	return fmt.Sprintf("/service/web/notify-settings/channels/%d/test", item.ID)
}

func notifyRuleRowID(item model.NotifyRule) string {
	return fmt.Sprintf("notify-rule-%d", item.ID)
}

func notifyRuleEditURL(item model.NotifyRule) string {
	return fmt.Sprintf("/service/web/notify-settings/rules/%d/edit", item.ID)
}

func notifyRuleDeleteURL(item model.NotifyRule) string {
	return fmt.Sprintf("/service/web/notify-settings/rules/%d", item.ID)
}

func notifyChannelFormID(item model.NotifyChannel, isNew bool) string {
	if isNew {
		return "notify-channel-form-new"
	}
	return "notify-channel-form-" + notifyChannelRowID(item)
}

func notifyRuleFormID(item model.NotifyRule, isNew bool) string {
	if isNew {
		return "notify-rule-form-new"
	}
	return "notify-rule-form-" + notifyRuleRowID(item)
}

func notifyChannelCancelURL() string {
	return "/service/web/notify-settings/channels/list"
}

func notifyRuleCancelURL() string {
	return "/service/web/notify-settings/rules/list"
}

func actionBadgeClass(action string) string {
	switch action {
	case "throttle":
		return "badge badge-warning"
	case "aggregate":
		return "badge badge-info"
	case "mute":
		return "badge badge-ghost"
	case "drop":
		return "badge badge-error"
	default:
		return "badge"
	}
}

func enabledBadgeClass(enabled bool) string {
	if enabled {
		return "badge badge-success"
	}
	return "badge badge-ghost"
}

func enabledText(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func escapePath(s string) string {
	return url.PathEscape(s)
}

func hasTemplateForRule(item model.NotifyRule, templateIDs []string) bool {
	if item.ParamsJSON == "" {
		return true
	}
	var params map[string]any
	if sonic.Unmarshal([]byte(item.ParamsJSON), &params) != nil {
		return true // can't parse - don't flag as stale
	}
	tid, ok := params["digest_tpl_id"].(string)
	if !ok || tid == "" {
		return true
	}
	for _, id := range templateIDs {
		if id == tid {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/views/partials/...`

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/views/partials/notify_settings_helpers.go
git commit -m "feat: add notify settings template helper functions"
```

---

### Task 13: Create channel table and row partials

**Files:**
- Create: `pkg/views/partials/notify_channels_table.templ`
- Create: `pkg/views/partials/notify_channel_row.templ`

- [ ] **Step 1: Write notify_channels_table.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ NotifyChannelsTable(channels []model.NotifyChannel) {
	<div class="card bg-base-100 shadow-sm">
		<div id="notify-channels-table"
			data-testid="notify-channels-table"
			class="overflow-x-auto">
			<table class="table">
				<thead>
				<tr>
					<th class="text-xs uppercase">Name</th>
					<th class="text-xs uppercase">Protocol</th>
					<th class="text-xs uppercase">URI</th>
					<th class="text-xs uppercase">Status</th>
					<th class="text-xs uppercase">Actions</th>
				</tr>
				</thead>
				<tbody id="notify-channels-rows">
				for _, ch := range channels {
					@NotifyChannelRow(ch)
				}
				if len(channels) == 0 {
					<tr id="notify-channels-empty">
						<td colspan="5" class="text-center text-base-content/50">No notification channels configured.</td>
					</tr>
				}
				</tbody>
			</table>
		</div>
	</div>
}
```

- [ ] **Step 2: Write notify_channel_row.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ NotifyChannelRow(item model.NotifyChannel) {
	<tr id={ notifyChannelRowID(item) } hx-target="this" hx-swap="outerHTML" class="hover">
		<td class="text-base-content font-medium">{ item.Name }</td>
		<td class="text-base-content/70">
			<span class="badge badge-outline">{ item.Protocol }</span>
		</td>
		<td class="text-base-content/50 font-mono text-xs max-w-xs truncate">{ item.URI }</td>
		<td>
			<span class={ enabledBadgeClass(item.Enabled) }>{ enabledText(item.Enabled) }</span>
		</td>
		<td>
			<div class="flex gap-1">
				<button hx-get={ notifyChannelEditURL(item) }
					data-testid="channel-edit"
					class="btn btn-ghost btn-xs text-primary">
					Edit
				</button>
				<button hx-post={ notifyChannelTestURL(item) }
					data-testid="channel-test"
					class="btn btn-ghost btn-xs text-info"
					hx-indicator="#channel-test-spinner-{ fmt.Sprintf("%d", item.ID) }">
					Test
					<span id={ "channel-test-spinner-" + fmt.Sprintf("%d", item.ID) } class="htmx-indicator loading loading-spinner loading-xs"></span>
				</button>
				<button hx-delete={ notifyChannelDeleteURL(item) }
					hx-confirm="Delete this channel?"
					data-testid="channel-delete"
					class="btn btn-ghost btn-xs text-error">
					Delete
				</button>
			</div>
		</td>
	</tr>
}
```

- [ ] **Step 3: Run templ generate**

Run: `templ generate pkg/views/partials/notify_channels_table.templ pkg/views/partials/notify_channel_row.templ`

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/views/...`

Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add pkg/views/partials/notify_channels_table.templ pkg/views/partials/notify_channels_table_templ.go pkg/views/partials/notify_channel_row.templ pkg/views/partials/notify_channel_row_templ.go
git commit -m "feat: add notify channel table and row templ partials"
```

---

### Task 14: Create channel form partial

**Files:**
- Create: `pkg/views/partials/notify_channel_form.templ`

- [ ] **Step 1: Write notify_channel_form.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ NotifyChannelForm(item model.NotifyChannel, isNew bool, errors map[string]string, protocols []string) {
	<tr id={ notifyChannelFormID(item, isNew) }>
		<td>
			<input type="text" name="name" value={ item.Name }
				data-testid="channel-name"
				class={ "input input-bordered input-sm w-full " + fieldError(errors, "name") }
				placeholder="Channel name"
			/>
			<div class="text-error text-xs">{ errors["name"] }</div>
		</td>
		<td>
			<select name="protocol"
				data-testid="channel-protocol"
				class={ "select select-bordered select-sm w-full " + fieldError(errors, "protocol") }>
				<option value="">Select...</option>
				for _, p := range protocols {
					<option value={ p } if item.Protocol == p { selected }>{ p }</option>
				}
			</select>
			<div class="text-error text-xs">{ errors["protocol"] }</div>
		</td>
		<td>
			<input type="password" name="uri"
				data-testid="channel-uri"
				class={ "input input-bordered input-sm w-full font-mono " + fieldError(errors, "uri") }
				placeholder="slack://hooks.slack.com/services/..."
			/>
			<div class="text-error text-xs">{ errors["uri"] }</div>
		</td>
		<td>
			<label class="flex items-center gap-1 cursor-pointer">
				<input type="checkbox" name="enabled"
					data-testid="channel-enabled"
					if item.Enabled { checked }
					class="checkbox checkbox-sm"/>
				<span class="text-xs">Enabled</span>
			</label>
		</td>
		<td>
			<div class="flex gap-1">
				<button type="button"
					if isNew {
						hx-post="/service/web/notify-settings/channels"
					} else {
						hx-put={ notifyChannelDeleteURL(item) }
					}
					hx-target="closest tr"
					hx-swap="outerHTML"
					hx-include="[name='name'], [name='protocol'], [name='uri'], [name='enabled']"
					data-testid="channel-save"
					class="btn btn-primary btn-sm">
					Save
				</button>
				<button type="button"
					hx-get={ notifyChannelCancelURL() }
					hx-target="#notify-channels-table"
					hx-swap="outerHTML"
					data-testid="channel-cancel"
					class="btn btn-ghost btn-sm">
					Cancel
				</button>
			</div>
			if errors["_save"] != "" {
				<div class="text-error text-xs mt-1">{ errors["_save"] }</div>
			}
		</td>
	</tr>
}
```

- [ ] **Step 2: Run templ generate**

Run: `templ generate pkg/views/partials/notify_channel_form.templ`

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/...`

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/views/partials/notify_channel_form.templ pkg/views/partials/notify_channel_form_templ.go
git commit -m "feat: add notify channel form templ partial"
```

---

### Task 15: Create rule table and row partials

**Files:**
- Create: `pkg/views/partials/notify_rules_table.templ`
- Create: `pkg/views/partials/notify_rule_row.templ`

- [ ] **Step 1: Write notify_rules_table.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ NotifyRulesTable(rules []model.NotifyRule, templateIDs []string) {
	<div class="card bg-base-100 shadow-sm">
		<div id="notify-rules-table"
			data-testid="notify-rules-table"
			class="overflow-x-auto">
			<table class="table">
				<thead>
				<tr>
					<th class="text-xs uppercase">Priority</th>
					<th class="text-xs uppercase">Name</th>
					<th class="text-xs uppercase">Action</th>
					<th class="text-xs uppercase">Event</th>
					<th class="text-xs uppercase">Channel</th>
					<th class="text-xs uppercase">Status</th>
					<th class="text-xs uppercase">Actions</th>
				</tr>
				</thead>
				<tbody id="notify-rules-rows">
				for _, rule := range rules {
					@NotifyRuleRow(rule, templateIDs)
				}
				if len(rules) == 0 {
					<tr id="notify-rules-empty">
						<td colspan="7" class="text-center text-base-content/50">No notification rules configured.</td>
					</tr>
				}
				</tbody>
			</table>
		</div>
	</div>
}
```

- [ ] **Step 2: Write notify_rule_row.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ NotifyRuleRow(item model.NotifyRule, templateIDs []string) {
	<tr id={ notifyRuleRowID(item) } hx-target="this" hx-swap="outerHTML" class="hover">
		<td class="text-base-content/70 text-center">{ fmt.Sprintf("%d", item.Priority) }</td>
		<td class="text-base-content font-medium">
			{ item.Name }
			if item.Action == "aggregate" && item.ParamsJSON != "" && !hasTemplateForRule(item, templateIDs) {
				<span class="tooltip" data-tip="Template not found in current config" data-testid="rule-stale-template">&#9888;</span>
			}
		</td>
		<td><span class={ actionBadgeClass(item.Action) }>{ item.Action }</span></td>
		<td class="text-base-content/70 font-mono text-sm">{ truncateString(item.EventPattern, 30) }</td>
		<td class="text-base-content/70 font-mono text-sm">{ truncateString(item.ChannelPattern, 30) }</td>
		<td>
			<span class={ enabledBadgeClass(item.Enabled) }>{ enabledText(item.Enabled) }</span>
		</td>
		<td>
			<div class="flex gap-1">
				<button hx-get={ notifyRuleEditURL(item) }
					data-testid="rule-edit"
					class="btn btn-ghost btn-xs text-primary">
					Edit
				</button>
				<button hx-delete={ notifyRuleDeleteURL(item) }
					hx-confirm="Delete this rule?"
					data-testid="rule-delete"
					class="btn btn-ghost btn-xs text-error">
					Delete
				</button>
			</div>
		</td>
	</tr>
}
```

- [ ] **Step 3: Run templ generate**

Run: `templ generate pkg/views/partials/notify_rules_table.templ pkg/views/partials/notify_rule_row.templ`

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/views/...`

Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add pkg/views/partials/notify_rules_table.templ pkg/views/partials/notify_rules_table_templ.go pkg/views/partials/notify_rule_row.templ pkg/views/partials/notify_rule_row_templ.go
git commit -m "feat: add notify rule table and row templ partials"
```

---

### Task 16: Create rule form partial

**Files:**
- Create: `pkg/views/partials/notify_rule_form.templ`

- [ ] **Step 1: Write notify_rule_form.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ NotifyRuleForm(item model.NotifyRule, isNew bool, errors map[string]string, templateIDs []string) {
	<tr id={ notifyRuleFormID(item, isNew) } x-data="{ action: '{ item.Action }' }">
		<td>
			<input type="number" name="priority" value={ fmt.Sprintf("%d", item.Priority) }
				data-testid="rule-priority"
				class={ "input input-bordered input-sm w-16 " + fieldError(errors, "priority") }
				min="0"
			/>
		</td>
		<td>
			<input type="text" name="name" value={ item.Name }
				data-testid="rule-name"
				class={ "input input-bordered input-sm w-full " + fieldError(errors, "name") }
				placeholder="Rule name"
			/>
			<div class="text-error text-xs">{ errors["name"] }</div>
		</td>
		<td>
			<select name="action"
				data-testid="rule-action"
				x-model="action"
				class={ "select select-bordered select-sm w-full " + fieldError(errors, "action") }>
				<option value="">Select...</option>
				<option value="throttle" if item.Action == "throttle" { selected }>throttle</option>
				<option value="aggregate" if item.Action == "aggregate" { selected }>aggregate</option>
				<option value="mute" if item.Action == "mute" { selected }>mute</option>
				<option value="drop" if item.Action == "drop" { selected }>drop</option>
			</select>
			<div class="text-error text-xs">{ errors["action"] }</div>
		</td>
		<td>
			<input type="text" name="event_pattern" value={ item.EventPattern }
				data-testid="rule-event-pattern"
				class={ "input input-bordered input-sm w-full font-mono " + fieldError(errors, "event_pattern") }
				placeholder='*'
			/>
			<input type="text" name="rule_id" value={ item.RuleID }
				data-testid="rule-id"
				class={ "input input-bordered input-sm w-full mt-1 font-mono " + fieldError(errors, "rule_id") }
				placeholder="rule_id (e.g. night_mute)"
			/>
			<div class="text-error text-xs">{ errors["rule_id"] }</div>
			<div class="text-error text-xs">{ errors["event_pattern"] }</div>
		</td>
		<td>
			<input type="text" name="channel_pattern" value={ item.ChannelPattern }
				data-testid="rule-channel-pattern"
				class={ "input input-bordered input-sm w-full font-mono " + fieldError(errors, "channel_pattern") }
				placeholder='*'
			/>
			<div class="text-error text-xs">{ errors["channel_pattern"] }</div>
		</td>
		<td>
			<label class="flex items-center gap-1 cursor-pointer">
				<input type="checkbox" name="enabled"
					data-testid="rule-enabled"
					if item.Enabled { checked }
					class="checkbox checkbox-sm"/>
				<span class="text-xs">Enabled</span>
			</label>
		</td>
		<td>
			<div class="flex flex-col gap-1">
				<div x-show="action === 'mute'">
					<input type="text" name="condition" value={ item.Condition }
						data-testid="rule-condition"
						class={ "input input-bordered input-sm w-full font-mono " + fieldError(errors, "condition") }
						placeholder='time.hour >= 23 || time.hour < 8'
					/>
					<div class="text-error text-xs">{ errors["condition"] }</div>
				</div>
				<div x-show="action === 'throttle' || action === 'aggregate'">
					<textarea name="params_json" rows="3"
						data-testid="rule-params"
						class={ "textarea textarea-bordered textarea-sm w-full font-mono " + fieldError(errors, "params_json") }
						placeholder='{"window": "5m", "limit": 1}'
					>{ item.ParamsJSON }</textarea>
					if errors["params_json"] != "" {
						<div class="text-error text-xs">{ errors["params_json"] }</div>
					} else if item.Action == "aggregate" && len(templateIDs) > 0 {
						<div class="text-xs text-base-content/50 mt-1">Available templates: {
							for i, tid := range templateIDs {
								if i > 0 { ", " }
								{ tid }
							}
						}</div>
					}
				</div>
				<div class="flex gap-1 mt-1">
					<button type="button"
						if isNew {
							hx-post="/service/web/notify-settings/rules"
						} else {
							hx-put={ notifyRuleDeleteURL(item) }
						}
						hx-target="closest tr"
						hx-swap="outerHTML"
						hx-include="[name='name'], [name='rule_id'], [name='action'], [name='event_pattern'], [name='channel_pattern'], [name='condition'], [name='priority'], [name='params_json'], [name='enabled']"
						data-testid="rule-save"
						class="btn btn-primary btn-sm">
						Save
					</button>
					<button type="button"
						hx-get={ notifyRuleCancelURL() }
						hx-target="#notify-rules-table"
						hx-swap="outerHTML"
						data-testid="rule-cancel"
						class="btn btn-ghost btn-sm">
						Cancel
					</button>
				</div>
				if errors["_save"] != "" {
					<div class="text-error text-xs mt-1">{ errors["_save"] }</div>
				}
			</div>
		</td>
	</tr>
}
```

- [ ] **Step 2: Run templ generate**

Run: `templ generate pkg/views/partials/notify_rule_form.templ`

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/...`

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/views/partials/notify_rule_form.templ pkg/views/partials/notify_rule_form_templ.go
git commit -m "feat: add notify rule form templ partial"
```

---

### Task 17: Create notify settings page template

**Files:**
- Create: `pkg/views/pages/notify_settings.templ`

- [ ] **Step 1: Write notify_settings.templ**

```templ
package pages

import (
	"github.com/flowline-io/flowbot/pkg/views/layout"
)

templ NotifySettingsPage(templateIDs, protocols []string) {
	@layout.Base("Notification Settings — Flowbot") {
		<div class="flex items-center justify-between mb-6">
			<h1 class="text-2xl font-bold text-base-content">Notification Settings</h1>
		</div>

		<div role="tablist" class="tabs tabs-bordered mb-4">
			<input type="radio" name="notify-tabs" class="tab" aria-label="Channels" checked="checked"/>
			<div class="tab-content p-4">
				<div class="flex justify-end mb-3">
					<button hx-get="/service/web/notify-settings/channels/new"
						hx-target="#notify-channels-rows"
						hx-swap="afterbegin"
						data-testid="channels-new"
						class="btn btn-primary btn-sm">
						New Channel
					</button>
				</div>
				<div hx-get="/service/web/notify-settings/channels/list"
					hx-trigger="load"
					hx-swap="outerHTML">
					<div class="skeleton h-32 w-full"></div>
				</div>
			</div>

			<input type="radio" name="notify-tabs" class="tab" aria-label="Rules"/>
			<div class="tab-content p-4">
				<div class="flex justify-end mb-3">
					<button hx-get="/service/web/notify-settings/rules/new"
						hx-target="#notify-rules-rows"
						hx-swap="afterbegin"
						data-testid="rules-new"
						class="btn btn-primary btn-sm">
						New Rule
					</button>
				</div>
				<div hx-get="/service/web/notify-settings/rules/list"
					hx-trigger="load once"
					hx-swap="outerHTML">
					<div class="skeleton h-32 w-full"></div>
				</div>
			</div>
		</div>
	}
}
```

- [ ] **Step 2: Run templ generate**

Run: `templ generate pkg/views/pages/notify_settings.templ`

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/...`

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/views/pages/notify_settings.templ pkg/views/pages/notify_settings_templ.go
git commit -m "feat: add notify settings page template"
```

---

### Task 18: Add navbar link

**Files:**
- Modify: `pkg/views/layout/base.templ`

- [ ] **Step 1: Add navbar link between Notifications and Relations**

Find the line with `href="/service/web/notifications"` and add the new link after it:

```templ
<a href="/service/web/notifications" data-testid="nav-notifications" class="btn btn-ghost btn-sm">Notifications</a>
<a href="/service/web/notify-settings" data-testid="nav-notify-settings" class="btn btn-ghost btn-sm">Notify Settings</a>
```

- [ ] **Step 2: Run templ generate**

Run: `templ generate pkg/views/layout/base.templ`

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/...`

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/views/layout/base.templ pkg/views/layout/base_templ.go
git commit -m "feat: add Notify Settings link to navbar"
```

---

### Task 19: Register routes in module.go

**Files:**
- Modify: `internal/modules/web/module.go`

- [ ] **Step 1: Add route registration**

Add after `module.Webservice(app, Name, notificationWebserviceRules)`:

```go
module.Webservice(app, Name, notifySettingsWebserviceRules)
```

- [ ] **Step 2: Verify full build**

Run: `go build ./...`

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/module.go
git commit -m "feat: register notify settings webservice routes"
```

---

### Task 20: Run format and lint

**Files:**
- (all modified files)

- [ ] **Step 1: Run formatter**

Run: `go tool task format`

- [ ] **Step 2: Run linter**

Run: `go tool task lint`

- [ ] **Step 3: Fix any issues**

Fix lint errors (unused imports, naming, etc.) and re-run.

- [ ] **Step 4: Commit fixes**

```bash
git add -A
git commit -m "chore: format and lint fixes for notify settings"
```

---

## Final Verification

Run the full test suite to verify nothing is broken:

```bash
go tool task build
go tool task lint
go tool task test
```

All should pass without errors.
