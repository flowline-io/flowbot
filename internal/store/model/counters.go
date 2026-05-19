package model

import (
	"time"
)


// Counter mapped from table <counters>
type Counter struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Topic     string    `json:"topic"`
	Flag      string    `json:"flag"`
	Digit     int64     `json:"digit"`
	Status    int32     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
