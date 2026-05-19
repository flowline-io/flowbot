package model

import (
	"time"
)

// Objective mapped from table <objectives>
type Objective struct {
	ID           int64        `json:"id"`
	UID          string       `json:"uid"`
	Topic        string       `json:"topic"`
	Sequence     int32        `json:"sequence"`
	Progress     int32        `json:"progress"`
	Title        string       `json:"title"`
	Memo         string       `json:"memo"`
	Motive       string       `json:"motive"`
	Feasibility  string       `json:"feasibility"`
	IsPlan       int32        `json:"is_plan"`
	PlanStart    time.Time    `json:"plan_start"`
	PlanEnd      time.Time    `json:"plan_end"`
	TotalValue   int32        `json:"total_value"`
	CurrentValue int32        `json:"current_value"`
	Tag          string       `json:"tag"`
	CreatedData  time.Time    `json:"created_data"`
	UpdatedDate  time.Time    `json:"updated_date"`
	KeyResults   []*KeyResult `json:"key_results"`
	Reviews      []*Review    `json:"reviews"`
}
