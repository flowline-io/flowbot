package model

import (
	"time"
)


// ReviewEvaluation mapped from table <review_evaluations>
type ReviewEvaluation struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Topic     string    `json:"topic"`
	ReviewID  int64     `json:"review_id"`
	Question  string    `json:"question"`
	Reason    string    `json:"reason"`
	Solving   string    `json:"solving"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
