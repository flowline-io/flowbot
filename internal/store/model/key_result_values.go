package model

import (
	"time"
)


// KeyResultValue mapped from table <key_result_values>
type KeyResultValue struct {
	ID          int64     `json:"id"`
	KeyResultID int64     `json:"key_result_id"`
	Value       int32     `json:"value"`
	Memo        string    `json:"memo"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
