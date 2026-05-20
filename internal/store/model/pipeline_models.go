package model

import "time"

type PipelineDefinition struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Trigger     JSON      `json:"trigger"`
	Steps       JSON      `json:"steps"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PipelineRun struct {
	ID             int64         `json:"id"`
	PipelineName   string        `json:"pipeline_name"`
	EventID        string        `json:"event_id"`
	EventType      string        `json:"event_type"`
	Status         PipelineState `json:"status"`
	Error          string        `json:"error,omitempty"`
	CheckpointData JSON          `json:"checkpoint_data,omitempty"`
	LastHeartbeat  *time.Time    `json:"last_heartbeat,omitempty"`
	StartedAt      time.Time     `json:"started_at"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}

type PipelineStepRun struct {
	ID            int64         `json:"id"`
	PipelineRunID int64         `json:"pipeline_run_id"`
	StepName      string        `json:"step_name"`
	Capability    string        `json:"capability"`
	Operation     string        `json:"operation"`
	Params        JSON          `json:"params"`
	Result        JSON          `json:"result"`
	Attempt       int           `json:"attempt"`
	RetryConfig   JSON          `json:"retry_config,omitempty"`
	Status        PipelineState `json:"status"`
	Error         string        `json:"error,omitempty"`
	StartedAt     time.Time     `json:"started_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

type EventConsumption struct {
	ID           int64     `json:"id"`
	ConsumerName string    `json:"consumer_name"`
	EventID      string    `json:"event_id"`
	CreatedAt    time.Time `json:"created_at"`
}
