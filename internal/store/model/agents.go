package model

import (
	"time"
)

// Agent mapped from table <agents>
type Agent struct {
	ID             int64     `json:"id"`
	UID            string    `json:"uid"`
	Topic          string    `json:"topic"`
	Hostid         string    `json:"hostid"`
	Hostname       string    `json:"hostname"`
	OnlineDuration int32     `json:"online_duration"`
	LastOnlineAt   time.Time `json:"last_online_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
