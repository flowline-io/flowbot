// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNamePlatformChannelUser = "platform_channel_users"

// PlatformChannelUser mapped from table <platform_channel_users>
type PlatformChannelUser struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	PlatformID  int64     `gorm:"column:platform_id;not null" json:"platform_id"`
	ChannelFlag string    `gorm:"column:channel_flag;not null" json:"channel_flag"`
	UserFlag    string    `gorm:"column:user_flag;not null" json:"user_flag"`
	CreatedAt   time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName PlatformChannelUser's table name
func (*PlatformChannelUser) TableName() string {
	return TableNamePlatformChannelUser
}
