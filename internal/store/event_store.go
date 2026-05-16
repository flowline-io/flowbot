package store

import (
	"context"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventconsumption"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventoutbox"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
)

type EventStore struct {
	client *gen.Client
}

func NewEventStore(client *gen.Client) *EventStore {
	return &EventStore{client: client}
}

func (s *EventStore) AppendDataEvent(event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	c := s.client.DataEvent.Create().
		SetEventID(event.EventID).
		SetEventType(event.EventType).
		SetSource(event.Source).
		SetCapability(event.Capability).
		SetOperation(event.Operation).
		SetBackend(event.Backend).
		SetApp(event.App).
		SetEntityID(event.EntityID).
		SetIdempotencyKey(event.IdempotencyKey).
		SetUID(event.UID).
		SetTopic(event.Topic).
		SetCreatedAt(time.Now())
	if event.Data != nil {
		c = c.SetData(map[string]interface{}(event.Data))
	}
	_, err := c.Save(ctx)
	return err
}

func (s *EventStore) AppendEventOutbox(event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	_, err := s.client.EventOutbox.Create().
		SetEventID(event.EventID).
		SetPayload(map[string]interface{}{
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
		}).
		SetPublished(false).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *EventStore) MarkOutboxPublished(eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	_, err := s.client.EventOutbox.Update().
		Where(eventoutbox.EventID(eventID)).
		SetPublished(true).
		Save(ctx)
	return err
}

// PipelineStore persists pipeline definitions, runs, step runs, and event consumptions.
type PipelineStore struct {
	client *gen.Client
}

func NewPipelineStore(client *gen.Client) *PipelineStore {
	return &PipelineStore{client: client}
}

func (s *PipelineStore) UpsertDefinition(name, description string, enabled bool, trigger, steps model.JSON) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	existing, err := s.client.PipelineDefinition.Query().
		Where(pipelinedefinition.Name(name)).
		Only(ctx)
	if err != nil {
		if !gen.IsNotFound(err) {
			return err
		}
		now := time.Now()
		_, err = s.client.PipelineDefinition.Create().
			SetName(name).
			SetDescription(description).
			SetEnabled(enabled).
			SetTrigger(map[string]interface{}(trigger)).
			SetSteps(map[string]interface{}(steps)).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		return err
	}
	_, err = s.client.PipelineDefinition.UpdateOneID(existing.ID).
		SetDescription(description).
		SetEnabled(enabled).
		SetTrigger(map[string]interface{}(trigger)).
		SetSteps(map[string]interface{}(steps)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *PipelineStore) CreateRun(pipelineName, eventID, eventType string) (*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	ctx := context.Background()
	now := time.Now()
	run, err := s.client.PipelineRun.Create().
		SetPipelineName(pipelineName).
		SetEventID(eventID).
		SetEventType(eventType).
		SetStatus(int(model.PipelineStart)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &model.PipelineRun{
		ID:             run.ID,
		PipelineName:   run.PipelineName,
		EventID:        run.EventID,
		EventType:      run.EventType,
		Status:         model.PipelineState(run.Status),
		Error:          run.Error,
		CheckpointData: model.JSON(run.CheckpointData),
		LastHeartbeat:  run.LastHeartbeat,
		StartedAt:      run.StartedAt,
		CompletedAt:    run.CompletedAt,
		CreatedAt:      run.CreatedAt,
	}, nil
}

func (s *PipelineStore) UpdateRunStatus(runID int64, status model.PipelineState, errMsg string) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	upd := s.client.PipelineRun.UpdateOneID(runID).
		SetStatus(int(status)).
		SetCompletedAt(time.Now())
	if errMsg != "" {
		upd = upd.SetError(errMsg)
	}
	_, err := upd.Save(ctx)
	return err
}

func (s *PipelineStore) CreateStepRun(runID int64, stepName, capability, operation string, params model.JSON, attempt int) (*model.PipelineStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	ctx := context.Background()
	now := time.Now()
	sr, err := s.client.PipelineStepRun.Create().
		SetPipelineRunID(runID).
		SetStepName(stepName).
		SetCapability(capability).
		SetOperation(operation).
		SetParams(map[string]interface{}(params)).
		SetAttempt(attempt).
		SetStatus(int(model.PipelineStart)).
		SetStartedAt(now).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &model.PipelineStepRun{
		ID:            sr.ID,
		PipelineRunID: sr.PipelineRunID,
		StepName:      sr.StepName,
		Capability:    sr.Capability,
		Operation:     sr.Operation,
		Params:        model.JSON(sr.Params),
		Result:        model.JSON(sr.Result),
		Attempt:       sr.Attempt,
		RetryConfig:   model.JSON(sr.RetryConfig),
		Status:        model.PipelineState(sr.Status),
		Error:         sr.Error,
		StartedAt:     sr.StartedAt,
		CompletedAt:   sr.CompletedAt,
		CreatedAt:     sr.CreatedAt,
	}, nil
}

func (s *PipelineStore) UpdateStepRun(stepRunID int64, status model.PipelineState, result model.JSON, errMsg string, attempt int) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	upd := s.client.PipelineStepRun.UpdateOneID(stepRunID).
		SetStatus(int(status)).
		SetAttempt(attempt)
	if status == model.PipelineDone || status == model.PipelineCancel {
		now := time.Now()
		upd = upd.SetCompletedAt(now)
	}
	if result != nil {
		upd = upd.SetResult(map[string]interface{}(result))
	}
	if errMsg != "" {
		upd = upd.SetError(errMsg)
	}
	_, err := upd.Save(ctx)
	return err
}

func (s *PipelineStore) RecordConsumption(consumerName, eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	_, err := s.client.EventConsumption.Create().
		SetConsumerName(consumerName).
		SetEventID(eventID).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *PipelineStore) HasConsumed(consumerName, eventID string) (bool, error) {
	if s == nil || s.client == nil {
		return false, nil
	}
	ctx := context.Background()
	count, err := s.client.EventConsumption.Query().
		Where(
			eventconsumption.ConsumerName(consumerName),
			eventconsumption.EventID(eventID),
		).
		Count(ctx)
	return count > 0, err
}

// SaveCheckpoint persists the intermediate pipeline run state.
func (s *PipelineStore) SaveCheckpoint(runID int64, data any) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	raw, err := sonic.Marshal(data)
	if err != nil {
		return err
	}
	var cp map[string]interface{}
	if err := sonic.Unmarshal(raw, &cp); err != nil {
		return err
	}
	_, err = s.client.PipelineRun.UpdateOneID(runID).
		SetCheckpointData(cp).
		Save(ctx)
	return err
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running pipeline.
func (s *PipelineStore) UpdateRunHeartbeat(runID int64) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	_, err := s.client.PipelineRun.UpdateOneID(runID).
		SetLastHeartbeat(time.Now()).
		Save(ctx)
	return err
}

// GetIncompleteRuns returns pipeline runs that are in Start state and may need recovery.
func (s *PipelineStore) GetIncompleteRuns() ([]*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	ctx := context.Background()
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.Status(int(model.PipelineStart))).
		Order(pipelinerun.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*model.PipelineRun, len(runs))
	for i, r := range runs {
		result[i] = &model.PipelineRun{
			ID:             r.ID,
			PipelineName:   r.PipelineName,
			EventID:        r.EventID,
			EventType:      r.EventType,
			Status:         model.PipelineState(r.Status),
			Error:          r.Error,
			CheckpointData: model.JSON(r.CheckpointData),
			LastHeartbeat:  r.LastHeartbeat,
			StartedAt:      r.StartedAt,
			CompletedAt:    r.CompletedAt,
			CreatedAt:      r.CreatedAt,
		}
	}
	return result, nil
}

// GetCheckpoint loads the checkpoint data for a pipeline run.
func (s *PipelineStore) GetCheckpoint(runID int64, target any) error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx := context.Background()
	run, err := s.client.PipelineRun.Query().
		Where(pipelinerun.ID(runID)).
		Select(pipelinerun.FieldCheckpointData).
		Only(ctx)
	if err != nil {
		return err
	}
	if run.CheckpointData == nil {
		return nil
	}
	raw, err := sonic.Marshal(run.CheckpointData)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(raw, target)
}

// GetRun returns a pipeline run by ID.
func (s *PipelineStore) GetRun(runID int64) (*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	ctx := context.Background()
	run, err := s.client.PipelineRun.Query().
		Where(pipelinerun.ID(runID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return &model.PipelineRun{
		ID:             run.ID,
		PipelineName:   run.PipelineName,
		EventID:        run.EventID,
		EventType:      run.EventType,
		Status:         model.PipelineState(run.Status),
		Error:          run.Error,
		CheckpointData: model.JSON(run.CheckpointData),
		LastHeartbeat:  run.LastHeartbeat,
		StartedAt:      run.StartedAt,
		CompletedAt:    run.CompletedAt,
		CreatedAt:      run.CreatedAt,
	}, nil
}
