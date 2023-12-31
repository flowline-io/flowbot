// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNamePlatformBot = "platform_bots"

// PlatformBot mapped from table <platform_bots>
type PlatformBot struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	PlatformID int64     `gorm:"column:platform_id;not null" json:"platform_id"`
	BotID      int64     `gorm:"column:bot_id;not null" json:"bot_id"`
	Flag       string    `gorm:"column:flag;not null;default:0" json:"flag"`
	CreatedAt  time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName PlatformBot's table name
func (*PlatformBot) TableName() string {
	return TableNamePlatformBot
}
