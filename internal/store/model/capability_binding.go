package model

import "time"

type CapabilityBinding struct {
	ID         int64     `json:"id"`
	Capability string    `json:"capability"`
	Backend    string    `json:"backend"`
	App        string    `json:"app"`
	Healthy    bool      `json:"healthy"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
