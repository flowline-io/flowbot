package model

import "time"

// NotifyTemplateOverride is a channel-specific template body stored as JSON.
type NotifyTemplateOverride struct {
	Channel  string `json:"channel"`
	Format   string `json:"format"`
	Template string `json:"template"`
}

// NotifyTemplate represents a notification message template for UI display and forms.
type NotifyTemplate struct {
	ID              int64     `json:"id"`
	TemplateID      string    `json:"template_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	DefaultFormat   string    `json:"default_format"`
	DefaultTemplate string    `json:"default_template"`
	OverridesJSON   string    `json:"overrides_json"` // JSON array string for form display
	IsDefault       bool      `json:"is_default"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
