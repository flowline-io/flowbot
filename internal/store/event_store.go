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

func (s *EventStore) AppendDataEvent(ctx context.Context, event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
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
		c = c.SetData(map[string]any(event.Data))
	}
	if event.Tags != nil {
		c = c.SetTags(map[string]any(event.Tags))
	}
	_, err := c.Save(ctx)
	return err
}

func (s *EventStore) AppendEventOutbox(ctx context.Context, event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	payload := map[string]any{
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
	}
	if event.Tags != nil {
		payload["tags"] = map[string]any(event.Tags)
	}
	_, err := s.client.EventOutbox.Create().
		SetEventID(event.EventID).
		SetPayload(payload).
		SetPublished(false).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *EventStore) MarkOutboxPublished(ctx context.Context, eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
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

func (s *PipelineStore) UpsertDefinition(ctx context.Context, name, description string, enabled bool, trigger, steps model.JSON) error {
	if s == nil || s.client == nil {
		return nil
	}
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
			SetTrigger(map[string]any(trigger)).
			SetSteps(map[string]any(steps)).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		return err
	}
	_, err = s.client.PipelineDefinition.UpdateOneID(existing.ID).
		SetDescription(description).
		SetEnabled(enabled).
		SetTrigger(map[string]any(trigger)).
		SetSteps(map[string]any(steps)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *PipelineStore) CreateRun(ctx context.Context, pipelineName, eventID, eventType string) (*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
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

func (s *PipelineStore) UpdateRunStatus(ctx context.Context, runID int64, status model.PipelineState, errMsg string) error {
	if s == nil || s.client == nil {
		return nil
	}
	upd := s.client.PipelineRun.UpdateOneID(runID).
		SetStatus(int(status)).
		SetCompletedAt(time.Now())
	if errMsg != "" {
		upd = upd.SetError(errMsg)
	}
	_, err := upd.Save(ctx)
	return err
}

func (s *PipelineStore) CreateStepRun(ctx context.Context, runID int64, stepName, capability, operation string, params model.JSON, attempt int) (*model.PipelineStepRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	now := time.Now()
	sr, err := s.client.PipelineStepRun.Create().
		SetPipelineRunID(runID).
		SetStepName(stepName).
		SetCapability(capability).
		SetOperation(operation).
		SetParams(map[string]any(params)).
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

func (s *PipelineStore) UpdateStepRun(ctx context.Context, stepRunID int64, status model.PipelineState, result model.JSON, errMsg string, attempt int) error {
	if s == nil || s.client == nil {
		return nil
	}
	upd := s.client.PipelineStepRun.UpdateOneID(stepRunID).
		SetStatus(int(status)).
		SetAttempt(attempt)
	if status == model.PipelineDone || status == model.PipelineCancel {
		now := time.Now()
		upd = upd.SetCompletedAt(now)
	}
	if result != nil {
		upd = upd.SetResult(map[string]any(result))
	}
	if errMsg != "" {
		upd = upd.SetError(errMsg)
	}
	_, err := upd.Save(ctx)
	return err
}

func (s *PipelineStore) RecordConsumption(ctx context.Context, consumerName, eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.EventConsumption.Create().
		SetConsumerName(consumerName).
		SetEventID(eventID).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *PipelineStore) HasConsumed(ctx context.Context, consumerName, eventID string) (bool, error) {
	if s == nil || s.client == nil {
		return false, nil
	}
	count, err := s.client.EventConsumption.Query().
		Where(
			eventconsumption.ConsumerName(consumerName),
			eventconsumption.EventID(eventID),
		).
		Count(ctx)
	return count > 0, err
}

// SaveCheckpoint persists the intermediate pipeline run state.
func (s *PipelineStore) SaveCheckpoint(ctx context.Context, runID int64, data any) error {
	if s == nil || s.client == nil {
		return nil
	}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return err
	}
	var cp map[string]any
	if err := sonic.Unmarshal(raw, &cp); err != nil {
		return err
	}
	_, err = s.client.PipelineRun.UpdateOneID(runID).
		SetCheckpointData(cp).
		Save(ctx)
	return err
}

// UpdateRunHeartbeat refreshes the last_heartbeat timestamp for a running pipeline.
func (s *PipelineStore) UpdateRunHeartbeat(ctx context.Context, runID int64) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.PipelineRun.UpdateOneID(runID).
		SetLastHeartbeat(time.Now()).
		Save(ctx)
	return err
}

// GetIncompleteRuns returns pipeline runs that are in Start state and may need recovery.
func (s *PipelineStore) GetIncompleteRuns(ctx context.Context) ([]*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
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
func (s *PipelineStore) GetCheckpoint(ctx context.Context, runID int64, target any) error {
	if s == nil || s.client == nil {
		return nil
	}
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
func (s *PipelineStore) GetRun(ctx context.Context, runID int64) (*model.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
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
