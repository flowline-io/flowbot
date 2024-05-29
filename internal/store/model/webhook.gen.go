// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNameWebhook = "webhook"

// Webhook mapped from table <webhook>
type Webhook struct {
	ID           int64        `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UID          string       `gorm:"column:uid;not null" json:"uid"`
	Topic        string       `gorm:"column:topic;not null" json:"topic"`
	Flag         string       `gorm:"column:flag;not null" json:"flag"`
	Secret       string       `gorm:"column:secret;not null" json:"secret"`
	TriggerCount int32        `gorm:"column:trigger_count;not null" json:"trigger_count"`
	State        WebhookState `gorm:"column:state;not null" json:"state"`
	CreatedAt    time.Time    `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt    time.Time    `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName Webhook's table name
func (*Webhook) TableName() string {
	return TableNameWebhook
}