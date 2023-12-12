// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNamePlatformUser = "platform_users"

// PlatformUser mapped from table <platform_users>
type PlatformUser struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	PlatformID int64     `gorm:"column:platform_id;not null" json:"platform_id"`
	UserID     int64     `gorm:"column:user_id;not null" json:"user_id"`
	Flag       string    `gorm:"column:flag;not null" json:"flag"`
	Name       string    `gorm:"column:name;not null" json:"name"`
	Email      string    `gorm:"column:email;not null" json:"email"`
	AvatarURL  string    `gorm:"column:avatar_url;not null" json:"avatar_url"`
	IsBot      bool      `gorm:"column:is_bot;not null" json:"is_bot"`
	CreatedAt  time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName PlatformUser's table name
func (*PlatformUser) TableName() string {
	return TableNamePlatformUser
}
