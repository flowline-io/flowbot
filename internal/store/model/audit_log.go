package model

import "time"

type AuditLog struct {
	ID           int64     `json:"id"`
	ActorType    string    `json:"actor_type"`
	ActorID      string    `json:"actor_id"`
	UID          string    `json:"uid"`
	Topic        string    `json:"topic"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	Request      JSON      `json:"request"`
	Result       string    `json:"result"`
	Error        string    `json:"error,omitempty"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
}
