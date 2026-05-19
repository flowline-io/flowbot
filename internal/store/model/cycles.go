package model

import (
	"time"
)


// Cycle mapped from table <cycles>
type Cycle struct {
	ID         int64      `json:"id"`
	UID        string     `json:"uid"`
	Topic      string     `json:"topic"`
	Objectives string     `json:"objectives"`
	StartDate  time.Time  `json:"start_date"`
	EndDate    time.Time  `json:"end_date"`
	State      CycleState `json:"state"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
