package model

import (
	"time"
)

// CounterRecord mapped from table <counter_records>
type CounterRecord struct {
	CounterID int64     `json:"counter_id"`
	Digit     int32     `json:"digit"`
	CreatedAt time.Time `json:"created_at"`
}
