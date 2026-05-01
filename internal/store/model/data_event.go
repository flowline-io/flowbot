package model

import "time"

const TableNameDataEvent = "data_events"

type DataEvent struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	EventID        string    `gorm:"column:event_id;not null;uniqueIndex" json:"event_id"`
	EventType      string    `gorm:"column:event_type;not null;index" json:"event_type"`
	Source         string    `gorm:"column:source;not null;default:''" json:"source"`
	Capability     string    `gorm:"column:capability;not null;default:''" json:"capability"`
	Operation      string    `gorm:"column:operation;not null;default:''" json:"operation"`
	Backend        string    `gorm:"column:backend;not null;default:''" json:"backend"`
	App            string    `gorm:"column:app;not null;default:''" json:"app"`
	EntityID       string    `gorm:"column:entity_id;not null;default:''" json:"entity_id"`
	IdempotencyKey string    `gorm:"column:idempotency_key;not null;default:''" json:"idempotency_key"`
	UID            string    `gorm:"column:uid;not null;default:''" json:"uid"`
	Topic          string    `gorm:"column:topic;not null;default:''" json:"topic"`
	Data           JSON      `gorm:"column:data" json:"data"`
	CreatedAt      time.Time `gorm:"column:created_at;not null" json:"created_at"`
}

func (*DataEvent) TableName() string {
	return TableNameDataEvent
}
