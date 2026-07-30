package store

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/message"
	"github.com/flowline-io/flowbot/pkg/types"
)

// MessageStore persists chat messages.
type MessageStore struct {
	client *gen.Client
}

// NewMessageStore creates a MessageStore with the given ent client.
func NewMessageStore(client *gen.Client) *MessageStore {
	return &MessageStore{client: client}
}

// MessageStoreFromDB returns a MessageStore using the global database client.
func MessageStoreFromDB() *MessageStore {
	return NewMessageStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *MessageStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// GetMessage returns the message.
func (s *MessageStore) GetMessage(ctx context.Context, flag string) (*gen.Message, error) {
	m, err := s.client.Message.Query().Where(message.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get message: %w", err)
	}
	return m, nil
}

// GetMessageByPlatform returns the message by platform.
func (s *MessageStore) GetMessageByPlatform(ctx context.Context, platformId int64, platformMsgId string) (*gen.Message, error) {
	m, err := s.client.Message.Query().
		Where(message.PlatformIDEQ(platformId), message.PlatformMsgIDEQ(platformMsgId)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get message by platform: %w", err)
	}
	return m, nil
}

// GetMessagesBySession returns the messages by session.
func (s *MessageStore) GetMessagesBySession(ctx context.Context, session string) ([]*gen.Message, error) {
	messages, err := s.client.Message.Query().
		Where(message.SessionEQ(session)).
		Order(gen.Asc(message.FieldCreatedAt)).
		Limit(queryMaxMessageResults()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get messages by session: %w", err)
	}
	result := make([]*gen.Message, len(messages))
	copy(result, messages)
	return result, nil
}

// CreateMessage persists a new message.
func (s *MessageStore) CreateMessage(ctx context.Context, msg gen.Message) error {
	c := s.client.Message.Create().
		SetFlag(msg.Flag).
		SetPlatformID(msg.PlatformID).
		SetPlatformMsgID(msg.PlatformMsgID).
		SetTopic(msg.Topic).
		SetRole(msg.Role).
		SetSession(msg.Session).
		SetState(int(msg.State)).
		SetCreatedAt(msg.CreatedAt).
		SetUpdatedAt(msg.UpdatedAt)
	if msg.Content != nil {
		c = c.SetContent(map[string]any(msg.Content))
	}
	if msg.DeletedAt != nil {
		c = c.SetDeletedAt(*msg.DeletedAt)
	}
	_, err := c.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create message: %w", err)
	}
	return nil
}
