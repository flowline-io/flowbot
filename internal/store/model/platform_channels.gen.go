// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNamePlatformChannel = "platform_channels"

// PlatformChannel mapped from table <platform_channels>
type PlatformChannel struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	PlatformID int64     `gorm:"column:platform_id;not null" json:"platform_id"`
	ChannelID  int64     `gorm:"column:channel_id;not null" json:"channel_id"`
	Flag       string    `gorm:"column:flag;not null;default:0" json:"flag"`
	CreatedAt  time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName PlatformChannel's table name
func (*PlatformChannel) TableName() string {
	return TableNamePlatformChannel
}
