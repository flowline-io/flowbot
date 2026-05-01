package model

import "time"

const TableNameAuditLog = "audit_logs"

type AuditLog struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	ActorType    string    `gorm:"column:actor_type;not null;default:user" json:"actor_type"`
	ActorID      string    `gorm:"column:actor_id;not null;default:''" json:"actor_id"`
	UID          string    `gorm:"column:uid;not null;default:''" json:"uid"`
	Topic        string    `gorm:"column:topic;not null;default:''" json:"topic"`
	Action       string    `gorm:"column:action;not null" json:"action"`
	ResourceType string    `gorm:"column:resource_type;not null;default:''" json:"resource_type"`
	ResourceName string    `gorm:"column:resource_name;not null;default:''" json:"resource_name"`
	Request      JSON      `gorm:"column:request" json:"request"`
	Result       string    `gorm:"column:result;not null;default:success" json:"result"`
	Error        string    `gorm:"column:error" json:"error,omitempty"`
	IPAddress    string    `gorm:"column:ip_address;not null;default:''" json:"ip_address"`
	UserAgent    string    `gorm:"column:user_agent;not null;default:''" json:"user_agent"`
	CreatedAt    time.Time `gorm:"column:created_at;not null" json:"created_at"`
}

func (*AuditLog) TableName() string {
	return TableNameAuditLog
}
