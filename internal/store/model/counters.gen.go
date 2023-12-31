// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNameCounter = "counters"

// Counter mapped from table <counters>
type Counter struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UID       string    `gorm:"column:uid;not null" json:"uid"`
	Topic     string    `gorm:"column:topic;not null" json:"topic"`
	Flag      string    `gorm:"column:flag;not null" json:"flag"`
	Digit     int64     `gorm:"column:digit;not null" json:"digit"`
	Status    int32     `gorm:"column:status;not null" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName Counter's table name
func (*Counter) TableName() string {
	return TableNameCounter
}
