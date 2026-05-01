package store

import (
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
	"gorm.io/gorm"
)

type EventStore struct {
	db *gorm.DB
}

func NewEventStore(db *gorm.DB) *EventStore {
	return &EventStore{db: db}
}

func (s *EventStore) AppendDataEvent(event types.DataEvent) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	record := model.DataEvent{
		EventID:        event.EventID,
		EventType:      event.EventType,
		Source:         event.Source,
		Capability:     event.Capability,
		Operation:      event.Operation,
		Backend:        event.Backend,
		App:            event.App,
		EntityID:       event.EntityID,
		IdempotencyKey: event.IdempotencyKey,
		UID:            event.UID,
		Topic:          event.Topic,
		CreatedAt:      now,
	}
	if event.Data != nil {
		dataJSON := model.JSON{}
		_ = dataJSON.Scan(types.KV(event.Data))
		record.Data = dataJSON
	}
	return s.db.Create(&record).Error
}

func (s *EventStore) AppendEventOutbox(event types.DataEvent) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	payload := model.JSON{}
	_ = payload.Scan(types.KV{
		"event_id":        event.EventID,
		"event_type":      event.EventType,
		"source":          event.Source,
		"capability":      event.Capability,
		"operation":       event.Operation,
		"backend":         event.Backend,
		"app":             event.App,
		"entity_id":       event.EntityID,
		"idempotency_key": event.IdempotencyKey,
		"uid":             event.UID,
		"topic":           event.Topic,
	})
	record := model.EventOutbox{
		EventID:   event.EventID,
		Payload:   payload,
		Published: false,
		CreatedAt: now,
	}
	return s.db.Create(&record).Error
}

func (s *EventStore) MarkOutboxPublished(eventID string) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Model(&model.EventOutbox{}).
		Where("event_id = ?", eventID).
		Update("published", true).Error
}

// PipelineStore persists pipeline definitions, runs, step runs, and event consumptions.
type PipelineStore struct {
	db *gorm.DB
}

func NewPipelineStore(db *gorm.DB) *PipelineStore {
	return &PipelineStore{db: db}
}

func (s *PipelineStore) UpsertDefinition(name, description string, enabled bool, trigger, steps model.JSON) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	def := model.PipelineDefinition{
		Name:        name,
		Description: description,
		Enabled:     enabled,
		Trigger:     trigger,
		Steps:       steps,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.db.Where("name = ?", name).Assign(def).FirstOrCreate(&def).Error
}

func (s *PipelineStore) CreateRun(pipelineName, eventID, eventType string) (*model.PipelineRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	now := time.Now()
	run := model.PipelineRun{
		PipelineName: pipelineName,
		EventID:      eventID,
		EventType:    eventType,
		Status:       model.PipelineStart,
		StartedAt:    now,
		CreatedAt:    now,
	}
	if err := s.db.Create(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *PipelineStore) UpdateRunStatus(runID int64, status model.PipelineState, errMsg string) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"completed_at": now,
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return s.db.Model(&model.PipelineRun{}).Where("id = ?", runID).Updates(updates).Error
}

func (s *PipelineStore) CreateStepRun(runID int64, stepName, capability, operation string, params model.JSON) (*model.PipelineStepRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	now := time.Now()
	sr := model.PipelineStepRun{
		PipelineRunID: runID,
		StepName:     stepName,
		Capability:   capability,
		Operation:    operation,
		Params:       params,
		Status:       model.PipelineStart,
		StartedAt:    now,
		CreatedAt:    now,
	}
	if err := s.db.Create(&sr).Error; err != nil {
		return nil, err
	}
	return &sr, nil
}

func (s *PipelineStore) UpdateStepRun(stepRunID int64, status model.PipelineState, result model.JSON, errMsg string) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"completed_at": now,
	}
	if result != nil {
		updates["result"] = result
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return s.db.Model(&model.PipelineStepRun{}).Where("id = ?", stepRunID).Updates(updates).Error
}

func (s *PipelineStore) RecordConsumption(consumerName, eventID string) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	record := model.EventConsumption{
		ConsumerName: consumerName,
		EventID:      eventID,
		CreatedAt:    now,
	}
	return s.db.Create(&record).Error
}

func (s *PipelineStore) HasConsumed(consumerName, eventID string) (bool, error) {
	if s == nil || s.db == nil {
		return false, nil
	}
	var count int64
	err := s.db.Model(&model.EventConsumption{}).
		Where("consumer_name = ? AND event_id = ?", consumerName, eventID).
		Count(&count).Error
	return count > 0, err
}
