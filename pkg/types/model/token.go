// Package model provides shared data types for UI views and transport.
package model

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// TokenItem represents a token row displayed in the token management UI.
type TokenItem struct {
	Token      string     `json:"token"`
	UID        types.Uid  `json:"uid"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiredAt  time.Time  `json:"expired_at"`
}
