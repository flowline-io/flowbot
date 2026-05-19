package model

import (
	"time"
)


// Review mapped from table <reviews>
type Review struct {
	ID          int64               `json:"id"`
	UID         string              `json:"uid"`
	Topic       string              `json:"topic"`
	ObjectiveID int64               `json:"objective_id"`
	Type        ReviewType          `json:"type"`
	Rating      int32               `json:"rating"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Evaluations []*ReviewEvaluation `json:"evaluations"`
}
