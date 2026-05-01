package store

import (
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"gorm.io/gorm"
)

type AuditStore struct {
	db *gorm.DB
}

func NewAuditStore(db *gorm.DB) *AuditStore {
	return &AuditStore{db: db}
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

func (s *AuditStore) Write(entry AuditEntry) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	record := model.AuditLog{
		ActorType:    entry.ActorType,
		ActorID:      entry.ActorID,
		UID:          entry.UID,
		Topic:        entry.Topic,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceName: entry.ResourceName,
		Request:      entry.Request,
		Result:       entry.Result,
		Error:        entry.Error,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		CreatedAt:    now,
	}
	return s.db.Create(&record).Error
}

func (s *AuditStore) Success(actorType, actorID, uid, topic, action, resourceType, resourceName, ip, ua string) error {
	return s.Write(AuditEntry{
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

func (s *AuditStore) Rejected(actorType, actorID, uid, topic, action, resourceType, resourceName, reason, ip, ua string) error {
	return s.Write(AuditEntry{
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

func (s *AuditStore) Failed(actorType, actorID, uid, topic, action, resourceType, resourceName, errMsg, ip, ua string) error {
	return s.Write(AuditEntry{
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
