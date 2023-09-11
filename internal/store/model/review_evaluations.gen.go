// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNameReviewEvaluation = "review_evaluations"

// ReviewEvaluation mapped from table <review_evaluations>
type ReviewEvaluation struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UID       string    `gorm:"column:uid;not null" json:"uid"`
	Topic     string    `gorm:"column:topic;not null" json:"topic"`
	ReviewID  int64     `gorm:"column:review_id;not null" json:"review_id"`
	Question  string    `gorm:"column:question;not null" json:"question"`
	Reason    string    `gorm:"column:reason;not null" json:"reason"`
	Solving   string    `gorm:"column:solving;not null" json:"solving"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName ReviewEvaluation's table name
func (*ReviewEvaluation) TableName() string {
	return TableNameReviewEvaluation
}