package model

import (
	"time"
)


// KeyResult mapped from table <key_results>
type KeyResult struct {
	ID              int64             `json:"id"`
	UID             string            `json:"uid"`
	Topic           string            `json:"topic"`
	ObjectiveID     int64             `json:"objective_id"`
	Sequence        int32             `json:"sequence"`
	Title           string            `json:"title"`
	Memo            string            `json:"memo"`
	InitialValue    int32             `json:"initial_value"`
	TargetValue     int32             `json:"target_value"`
	CurrentValue    int32             `json:"current_value"`
	ValueMode       ValueModeType     `json:"value_mode"`
	Tag             string            `json:"tag"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	KeyResultValues []*KeyResultValue `json:"key_result_values"`
	Todos           []*Todo           `json:"todos"`
}
