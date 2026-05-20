package model

import (
	"time"
)

// Instruct mapped from table <instruct>
type Instruct struct {
	ID        int64            `json:"id"`
	No        string           `json:"no"`
	UID       string           `json:"uid"`
	Object    InstructObject   `json:"object"`
	Bot       string           `json:"bot"`
	Flag      string           `json:"flag"`
	Content   JSON             `json:"content"`
	Priority  InstructPriority `json:"priority"`
	State     InstructState    `json:"state"`
	ExpireAt  time.Time        `json:"expire_at"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
