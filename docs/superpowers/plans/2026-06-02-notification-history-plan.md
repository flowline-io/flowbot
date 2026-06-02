# Notification History Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add notification delivery history with per-user persistence, a web UI page with retry support, and non-blocking async recording in the notify gateway.

**Architecture:** New ent schema `NotificationRecord` maps 1:1 to new `notification_records` table. `NotifyStore` (in `store.go`) provides CRUD with ID-based cursor pagination. `GatewaySend()` spawns goroutines that call `NotifyStore.Record()` + `DeleteOldest()` with a 2s timeout. Web page at `/service/web/notifications` renders per-user history with HTMX  pagination and retry.

**Tech Stack:** Ent ORM (auto-migration), templ + HTMX + DaisyUI, PostgreSQL via ent client, Go text/template with Sprig

---

### Task 1: Create Ent Schema

**Files:**
- Create: `internal/store/ent/schema/notification_record.go`

- [ ] **Step 1: Write the ent schema file**

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

type NotificationRecord struct {
	ent.Schema
}

func (NotificationRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Immutable(),
		field.String("uid").NotEmpty(),
		field.String("channel").NotEmpty(),
		field.String("template_id").NotEmpty(),
		field.String("summary").Default(""),
		field.Enum("status").Values("success", "failed", "dropped", "throttled", "aggregated", "muted").Default("success"),
		field.String("error_msg").Default(""),
		field.JSON("payload_snapshot", map[string]any{}).Optional(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (NotificationRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "id"),
		index.Fields("uid", "created_at"),
	}
}

func (NotificationRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("notification_records"),
	}
}
```

- [ ] **Step 2: Generate ent code**

Run: `go tool task ent`
Expected: New files generated under `internal/store/ent/gen/notificationrecord/` and `internal/store/ent/gen/notification_record.go`

- [ ] **Step 3: Verify generated code compiles**

Run: `go build ./internal/store/...`
Expected: BUILD OK

- [ ] **Step 4: Commit**

```bash
git add internal/store/ent/schema/notification_record.go internal/store/ent/gen/
git commit -m "feat: add notification_record ent schema"
```

---

### Task 2: Add NotifyStore to store.go

**Files:**
- Modify: `internal/store/store.go` — add NotifyStore struct, NewNotifyStore, methods, and imports
- Test: `internal/store/store_test.go` — add NotifyStore tests

- [ ] **Step 1: Add imports to store.go**

Add these imports alongside existing gen package imports in `internal/store/store.go`:

```go
import (
    // ... existing imports ...
    "strconv"

    "github.com/flowline-io/flowbot/internal/store/ent/gen/notificationrecord"
)
```

Place `strconv` in the stdlib block, `notificationrecord` in the ent gen block.

- [ ] **Step 2: Write the NotifyStore code**

Add this at the end of `internal/store/store.go` (before the closing line):

```go
// NotifyStore provides CRUD for notification delivery records.
type NotifyStore struct {
	client *gen.Client
}

// NewNotifyStore returns a NotifyStore backed by the given Ent client.
func NewNotifyStore(client *gen.Client) *NotifyStore {
	return &NotifyStore{client: client}
}

// ListNotifyRecordsOptions holds filters and pagination for listing notification records.
type ListNotifyRecordsOptions struct {
	Limit  int    // max 100, default 20
	Cursor string // opaque cursor: ID value as string
}

// Record inserts a notification delivery record and returns the new row ID.
func (s *NotifyStore) Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg string, payload map[string]any) (int, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	create := s.client.NotificationRecord.Create().
		SetUID(uid).
		SetChannel(channel).
		SetTemplateID(templateID).
		SetSummary(summary).
		SetStatus(notificationrecord.Status(status)).
		SetErrorMsg(errorMsg).
		SetCreatedAt(time.Now())
	if payload != nil {
		create = create.SetPayloadSnapshot(payload)
	}
	rec, err := create.Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("record notification: %w", err)
	}
	return rec.ID, nil
}

// ListRecords returns per-user notification records, cursor-paginated (newest first).
func (s *NotifyStore) ListRecords(ctx context.Context, uid string, opts ListNotifyRecordsOptions) ([]*gen.NotificationRecord, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	q := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Order(gen.Desc(notificationrecord.FieldID)).
		Limit(opts.Limit + 1)

	if opts.Cursor != "" {
		id, err := strconv.Atoi(opts.Cursor)
		if err == nil {
			q = q.Where(notificationrecord.IDLT(id))
		}
	}

	records, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("list notification records: %w", err)
	}

	var nextCursor string
	if len(records) > opts.Limit {
		nextCursor = strconv.Itoa(records[opts.Limit-1].ID)
		records = records[:opts.Limit]
	}

	return records, nextCursor, nil
}

// GetRecord returns a single notification record by ID.
func (s *NotifyStore) GetRecord(ctx context.Context, id int) (*gen.NotificationRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rec, err := s.client.NotificationRecord.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get notification record: %w", err)
	}
	return rec, nil
}

// DeleteOldest removes the oldest records for a user exceeding keepN.
func (s *NotifyStore) DeleteOldest(ctx context.Context, uid string, keepN int) error {
	if s == nil || s.client == nil {
		return nil
	}
	if keepN <= 0 {
		return nil
	}

	total, err := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("count records for cleanup: %w", err)
	}
	if total <= keepN {
		return nil
	}

	excess := total - keepN
	oldest, err := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Order(gen.Asc(notificationrecord.FieldCreatedAt)).
		Limit(excess).
		All(ctx)
	if err != nil {
		return fmt.Errorf("find oldest records: %w", err)
	}

	ids := make([]int, len(oldest))
	for i, rec := range oldest {
		ids[i] = rec.ID
	}
	_, err = s.client.NotificationRecord.Delete().
		Where(notificationrecord.IDIn(ids...)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete oldest records: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Write store tests**

Add to the existing `internal/store/store_test.go` file at the end:

```go
func TestNotifyStore_Record(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	ns := NewNotifyStore(client)
	ctx := context.Background()

	tests := []struct {
		name    string
		uid     string
		channel string
		tpl     string
		summary string
		status  string
		errMsg  string
		payload map[string]any
	}{
		{
			name:    "success record with summary",
			uid:     "user1",
			channel: "slack",
			tpl:     "bookmark.created",
			summary: "New bookmark: example.com",
			status:  "success",
			errMsg:  "",
			payload: map[string]any{"url": "https://example.com", "summary": "New bookmark: example.com"},
		},
		{
			name:    "failed record with error",
			uid:     "user1",
			channel: "pushover",
			tpl:     "task.alert",
			summary: "Task #42 created",
			status:  "failed",
			errMsg:  "connection timeout",
			payload: map[string]any{"summary": "Task #42 created"},
		},
		{
			name:    "dropped record no error",
			uid:     "user2",
			channel: "ntfy",
			tpl:     "system.health",
			summary: "",
			status:  "dropped",
			errMsg:  "",
			payload: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := ns.Record(ctx, tt.uid, tt.channel, tt.tpl, tt.summary, tt.status, tt.errMsg, tt.payload)
			require.NoError(t, err)
			assert.Greater(t, id, 0)

			rec, err := ns.GetRecord(ctx, id)
			require.NoError(t, err)
			require.NotNil(t, rec)
			assert.Equal(t, tt.uid, rec.UID)
			assert.Equal(t, tt.channel, rec.Channel)
			assert.Equal(t, tt.tpl, rec.TemplateID)
			assert.Equal(t, tt.summary, rec.Summary)
			assert.Equal(t, tt.status, string(rec.Status))
			assert.Equal(t, tt.errMsg, rec.ErrorMsg)
		})
	}
}

func TestNotifyStore_ListRecords_Pagination(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	ns := NewNotifyStore(client)
	ctx := context.Background()

	for i := range 25 {
		_, err := ns.Record(ctx, "user_p", "slack", "test.template", "", "success", "", nil)
		require.NoError(t, err)
		// brief sleep ensures distinct created_at ordering
		time.Sleep(time.Millisecond)
		_ = i
	}

	tests := []struct {
		name       string
		limit      int
		cursor     string
		wantCount  int
		wantCursor bool
	}{
		{
			name:       "first page of 10",
			limit:      10,
			cursor:     "",
			wantCount:  10,
			wantCursor: true,
		},
		{
			name:       "page of 30 exceeds total",
			limit:      30,
			cursor:     "",
			wantCount:  25,
			wantCursor: false,
		},
		{
			name:       "page of 0 defaults to 20",
			limit:      0,
			cursor:     "",
			wantCount:  20,
			wantCursor: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			records, nextCursor, err := ns.ListRecords(ctx, "user_p", ListNotifyRecordsOptions{
				Limit:  tt.limit,
				Cursor: tt.cursor,
			})
			require.NoError(t, err)
			assert.Len(t, records, tt.wantCount)
			if tt.wantCursor {
				assert.NotEmpty(t, nextCursor)
			} else {
				assert.Empty(t, nextCursor)
			}
		})
	}
}

func TestNotifyStore_DeleteOldest(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	ns := NewNotifyStore(client)
	ctx := context.Background()

	for range 10 {
		_, err := ns.Record(ctx, "user_d", "slack", "test.template", "", "success", "", nil)
		require.NoError(t, err)
	}

	tests := []struct {
		name         string
		keepN        int
		wantCount    int
		wantAtMost   int
	}{
		{
			name:       "keep 5 deletes excess",
			keepN:      5,
			wantAtMost: 5,
		},
		{
			name:       "keep 20 no deletion",
			keepN:      20,
			wantAtMost: 10,
		},
		{
			name:       "keep 0 no-op",
			keepN:      0,
			wantAtMost: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ns.DeleteOldest(ctx, "user_d", tt.keepN)
			require.NoError(t, err)
		})
	}
}

func TestNotifyStore_Cursor_Pagination_Continuity(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	ns := NewNotifyStore(client)
	ctx := context.Background()

	for range 25 {
		_, err := ns.Record(ctx, "user_c", "slack", "test.template", "", "success", "", nil)
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
	}

	page1, cursor1, err := ns.ListRecords(ctx, "user_c", ListNotifyRecordsOptions{Limit: 10})
	require.NoError(t, err)
	require.Len(t, page1, 10)
	require.NotEmpty(t, cursor1)

	page2, cursor2, err := ns.ListRecords(ctx, "user_c", ListNotifyRecordsOptions{Limit: 10, Cursor: cursor1})
	require.NoError(t, err)
	require.Len(t, page2, 10)
	require.NotEmpty(t, cursor2)

	page3, cursor3, err := ns.ListRecords(ctx, "user_c", ListNotifyRecordsOptions{Limit: 10, Cursor: cursor2})
	require.NoError(t, err)
	require.Len(t, page3, 5)
	require.Empty(t, cursor3)

	idSet := make(map[int]bool)
	for _, rec := range page1 {
		idSet[rec.ID] = true
	}
	for _, rec := range page2 {
		assert.False(t, idSet[rec.ID], "page2 should not contain IDs from page1")
		idSet[rec.ID] = true
	}
	for _, rec := range page3 {
		assert.False(t, idSet[rec.ID], "page3 should not contain IDs from page1/page2")
		idSet[rec.ID] = true
	}
	assert.Len(t, idSet, 25)
}
```

- [ ] **Step 4: Run store tests**

Run: `go tool task test -- -run "TestNotifyStore" -v ./internal/store/`
Expected: ALL PASS (7 subtests across 4 test functions)

- [ ] **Step 5: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat: add NotifyStore with cursor pagination and rolling window cleanup"
```

---

### Task 3: Integrate Notification Recording into GatewaySend

**Files:**
- Modify: `pkg/notify/notify.go` — add recordAsync helper, call from GatewaySend and evaluateAndRenderNotification
- Test: `pkg/notify/notify_test.go` — add tests for recordAsync behavior

- [ ] **Step 1: Add recordAsync helper to notify.go**

Add this function in `pkg/notify/notify.go` after the existing imports and before `Register`:

First, add these imports to the existing import block:
```go
import (
    // ... existing imports ...
    "time"
    "github.com/flowline-io/flowbot/internal/store"
    // ...
)
```

Note: `internal/store` is already imported in notify.go. `time` is also already imported. No import changes needed.

Add the `getNotifyStore` and `recordAsync` helpers after `sendToUserChannel` (before the file ends):

```go
// getNotifyStore returns the NotifyStore from the global database adapter,
// or nil if the store is not available.
func getNotifyStore() *store.NotifyStore {
	if store.Database == nil || store.Database.GetDB() == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewNotifyStore(client)
}

// recordAsync writes a notification delivery record in a goroutine with a 2s timeout.
// It also triggers deferred rolling window cleanup (best-effort).
func recordAsync(uid types.Uid, channel, templateID, summary, status, errMsg string, payload map[string]any) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		ns := getNotifyStore()
		if ns == nil {
			return
		}
		if _, err := ns.Record(ctx, uid.String(), channel, templateID, summary, status, errMsg, payload); err != nil {
			flog.Warn("[notify] failed to record notification: %v", err)
			return
		}
		// Rolling window cleanup (best-effort, keep last 200 per user)
		if err := ns.DeleteOldest(ctx, uid.String(), 200); err != nil {
			flog.Warn("[notify] failed to cleanup old notifications: %v", err)
		}
	}()
}
```

- [ ] **Step 2: Add recording calls in GatewaySend**

Modify the `GatewaySend` function to add a `summary` extraction and recording calls:

```go
func GatewaySend(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error {
	engine := notifytmpl.GetEngine()
	if engine == nil {
		flog.Warn("[notify] template engine not initialized, skipping notification %s", templateID)
		return nil
	}

	// check if template exists
	if engine.GetTemplateID(templateID) == "" {
		return types.Errorf(types.ErrNotFound, "template %s not found", templateID)
	}

	summary, _ := payload["summary"].(string)

	var errs []error
	for _, channel := range channels {
		result, err := evaluateAndRenderNotification(ctx, templateID, channel, payload)
		if err != nil {
			errs = append(errs, err)
			recordAsync(uid, channel, templateID, summary, "failed", err.Error(), payload)
			continue
		}
		if result == nil {
			continue
		}

		msg := buildNotifyMessage(result, payload)

		if err := sendToUserChannel(ctx, uid, templateID, channel, msg); err != nil {
			errs = append(errs, err)
			recordAsync(uid, channel, templateID, summary, "failed", err.Error(), payload)
		} else {
			recordAsync(uid, channel, templateID, summary, "success", "", payload)
		}
	}

	if len(errs) > 0 {
		return types.Errorf(types.ErrInternal, "notification errors: %v", errs)
	}
	return nil
}
```

- [ ] **Step 3: Add recording calls for rule outcomes in evaluateAndRenderNotification**

Modify the `evaluateAndRenderNotification` function to record suppressed statuses. Since this function doesn't have the `uid` parameter, move the recording back to `GatewaySend` — instead, change `evaluateAndRenderNotification` to return the action alongside the result.

Change the function signature to return a new type:

Add this before `evaluateAndRenderNotification`:

```go
type evalResult struct {
	renderResult *notifytmpl.RenderResult
	action       string // "drop", "mute", "throttle", "aggregate", or ""
}
```

Modify `evaluateAndRenderNotification`:

```go
func evaluateAndRenderNotification(ctx context.Context, templateID, channel string, payload map[string]any) (*evalResult, error) {
	ruleEngine := notifyrules.GetEngine()
	if ruleEngine != nil {
		ruleResult := ruleEngine.Evaluate(ctx, templateID, channel)
		if ruleResult != nil {
			switch ruleResult.Action {
			case config.NotifyRuleActionDrop:
				flog.Info("[notify] message dropped by rule %s for %s/%s", ruleResult.RuleID, templateID, channel)
				return &evalResult{action: "dropped"}, nil
			case config.NotifyRuleActionMute:
				flog.Info("[notify] message muted by rule %s for %s/%s", ruleResult.RuleID, templateID, channel)
				return &evalResult{action: "muted"}, nil
			case config.NotifyRuleActionThrottle:
				if handleThrottleRule(ctx, ruleResult, templateID, channel) {
					return &evalResult{action: "throttled"}, nil
				}
			case config.NotifyRuleActionAggregate:
				if handleAggregateRule(ctx, ruleResult, templateID, channel, payload) {
					return &evalResult{action: "aggregated"}, nil
				}
			}
		}
	}

	engine := notifytmpl.GetEngine()
	result, err := engine.Render(templateID, channel, payload)
	if err != nil {
		flog.Warn("[notify] failed to render template %s for channel %s: %v", templateID, channel, err)
		return nil, err
	}
	return &evalResult{renderResult: result}, nil
}
```

Now update `GatewaySend` to use the new return type and record rule outcomes:

```go
func GatewaySend(ctx context.Context, uid types.Uid, templateID string, channels []string, payload map[string]any) error {
	engine := notifytmpl.GetEngine()
	if engine == nil {
		flog.Warn("[notify] template engine not initialized, skipping notification %s", templateID)
		return nil
	}

	if engine.GetTemplateID(templateID) == "" {
		return types.Errorf(types.ErrNotFound, "template %s not found", templateID)
	}

	summary, _ := payload["summary"].(string)

	var errs []error
	for _, channel := range channels {
		eval, err := evaluateAndRenderNotification(ctx, templateID, channel, payload)
		if err != nil {
			errs = append(errs, err)
			recordAsync(uid, channel, templateID, summary, "failed", err.Error(), payload)
			continue
		}
		if eval == nil {
			continue
		}
		if eval.action != "" {
			recordAsync(uid, channel, templateID, summary, eval.action, "", payload)
			continue
		}
		if eval.renderResult == nil {
			continue
		}

		msg := buildNotifyMessage(eval.renderResult, payload)

		if err := sendToUserChannel(ctx, uid, templateID, channel, msg); err != nil {
			errs = append(errs, err)
			recordAsync(uid, channel, templateID, summary, "failed", err.Error(), payload)
		} else {
			recordAsync(uid, channel, templateID, summary, "success", "", payload)
		}
	}

	if len(errs) > 0 {
		return types.Errorf(types.ErrInternal, "notification errors: %v", errs)
	}
	return nil
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/notify/...`
Expected: BUILD OK

- [ ] **Step 5: Commit**

```bash
git add pkg/notify/notify.go
git commit -m "feat: record notification delivery outcomes in GatewaySend"
```

---

### Task 4: Create Web Templates

**Files:**
- Create: `pkg/views/pages/notifications.templ`
- Create: `pkg/views/partials/notifications_table.templ`

- [ ] **Step 1: Write the page template**

`pkg/views/pages/notifications.templ`:

```templ
package pages

import (
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

type NotificationsPageParams struct {
	Records    []*gen.NotificationRecord
	NextCursor string
}

templ NotificationsPage(p NotificationsPageParams) {
	@layout.Base("Notifications") {
		<div class="container mx-auto p-4">
			<h1 class="text-2xl font-bold mb-4">Notifications</h1>
			<div id="notifications-content">
				@partials.NotificationsTable(p.Records, p.NextCursor)
			</div>
		</div>
	}
}
```

- [ ] **Step 2: Write the table partial**

`pkg/views/partials/notifications_table.templ`:

```templ
package partials

import (
	"strconv"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

func statusBadgeClass(status string) string {
	switch status {
	case "success":
		return "badge badge-success badge-sm"
	case "failed":
		return "badge badge-error badge-sm"
	default:
		return "badge badge-ghost badge-sm"
	}
}

templ NotificationsTable(records []*gen.NotificationRecord, nextCursor string) {
	<div class="card bg-base-100 shadow-sm" id="notifications-table" data-testid="notifications-table">
		<div class="overflow-x-auto">
			<table class="table">
				<thead>
				<tr>
					<th class="text-xs uppercase">Time</th>
					<th class="text-xs uppercase">Channel</th>
					<th class="text-xs uppercase">Template</th>
					<th class="text-xs uppercase">Summary</th>
					<th class="text-xs uppercase">Status</th>
				</tr>
				</thead>
				<tbody id="notifications-rows">
				for _, r := range records {
					<tr id={ "notify-row-" + strconv.Itoa(r.ID) }>
						<td class="text-xs whitespace-nowrap">{ r.CreatedAt.Format("15:04:05") }</td>
						<td><span class="badge badge-outline badge-xs">{ r.Channel }</span></td>
						<td class="text-xs font-mono">{ r.TemplateID }</td>
						<td class="text-xs max-w-48 truncate" title={ r.Summary }>{ r.Summary }</td>
						<td class="flex items-center gap-1">
							<span class={ statusBadgeClass(string(r.Status)) } title={ r.ErrorMsg }>{ string(r.Status) }</span>
							if string(r.Status) == "failed" {
								<button class="btn btn-xs btn-outline btn-error"
									hx-post={ templ.URL("/service/web/notifications/" + strconv.Itoa(r.ID) + "/retry") }
									hx-target={ "#notify-row-" + strconv.Itoa(r.ID) }
									hx-swap="outerHTML"
									hx-disabled-elt="this"
									title="Retries go through normal delivery. Rate limits still apply."
									data-testid={ "retry-btn-" + strconv.Itoa(r.ID) }>Retry</button>
							}
							if r.ErrorMsg != "" && string(r.Status) != "failed" {
								<div class="tooltip tooltip-left" data-tip={ r.ErrorMsg }>
									<span class="text-xs text-base-content/50 cursor-help">info</span>
								</div>
							}
						</td>
					</tr>
				}
				if len(records) == 0 {
					<tr id="notifications-empty">
						<td colspan="5" class="text-center text-base-content/50 py-4">No notifications yet.</td>
					</tr>
				}
				</tbody>
			</table>
		</div>

		if nextCursor != "" {
			<div class="p-3 text-center" id="load-more-container">
				<button class="btn btn-sm btn-ghost"
					hx-get={ templ.URL("/service/web/notifications/list?cursor=" + nextCursor) }
					hx-target="#load-more-container"
					hx-swap="outerHTML"
					data-testid="load-more">Load more</button>
			</div>
		}
	</div>
}
```

- [ ] **Step 3: Generate templ code**

Run: `templ generate pkg/views/pages/notifications.templ pkg/views/partials/notifications_table.templ`
Expected: `*_templ.go` files generated alongside the `.templ` files

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/views/...`
Expected: BUILD OK

- [ ] **Step 5: Commit**

```bash
git add pkg/views/pages/notifications.templ pkg/views/pages/notifications_templ.go pkg/views/partials/notifications_table.templ pkg/views/partials/notifications_table_templ.go
git commit -m "feat: add notifications page and table templates"
```

---

### Task 5: Add Web Route Handlers

**Files:**
- Create: `internal/modules/web/notification_webservice.go`

- [ ] **Step 1: Write the webservice file**

```go
package web

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var notificationWebserviceRules = []webservice.Rule{
	webservice.Get("/notifications", notificationsPage, route.WithNotAuth()),
	webservice.Get("/notifications/list", notificationsTable, route.WithNotAuth()),
	webservice.Post("/notifications/:id/retry", retryNotification, route.WithNotAuth()),
}

func getNotifyStore() *store.NotifyStore {
	if store.Database == nil || store.Database.GetDB() == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewNotifyStore(client)
}

func getUID(ctx fiber.Ctx) string {
	rc := route.GetRequestContext(ctx)
	if rc == nil {
		return ""
	}
	return rc.UID.String()
}

func notificationsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState("Not authenticated").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ns := getNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	records, nextCursor, err := ns.ListRecords(ctx.Context(), uid, store.ListNotifyRecordsOptions{Limit: 20})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load notifications").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return pages.NotificationsPage(pages.NotificationsPageParams{
		Records:    records,
		NextCursor: nextCursor,
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func notificationsTable(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState("Not authenticated").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	cursor := ctx.Query("cursor")

	ns := getNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	records, nextCursor, err := ns.ListRecords(ctx.Context(), uid, store.ListNotifyRecordsOptions{
		Limit:  20,
		Cursor: cursor,
	})
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load notifications").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.NotificationsTable(records, nextCursor).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func retryNotification(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid := getUID(ctx)
	if uid == "" {
		ctx.Type("html")
		return partials.EmptyState("Not authenticated").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.SendString("Invalid ID")
	}

	ns := getNotifyStore()
	if ns == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	rec, err := ns.GetRecord(ctx.Context(), id)
	if err != nil || rec == nil {
		ctx.Type("html")
		return partials.EmptyState("Record not found").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
	if rec.UID != uid {
		ctx.Status(fiber.StatusForbidden)
		return ctx.SendString("Not your notification")
	}
	if string(rec.Status) != "failed" {
		ctx.Type("html")
		return partials.EmptyState("Only failed notifications can be retried").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	payload := make(map[string]any)
	if rec.PayloadSnapshot != nil {
		for k, v := range rec.PayloadSnapshot {
			payload[k] = v
		}
	}

	notifyUid := types.Uid(rec.UID)
	err = notifypkg.GatewaySend(context.Background(), notifyUid, rec.TemplateID, []string{rec.Channel}, payload)

	if err != nil {
		failedRec := &gen.NotificationRecord{
			ID:         rec.ID,
			CreatedAt:  rec.CreatedAt,
			Channel:    rec.Channel,
			TemplateID: rec.TemplateID,
			Summary:    rec.Summary,
			ErrorMsg:   err.Error(),
		}
		failedRec.SetStatus("failed")
		ctx.Type("html")
		return partials.NotificationsTable([]*gen.NotificationRecord{failedRec}, "").
			Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	successRec := &gen.NotificationRecord{
		ID:         rec.ID,
		CreatedAt:  rec.CreatedAt,
		Channel:    rec.Channel,
		TemplateID: rec.TemplateID,
		Summary:    rec.Summary,
	}
	successRec.SetStatus("success")
	ctx.Type("html")
	return partials.NotificationsTable([]*gen.NotificationRecord{successRec}, "").
		Render(ctx.Context(), ctx.Response().BodyWriter())
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/web/...`
Expected: BUILD OK — but may fail if `gen.NotificationRecord` doesn't have `SetStatus()` method. If so, fix below.

- [ ] **Step 3: Fix retry response type if ent doesn't expose setters on generated structs**

If `gen.NotificationRecord` doesn't have `SetStatus`, adjust the retry handler to re-query the actual record after the send and use that:

Actually, looking at ent generated code patterns, the generated struct typically has exported fields that can be set directly. But since ent uses a pattern where fields might be wrapped, the safest approach is to return a fresh list from the store after retry instead of constructing synthetic records. Let's replace the retry response with a table reload pattern:

Replace the retry response at the end of `retryNotification` with:

```go
	// Reload the records list to show the new retry outcome
	records, nextCursor, listErr := ns.ListRecords(context.Background(), uid, store.ListNotifyRecordsOptions{Limit: 20})
	if listErr != nil {
		ctx.Type("html")
		return partials.EmptyState("Retried but failed to reload").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.NotificationsTable(records, nextCursor).
		Render(ctx.Context(), ctx.Response().BodyWriter())
```

And remove the synthetic record construction. This is simpler and always correct — the table gets a fresh render with the retry result at the top.

- [ ] **Step 4: Verify compilation after fix**

Run: `go build ./internal/modules/web/...`
Expected: BUILD OK

- [ ] **Step 5: Commit**

```bash
git add internal/modules/web/notification_webservice.go
git commit -m "feat: add notifications web route handlers with retry support"
```

---

### Task 6: Register Web Routes and Add Nav Link

**Files:**
- Modify: `internal/modules/web/module.go` — add `notificationWebserviceRules` to `Webservice()` and `Rules()`
- Modify: `pkg/views/layout/base.templ` — add "Notifications" nav link

- [ ] **Step 1: Register the webservice rules in module.go**

In `internal/modules/web/module.go`, add to `Webservice()`:

```go
func (moduleHandler) Webservice(app *fiber.App) {
	app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))
	module.Webservice(app, Name, webserviceRules)
	module.Webservice(app, Name, pipelineWebserviceRules)
	module.Webservice(app, Name, viewWebserviceRules)
	module.Webservice(app, Name, eventWebserviceRules)
	module.Webservice(app, Name, relationsWebserviceRules)
	module.Webservice(app, Name, notificationWebserviceRules)
}
```

And in `Rules()`:

```go
func (moduleHandler) Rules() []any {
	return []any{webserviceRules, pipelineWebserviceRules, viewWebserviceRules, eventWebserviceRules, relationsWebserviceRules, notificationWebserviceRules}
}
```

- [ ] **Step 2: Add nav link in base.templ**

In `pkg/views/layout/base.templ`, add after the Relations link (line 28):

```templ
<a href="/service/web/notifications" data-testid="nav-notifications" class="btn btn-ghost btn-sm">Notifications</a>
```

Place it between Relations and Configs:
```templ
<a href="/service/web/events" data-testid="nav-events" class="btn btn-ghost btn-sm">Events</a>
<a href="/service/web/notifications" data-testid="nav-notifications" class="btn btn-ghost btn-sm">Notifications</a>
<a href="/service/web/relations" data-testid="nav-relations" class="btn btn-ghost btn-sm">Relations</a>
```

- [ ] **Step 3: Regenerate base.templ**

Run: `templ generate pkg/views/layout/base.templ`
Expected: `base_templ.go` updated

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/modules/web/... ./pkg/views/...`
Expected: BUILD OK

- [ ] **Step 5: Commit**

```bash
git add internal/modules/web/module.go pkg/views/layout/base.templ pkg/views/layout/base_templ.go
git commit -m "feat: register notification routes and add nav link"
```

---

### Task 7: Run Format, Lint, and Tests

**Files:**
- All modified files

- [ ] **Step 1: Run format**

Run: `go tool task format`
Expected: No changes or clean formatting applied

- [ ] **Step 2: Run lint**

Run: `go tool task lint`
Expected: No lint errors

- [ ] **Step 3: Run unit tests**

Run: `go tool task test:short`
Expected: ALL PASS

- [ ] **Step 4: Run full build**

Run: `go tool task build`
Expected: BUILD OK, binary compiles

- [ ] **Step 5: Fix any issues**

If tests or lint fail, fix the specific issues, then re-run from Step 1.

- [ ] **Step 6: Commit (if fixes were needed)**

```bash
git add -u
git commit -m "chore: format, lint fixes for notification history"
```
