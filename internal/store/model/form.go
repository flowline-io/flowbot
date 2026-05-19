package model

import (
	"time"
)

// Form mapped from table <form>
type Form struct {
	ID        int64     `json:"id"`
	FormID    string    `json:"form_id"`
	UID       string    `json:"uid"`
	Topic     string    `json:"topic"`
	Schema    JSON      `json:"schema"`
	Values    JSON      `json:"values"`
	Extra     JSON      `json:"extra"`
	State     FormState `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
