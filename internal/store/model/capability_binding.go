package model

import "time"

const TableNameCapabilityBinding = "capability_bindings"

type CapabilityBinding struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Capability string    `gorm:"column:capability;not null" json:"capability"`
	Backend    string    `gorm:"column:backend;not null" json:"backend"`
	App        string    `gorm:"column:app;not null" json:"app"`
	Healthy    bool      `gorm:"column:healthy;not null;default:0" json:"healthy"`
	CreatedAt  time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (*CapabilityBinding) TableName() string {
	return TableNameCapabilityBinding
}
