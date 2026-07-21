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
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
