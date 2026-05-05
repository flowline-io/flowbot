package store

import (
	"encoding/json"
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

func (s *PipelineStore) CreateStepRun(runID int64, stepName, capability, operation string, params model.JSON, attempt int) (*model.PipelineStepRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	now := time.Now()
	sr := model.PipelineStepRun{
		PipelineRunID: runID,
		StepName:      stepName,
		Capability:    capability,
		Operation:     operation,
		Params:        params,
		Attempt:       attempt,
		Status:        model.PipelineStart,
		StartedAt:     now,
		CreatedAt:     now,
	}
	if err := s.db.Create(&sr).Error; err != nil {
		return nil, err
	}
	return &sr, nil
}

func (s *PipelineStore) UpdateStepRun(stepRunID int64, status model.PipelineState, result model.JSON, errMsg string, attempt int) error {
	if s == nil || s.db == nil {
		return nil
	}
	updates := map[string]any{
		"status":  status,
		"attempt": attempt,
	}
	if status == model.PipelineDone || status == model.PipelineCancel {
		now := time.Now()
		updates["completed_at"] = now
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

// SaveCheckpoint persists the intermediate pipeline run state.
func (s *PipelineStore) SaveCheckpoint(runID int64, data any) error {
	if s == nil || s.db == nil {
		return nil
	}
	cp := model.JSON{}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err := cp.Scan(raw); err != nil {
		return err
	}
	return s.db.Model(&model.PipelineRun{}).
		Where("id = ?", runID).
		Update("checkpoint_data", cp).Error
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running pipeline.
func (s *PipelineStore) UpdateRunHeartbeat(runID int64) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	return s.db.Model(&model.PipelineRun{}).
		Where("id = ?", runID).
		Update("last_heartbeat", now).Error
}

// GetIncompleteRuns returns pipeline runs that are in Start state and may need recovery.
func (s *PipelineStore) GetIncompleteRuns() ([]*model.PipelineRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var runs []*model.PipelineRun
	err := s.db.Where("status = ?", model.PipelineStart).
		Order("created_at ASC").
		Find(&runs).Error
	return runs, err
}

// GetCheckpoint loads the checkpoint data for a pipeline run.
func (s *PipelineStore) GetCheckpoint(runID int64, target any) error {
	if s == nil || s.db == nil {
		return nil
	}
	var run model.PipelineRun
	if err := s.db.Select("checkpoint_data").Where("id = ?", runID).First(&run).Error; err != nil {
		return err
	}
	if run.CheckpointData == nil {
		return nil
	}
	raw, err := json.Marshal(run.CheckpointData)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

// GetRun returns a pipeline run by ID.
func (s *PipelineStore) GetRun(runID int64) (*model.PipelineRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var run model.PipelineRun
	if err := s.db.Where("id = ?", runID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}
