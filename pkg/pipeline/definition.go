package pipeline

import (
	"context"
	"time"
)

// EditorDefinition is the YAML schema used by the pipeline editor UI.
// It supports multiple triggers (array) unlike the engine's Definition (single Trigger).
type EditorDefinition struct {
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Enabled     bool           `json:"enabled" yaml:"enabled"`
	Resumable   bool           `json:"resumable" yaml:"resumable"`
	Triggers    []TriggerEntry `json:"triggers" yaml:"triggers"`
	Steps       []Step         `json:"steps" yaml:"steps"`
}

// TriggerEntry represents a single trigger in the editor's triggers array.
type TriggerEntry struct {
	Enabled     bool           `json:"enabled" yaml:"enabled"`
	Type        string         `json:"type" yaml:"type"` // "event", "cron", "webhook"
	Event       string         `json:"event,omitempty" yaml:"event,omitempty"`
	Cron        string         `json:"cron,omitempty" yaml:"cron,omitempty"`
	CronTimeout string         `json:"cron_timeout,omitempty" yaml:"cron_timeout,omitempty"`
	Webhook     *WebhookConfig `json:"webhook,omitempty" yaml:"webhook,omitempty"`
}

// DefinitionRecord holds a published pipeline definition loaded from the database.
type DefinitionRecord struct {
	Name        string
	Description string
	YAML        string
	UpdatedAt   time.Time
}

// DefinitionReader is the interface for loading published definitions from a store.
type DefinitionReader interface {
	ListPublishedDefinitions(ctx context.Context) ([]DefinitionRecord, error)
}
