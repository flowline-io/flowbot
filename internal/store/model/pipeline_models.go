package model

import "time"

const TableNamePipelineDefinition = "pipeline_definitions"

type PipelineDefinition struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Name        string    `gorm:"column:name;not null;uniqueIndex" json:"name"`
	Description string    `gorm:"column:description" json:"description"`
	Enabled     bool      `gorm:"column:enabled;not null;default:1" json:"enabled"`
	Trigger     JSON      `gorm:"column:trigger" json:"trigger"`
	Steps       JSON      `gorm:"column:steps" json:"steps"`
	CreatedAt   time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (*PipelineDefinition) TableName() string {
	return TableNamePipelineDefinition
}

const TableNamePipelineRun = "pipeline_runs"

type PipelineRun struct {
	ID           int64         `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	PipelineName string        `gorm:"column:pipeline_name;not null;index" json:"pipeline_name"`
	EventID      string        `gorm:"column:event_id;not null;uniqueIndex" json:"event_id"`
	EventType    string        `gorm:"column:event_type;not null;default:''" json:"event_type"`
	Status       PipelineState `gorm:"column:status;not null;default:0" json:"status"`
	Error        string        `gorm:"column:error" json:"error,omitempty"`
	StartedAt    time.Time     `gorm:"column:started_at" json:"started_at"`
	CompletedAt  *time.Time    `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt    time.Time     `gorm:"column:created_at;not null" json:"created_at"`
}

func (*PipelineRun) TableName() string {
	return TableNamePipelineRun
}

const TableNamePipelineStepRun = "pipeline_step_runs"

type PipelineStepRun struct {
	ID            int64         `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	PipelineRunID int64         `gorm:"column:pipeline_run_id;not null;index" json:"pipeline_run_id"`
	StepName      string        `gorm:"column:step_name;not null" json:"step_name"`
	Capability    string        `gorm:"column:capability;not null;default:''" json:"capability"`
	Operation     string        `gorm:"column:operation;not null;default:''" json:"operation"`
	Params        JSON          `gorm:"column:params" json:"params"`
	Result        JSON          `gorm:"column:result" json:"result"`
	Status        PipelineState `gorm:"column:status;not null;default:0" json:"status"`
	Error         string        `gorm:"column:error" json:"error,omitempty"`
	StartedAt     time.Time     `gorm:"column:started_at" json:"started_at"`
	CompletedAt   *time.Time    `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt     time.Time     `gorm:"column:created_at;not null" json:"created_at"`
}

func (*PipelineStepRun) TableName() string {
	return TableNamePipelineStepRun
}

const TableNameEventConsumption = "event_consumptions"

type EventConsumption struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	ConsumerName string    `gorm:"column:consumer_name;not null;index:idx_consumer_event,unique" json:"consumer_name"`
	EventID      string    `gorm:"column:event_id;not null;index:idx_consumer_event,unique" json:"event_id"`
	CreatedAt    time.Time `gorm:"column:created_at;not null" json:"created_at"`
}

func (*EventConsumption) TableName() string {
	return TableNameEventConsumption
}
