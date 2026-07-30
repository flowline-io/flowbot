package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agent"
	"github.com/flowline-io/flowbot/pkg/types"
)

// RuntimeAgentStore persists host/runtime agent heartbeats.
type RuntimeAgentStore struct {
	client *gen.Client
}

// NewRuntimeAgentStore creates a RuntimeAgentStore with the given ent client.
func NewRuntimeAgentStore(client *gen.Client) *RuntimeAgentStore {
	return &RuntimeAgentStore{client: client}
}

// RuntimeAgentStoreFromDB returns a RuntimeAgentStore using the global database client.
func RuntimeAgentStoreFromDB() *RuntimeAgentStore {
	return NewRuntimeAgentStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *RuntimeAgentStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// GetAgents returns the agents.
func (s *RuntimeAgentStore) GetAgents(ctx context.Context) ([]*gen.Agent, error) {
	agents, err := s.client.Agent.Query().Order(gen.Asc(agent.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getagents: %w", err)
	}
	result := make([]*gen.Agent, len(agents))
	copy(result, agents)
	return result, nil
}

// GetAgentByHostid returns the agent by hostid.
func (s *RuntimeAgentStore) GetAgentByHostid(ctx context.Context, uid types.Uid, topic, hostid string) (*gen.Agent, error) {
	ag, err := s.client.Agent.Query().
		Where(agent.UID(uid.String()), agent.TopicEQ(topic), agent.HostidEQ(hostid)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getagentbyhostid: %w", err)
	}
	return ag, nil
}

// CreateAgent persists a new agent.
func (s *RuntimeAgentStore) CreateAgent(ctx context.Context, agentModel *gen.Agent) (int64, error) {
	ag, err := s.client.Agent.Create().
		SetUID(agentModel.UID).
		SetTopic(agentModel.Topic).
		SetHostid(agentModel.Hostid).
		SetHostname(agentModel.Hostname).
		SetOnlineDuration(agentModel.OnlineDuration).
		SetLastOnlineAt(agentModel.LastOnlineAt).
		SetCreatedAt(agentModel.CreatedAt).
		SetUpdatedAt(agentModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createagent: %w", err)
	}
	return ag.ID, nil
}

// UpdateAgentLastOnlineAt updates the agent last online at.
func (s *RuntimeAgentStore) UpdateAgentLastOnlineAt(ctx context.Context, uid types.Uid, topic, hostid string, lastOnlineAt time.Time) error {
	_, err := s.client.Agent.Update().
		Where(agent.UID(uid.String()), agent.TopicEQ(topic), agent.HostidEQ(hostid)).
		SetLastOnlineAt(lastOnlineAt).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updateagentlastonlineat: %w", err)
	}
	return nil
}

// UpdateAgentOnlineDuration updates the agent online duration.
func (s *RuntimeAgentStore) UpdateAgentOnlineDuration(ctx context.Context, uid types.Uid, topic, hostid string, offlineTime time.Time) error {
	ag, err := s.client.Agent.Query().
		Where(agent.UID(uid.String()), agent.TopicEQ(topic), agent.HostidEQ(hostid)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updateagentonlineduration query: %w", err)
	}

	onlineDuration := int32(offlineTime.Sub(ag.LastOnlineAt).Seconds())
	_, err = s.client.Agent.Update().
		Where(agent.IDEQ(ag.ID)).
		AddOnlineDuration(onlineDuration).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: updateagentonlineduration: %w", err)
	}
	return nil
}

