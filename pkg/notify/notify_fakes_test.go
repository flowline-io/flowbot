package notify

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

type memNotifyRecords struct {
	mu      sync.Mutex
	nextID  int64
	records map[int64]*model.NotificationRecord
}

func newMemNotifyRecords() *memNotifyRecords {
	return &memNotifyRecords{records: make(map[int64]*model.NotificationRecord)}
}

func (m *memNotifyRecords) Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg, ruleID string, payload map[string]any) (int64, error) {
	return m.RecordParams(ctx, RecordParams{
		UID: uid, Channel: channel, TemplateID: templateID, Summary: summary,
		Status: status, ErrorMsg: errorMsg, RuleID: ruleID, Payload: payload,
	})
}

func (m *memNotifyRecords) RecordParams(_ context.Context, p RecordParams) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := m.nextID
	rec := &model.NotificationRecord{
		ID:              id,
		UID:             p.UID,
		Channel:         p.Channel,
		TemplateID:      p.TemplateID,
		Summary:         p.Summary,
		Status:          p.Status,
		ErrorMsg:        p.ErrorMsg,
		RuleID:          p.RuleID,
		CorrelationID:   p.CorrelationID,
		PayloadSnapshot: p.Payload,
		EscalateAt:      p.EscalateAt,
		CreatedAt:       time.Now(),
	}
	m.records[id] = rec
	return id, nil
}

func memRecordMatches(rec *model.NotificationRecord, uid string, opts ListNotifyRecordsOptions) bool {
	if rec.UID != uid {
		return false
	}
	if opts.Channel != "" && rec.Channel != opts.Channel {
		return false
	}
	if opts.RuleID != "" && rec.RuleID != opts.RuleID {
		return false
	}
	if opts.Status != "" && rec.Status != opts.Status {
		return false
	}
	if opts.UnreadOnly && rec.ReadAt != nil {
		return false
	}
	if opts.Cursor != "" {
		cur, err := strconv.ParseInt(opts.Cursor, 10, 64)
		if err == nil && rec.ID >= cur {
			return false
		}
	}
	return true
}

func (m *memNotifyRecords) ListRecords(_ context.Context, uid string, opts ListNotifyRecordsOptions) ([]model.NotificationRecord, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	ids := make([]int64, 0, len(m.records))
	for id, rec := range m.records {
		if memRecordMatches(rec, uid, opts) {
			ids = append(ids, id)
		}
	}
	slices.SortFunc(ids, func(a, b int64) int { return cmp.Compare(b, a) })
	var next string
	if len(ids) > opts.Limit {
		next = strconv.FormatInt(ids[opts.Limit-1], 10)
		ids = ids[:opts.Limit]
	}
	out := make([]model.NotificationRecord, 0, len(ids))
	for _, id := range ids {
		out = append(out, *m.records[id])
	}
	return out, next, nil
}

func (m *memNotifyRecords) CountUnread(_ context.Context, uid, channel, status string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, rec := range m.records {
		if rec.UID != uid || rec.ReadAt != nil {
			continue
		}
		if channel != "" && rec.Channel != channel {
			continue
		}
		if status != "" && rec.Status != status {
			continue
		}
		n++
	}
	return n, nil
}

func (m *memNotifyRecords) MarkRead(_ context.Context, uid string, ids ...int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, id := range ids {
		rec, ok := m.records[id]
		if !ok || rec.UID != uid || rec.ReadAt != nil {
			continue
		}
		rec.ReadAt = &now
	}
	return nil
}

func (m *memNotifyRecords) MarkReadByCorrelation(_ context.Context, uid, correlationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, rec := range m.records {
		if rec.UID != uid || rec.CorrelationID != correlationID || rec.ReadAt != nil {
			continue
		}
		if rec.Status == "success" {
			rec.ReadAt = &now
		}
	}
	return nil
}

func (m *memNotifyRecords) ListDueDeferred(_ context.Context, now time.Time, limit int) ([]model.NotificationRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]model.NotificationRecord, 0)
	for _, rec := range m.records {
		if rec.Status != "deferred" || rec.EscalateAt == nil || rec.EscalateAt.After(now) {
			continue
		}
		out = append(out, *rec)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memNotifyRecords) UpdateRecordStatus(_ context.Context, id int64, status, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[id]
	if !ok {
		return types.ErrNotFound
	}
	rec.Status = status
	rec.ErrorMsg = errorMsg
	rec.EscalateAt = nil
	return nil
}

func (m *memNotifyRecords) HasUnreadSuccessByCorrelation(_ context.Context, uid, correlationID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rec := range m.records {
		if rec.UID == uid && rec.CorrelationID == correlationID && rec.Status == "success" && rec.ReadAt == nil {
			return true, nil
		}
	}
	return false, nil
}

func (m *memNotifyRecords) GetRecord(_ context.Context, id int64) (*model.NotificationRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[id]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *rec
	return &cp, nil
}

func (m *memNotifyRecords) DeleteOldest(_ context.Context, uid string, keepN int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if keepN <= 0 {
		return nil
	}
	ids := make([]int64, 0)
	for id, rec := range m.records {
		if rec.UID == uid {
			ids = append(ids, id)
		}
	}
	if len(ids) <= keepN {
		return nil
	}
	slices.Sort(ids)
	for _, id := range ids[:len(ids)-keepN] {
		delete(m.records, id)
	}
	return nil
}

type memNotifyConfigStore struct {
	mu              sync.Mutex
	nextID          int64
	channels        map[int64]model.NotifyChannel
	byName          map[string]int64
	templates       map[int64]model.NotifyTemplate
	byTemplateID    map[string]int64
	defaultChannel  int64
	defaultTemplate int64
}

func newMemNotifyConfigStore() *memNotifyConfigStore {
	return &memNotifyConfigStore{
		channels:     make(map[int64]model.NotifyChannel),
		byName:       make(map[string]int64),
		templates:    make(map[int64]model.NotifyTemplate),
		byTemplateID: make(map[string]int64),
	}
}

func (m *memNotifyConfigStore) GetDefaultNotifyChannelRaw(_ context.Context) (model.NotifyChannel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.channels[m.defaultChannel]
	if !ok || !ch.Enabled {
		return model.NotifyChannel{}, types.ErrNotFound
	}
	return ch, nil
}

func (m *memNotifyConfigStore) GetNotifyChannelByNameRaw(_ context.Context, name string) (model.NotifyChannel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byName[name]
	if !ok {
		return model.NotifyChannel{}, types.ErrNotFound
	}
	return m.channels[id], nil
}

func (m *memNotifyConfigStore) CreateNotifyChannel(_ context.Context, name, protocol, uri string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := m.nextID
	ch := model.NotifyChannel{ID: id, Name: name, Protocol: protocol, URI: uri, Enabled: true}
	m.channels[id] = ch
	m.byName[name] = id
	return id, nil
}

func (m *memNotifyConfigStore) updateChannel(id int64, name, protocol, uri string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := m.channels[id]
	if ch.Name != "" && ch.Name != name {
		delete(m.byName, ch.Name)
	}
	ch.Name = name
	ch.Protocol = protocol
	ch.URI = uri
	ch.Enabled = enabled
	m.channels[id] = ch
	m.byName[name] = id
}

func (m *memNotifyConfigStore) setDefaultChannel(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultChannel = id
	for k, ch := range m.channels {
		ch.IsDefault = k == id
		m.channels[k] = ch
	}
}

func (m *memNotifyConfigStore) GetDefaultNotifyTemplate(_ context.Context) (model.NotifyTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tmpl, ok := m.templates[m.defaultTemplate]
	if !ok {
		return model.NotifyTemplate{}, types.ErrNotFound
	}
	return tmpl, nil
}

func (m *memNotifyConfigStore) GetNotifyTemplateByTemplateID(_ context.Context, templateID string) (model.NotifyTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byTemplateID[templateID]
	if !ok {
		return model.NotifyTemplate{}, types.ErrNotFound
	}
	return m.templates[id], nil
}

func (m *memNotifyConfigStore) CreateNotifyTemplate(_ context.Context, tmpl model.NotifyTemplate) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := m.nextID
	tmpl.ID = id
	m.templates[id] = tmpl
	m.byTemplateID[tmpl.TemplateID] = id
	return id, nil
}

func (m *memNotifyConfigStore) setDefaultTemplate(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultTemplate = id
	for k, tmpl := range m.templates {
		tmpl.IsDefault = k == id
		m.templates[k] = tmpl
	}
}

type memNotifyUserConfig struct {
	mu   sync.Mutex
	data map[string]map[string]types.KV // uid -> key -> value
}

func newMemNotifyUserConfig() *memNotifyUserConfig {
	return &memNotifyUserConfig{data: make(map[string]map[string]types.KV)}
}

func (m *memNotifyUserConfig) ListConfigByPrefix(_ context.Context, uid types.Uid, _, prefix string) ([]model.ConfigItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := m.data[uid.String()]
	out := make([]model.ConfigItem, 0)
	for k, v := range keys {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			out = append(out, model.ConfigItem{UID: uid.String(), Key: k, Value: v})
		}
	}
	return out, nil
}

func (m *memNotifyUserConfig) ConfigGet(_ context.Context, uid types.Uid, _, key string) (types.KV, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := m.data[uid.String()]
	if keys == nil {
		return nil, types.ErrNotFound
	}
	v, ok := keys[key]
	if !ok {
		return nil, types.ErrNotFound
	}
	return v, nil
}

func (m *memNotifyUserConfig) configSet(uid types.Uid, key string, value types.KV) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uidStr := uid.String()
	if m.data[uidStr] == nil {
		m.data[uidStr] = make(map[string]types.KV)
	}
	m.data[uidStr][key] = value
}

// installNotifyTestStores wires in-memory notify backends for the test lifetime.
func installNotifyTestStores(t *testing.T) (*memNotifyRecords, *memNotifyConfigStore, *memNotifyUserConfig) {
	t.Helper()
	recs := newMemNotifyRecords()
	cfg := newMemNotifyConfigStore()
	user := newMemNotifyUserConfig()
	prevRecs := GetNotifyStore()
	prevCfg := GetNotifyConfigStore()
	prevUser := GetNotifyUserConfig()
	SetNotifyRecords(recs)
	SetNotifyConfigStore(cfg)
	SetNotifyUserConfig(user)
	t.Cleanup(func() {
		WaitForRecordAsyncForTest()
		SetNotifyRecords(prevRecs)
		SetNotifyConfigStore(prevCfg)
		SetNotifyUserConfig(prevUser)
	})
	return recs, cfg, user
}

func clearNotifyTestStores(t *testing.T) {
	t.Helper()
	prevRecs := GetNotifyStore()
	prevCfg := GetNotifyConfigStore()
	prevUser := GetNotifyUserConfig()
	SetNotifyRecords(nil)
	SetNotifyConfigStore(nil)
	SetNotifyUserConfig(nil)
	t.Cleanup(func() {
		WaitForRecordAsyncForTest()
		SetNotifyRecords(prevRecs)
		SetNotifyConfigStore(prevCfg)
		SetNotifyUserConfig(prevUser)
	})
}

func seedNotifyTestChannel(t *testing.T, name, protocol, uri string, enabled, isDefault bool) int64 {
	t.Helper()
	cfg, ok := GetNotifyConfigStore().(*memNotifyConfigStore)
	require.True(t, ok, "notify config store must be memNotifyConfigStore")
	id, err := cfg.CreateNotifyChannel(context.Background(), name, protocol, uri)
	require.NoError(t, err)
	if !enabled {
		cfg.updateChannel(id, name, protocol, uri, false)
	}
	if isDefault && enabled {
		cfg.setDefaultChannel(id)
	}
	return id
}

func seedNotifyTestTemplate(t *testing.T, tmpl model.NotifyTemplate, isDefault bool) int64 {
	t.Helper()
	cfg, ok := GetNotifyConfigStore().(*memNotifyConfigStore)
	require.True(t, ok, "notify config store must be memNotifyConfigStore")
	if tmpl.Name == "" {
		tmpl.Name = tmpl.TemplateID
	}
	if tmpl.Name == "" {
		tmpl.Name = "test-template"
	}
	if tmpl.DefaultFormat == "" {
		tmpl.DefaultFormat = "markdown"
	}
	if tmpl.DefaultTemplate == "" {
		tmpl.DefaultTemplate = "{{ .message }}"
	}
	if tmpl.OverridesJSON == "" {
		tmpl.OverridesJSON = "[]"
	}
	id, err := cfg.CreateNotifyTemplate(context.Background(), tmpl)
	require.NoError(t, err)
	if isDefault {
		cfg.setDefaultTemplate(id)
	}
	return id
}

func seedUserNotifyConfig(t *testing.T, uid types.Uid, channel, uri string) {
	t.Helper()
	uc, ok := GetNotifyUserConfig().(*memNotifyUserConfig)
	require.True(t, ok, "notify user config must be memNotifyUserConfig")
	uc.configSet(uid, notifyConfigKeyPrefix+channel, types.KV{"value": uri})
}

func setupNotifyTestDB(t *testing.T) {
	t.Helper()
	installNotifyTestStores(t)
}
