// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNameObjective = "chatbot_objectives"

// Objective mapped from table <chatbot_objectives>
type Objective struct {
	ID           int32        `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UID          string       `gorm:"column:uid;not null" json:"uid"`
	Topic        string       `gorm:"column:topic;not null" json:"topic"`
	Sequence     int32        `gorm:"column:sequence;not null" json:"sequence"`
	Title        string       `gorm:"column:title;not null" json:"title"`
	Memo         string       `gorm:"column:memo;not null" json:"memo"`
	Motive       string       `gorm:"column:motive;not null" json:"motive"`
	Feasibility  string       `gorm:"column:feasibility;not null" json:"feasibility"`
	IsPlan       int32        `gorm:"column:is_plan;not null" json:"is_plan"`
	PlanStart    int64        `gorm:"column:plan_start;not null" json:"plan_start"`
	PlanEnd      int64        `gorm:"column:plan_end;not null" json:"plan_end"`
	TotalValue   int32        `gorm:"column:total_value;not null" json:"total_value"`
	CurrentValue int32        `gorm:"column:current_value;not null" json:"current_value"`
	Tag          string       `gorm:"column:tag;not null" json:"tag"`
	CreatedData  time.Time    `gorm:"column:created_data;not null" json:"created_data"`
	UpdatedDate  time.Time    `gorm:"column:updated_date;not null" json:"updated_date"`
	KeyResults   []*KeyResult `gorm:"foreignKey:objective_id" json:"key_results"`
	Reviews      []*Review    `gorm:"foreignKey:objective_id" json:"reviews"`
}

// TableName Objective's table name
func (*Objective) TableName() string {
	return TableNameObjective
}
