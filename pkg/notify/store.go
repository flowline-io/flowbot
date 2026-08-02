package notify

import (
	"context"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// ListNotifyRecordsOptions holds filters and pagination for listing notification records.
type ListNotifyRecordsOptions struct {
	Limit      int    // max 100, default 20
	Cursor     string // opaque cursor: ID value as string
	Channel    string // exact channel name filter; empty means any
	RuleID     string // exact rule_id filter; empty means any
	Status     string // exact status filter; empty means any
	UnreadOnly bool   // when true, only rows with nil read_at
}

// RecordParams holds fields for inserting a notification delivery record.
type RecordParams struct {
	UID           string
	Channel       string
	TemplateID    string
	Summary       string
	Status        string
	ErrorMsg      string
	RuleID        string
	CorrelationID string
	Payload       map[string]any
	EscalateAt    *time.Time
}

// NotifyRecords persists notification delivery history without exposing ORM types.
type NotifyRecords interface {
	Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg, ruleID string, payload map[string]any) (int64, error)
	RecordParams(ctx context.Context, p RecordParams) (int64, error)
	ListRecords(ctx context.Context, uid string, opts ListNotifyRecordsOptions) ([]model.NotificationRecord, string, error)
	CountUnread(ctx context.Context, uid, channel, status string) (int, error)
	MarkRead(ctx context.Context, uid string, ids ...int64) error
	MarkReadByCorrelation(ctx context.Context, uid, correlationID string) error
	ListDueDeferred(ctx context.Context, now time.Time, limit int) ([]model.NotificationRecord, error)
	UpdateRecordStatus(ctx context.Context, id int64, status, errorMsg string) error
	HasUnreadSuccessByCorrelation(ctx context.Context, uid, correlationID string) (bool, error)
	GetRecord(ctx context.Context, id int64) (*model.NotificationRecord, error)
	DeleteOldest(ctx context.Context, uid string, keepN int) error
}

// NotifyConfigStore persists global notify channels and templates used by the gateway.
type NotifyConfigStore interface {
	GetDefaultNotifyChannelRaw(ctx context.Context) (model.NotifyChannel, error)
	GetNotifyChannelByNameRaw(ctx context.Context, name string) (model.NotifyChannel, error)
	CreateNotifyChannel(ctx context.Context, name, protocol, uri string) (int64, error)
	GetDefaultNotifyTemplate(ctx context.Context) (model.NotifyTemplate, error)
	GetNotifyTemplateByTemplateID(ctx context.Context, templateID string) (model.NotifyTemplate, error)
	CreateNotifyTemplate(ctx context.Context, tmpl model.NotifyTemplate) (int64, error)
}

// NotifyUserConfig reads per-user notify:<channel> configuration.
type NotifyUserConfig interface {
	ListConfigByPrefix(ctx context.Context, uid types.Uid, topic, prefix string) ([]model.ConfigItem, error)
	ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error)
}

var (
	notifyRecordsMu sync.RWMutex
	notifyRecords   NotifyRecords

	notifyConfigMu sync.RWMutex
	notifyConfig   NotifyConfigStore

	notifyUserConfigMu sync.RWMutex
	notifyUserConfig   NotifyUserConfig
)

// SetNotifyRecords wires the persistence backend for notification delivery records.
func SetNotifyRecords(s NotifyRecords) {
	notifyRecordsMu.Lock()
	defer notifyRecordsMu.Unlock()
	notifyRecords = s
}

// GetNotifyStore returns the injected NotifyRecords backend, or nil when unset.
func GetNotifyStore() NotifyRecords {
	notifyRecordsMu.RLock()
	defer notifyRecordsMu.RUnlock()
	return notifyRecords
}

// SetNotifyConfigStore wires the global notify channel/template store.
func SetNotifyConfigStore(s NotifyConfigStore) {
	notifyConfigMu.Lock()
	defer notifyConfigMu.Unlock()
	notifyConfig = s
}

// GetNotifyConfigStore returns the injected notify config store, or nil when unset.
func GetNotifyConfigStore() NotifyConfigStore {
	notifyConfigMu.RLock()
	defer notifyConfigMu.RUnlock()
	return notifyConfig
}

// SetNotifyUserConfig wires the per-user notify config backend.
func SetNotifyUserConfig(s NotifyUserConfig) {
	notifyUserConfigMu.Lock()
	defer notifyUserConfigMu.Unlock()
	notifyUserConfig = s
}

// GetNotifyUserConfig returns the injected user notify config store, or nil when unset.
func GetNotifyUserConfig() NotifyUserConfig {
	notifyUserConfigMu.RLock()
	defer notifyUserConfigMu.RUnlock()
	return notifyUserConfig
}
