package model

import "time"

const TableNameEventOutbox = "event_outbox"

type EventOutbox struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	EventID   string    `gorm:"column:event_id;not null;uniqueIndex" json:"event_id"`
	Payload   JSON      `gorm:"column:payload" json:"payload"`
	Published bool      `gorm:"column:published;not null;default:0" json:"published"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
}

func (*EventOutbox) TableName() string {
	return TableNameEventOutbox
}
