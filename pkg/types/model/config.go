// Package model provides shared data types for UI views and transport.
package model

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// ConfigItem represents a row from the configs database table.
type ConfigItem struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Topic     string    `json:"topic"`
	Key       string    `json:"key"`
	Value     types.KV  `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
