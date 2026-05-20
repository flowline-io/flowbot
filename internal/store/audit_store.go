// Package store provides database storage implementations.
package store

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/model"
)

type AuditStore struct {
	client *gen.Client
}

func NewAuditStore(client *gen.Client) *AuditStore {
	return &AuditStore{client: client}
}

type AuditEntry struct {
	ActorType    string
	ActorID      string
	UID          string
	Topic        string
	Action       string
	ResourceType string
	ResourceName string
	Request      model.JSON
	Result       string
	Error        string
	IPAddress    string
	UserAgent    string
}

func (s *AuditStore) Write(ctx context.Context, entry AuditEntry) error {
	if s == nil || s.client == nil {
		return nil
	}
	now := time.Now()
	_, err := s.client.AuditLog.Create().
		SetAction(entry.Action).
		SetTargetType(entry.ResourceType).
		SetTargetID(entry.ResourceName).
		SetActorUID(entry.ActorType + ":" + entry.ActorID).
		SetDetails(map[string]any{
			"actor_type":    entry.ActorType,
			"actor_id":      entry.ActorID,
			"uid":           entry.UID,
			"topic":         entry.Topic,
			"action":        entry.Action,
			"resource_type": entry.ResourceType,
			"resource_name": entry.ResourceName,
			"request":       map[string]any(entry.Request),
			"result":        entry.Result,
			"error":         entry.Error,
			"ip_address":    entry.IPAddress,
			"user_agent":    entry.UserAgent,
		}).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *AuditStore) Success(ctx context.Context, actorType, actorID, uid, topic, action, resourceType, resourceName, ip, ua string) error {
	return s.Write(ctx, AuditEntry{
		ActorType:    actorType,
		ActorID:      actorID,
		UID:          uid,
		Topic:        topic,
		Action:       action,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Result:       "success",
		IPAddress:    ip,
		UserAgent:    ua,
	})
}

func (s *AuditStore) Rejected(ctx context.Context, actorType, actorID, uid, topic, action, resourceType, resourceName, reason, ip, ua string) error {
	return s.Write(ctx, AuditEntry{
		ActorType:    actorType,
		ActorID:      actorID,
		UID:          uid,
		Topic:        topic,
		Action:       action,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Result:       "rejected",
		Error:        reason,
		IPAddress:    ip,
		UserAgent:    ua,
	})
}

func (s *AuditStore) Failed(ctx context.Context, actorType, actorID, uid, topic, action, resourceType, resourceName, errMsg, ip, ua string) error {
	return s.Write(ctx, AuditEntry{
		ActorType:    actorType,
		ActorID:      actorID,
		UID:          uid,
		Topic:        topic,
		Action:       action,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Result:       "failed",
		Error:        errMsg,
		IPAddress:    ip,
		UserAgent:    ua,
	})
}
