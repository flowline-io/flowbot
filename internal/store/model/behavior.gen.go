// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNameBehavior = "behavior"

// Behavior mapped from table <behavior>
type Behavior struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UID       string    `gorm:"column:uid;not null" json:"uid"`
	Flag      string    `gorm:"column:flag;not null" json:"flag"`
	Count_    int32     `gorm:"column:count;not null" json:"count"`
	Extra     *JSON     `gorm:"column:extra" json:"extra"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName Behavior's table name
func (*Behavior) TableName() string {
	return TableNameBehavior
}
