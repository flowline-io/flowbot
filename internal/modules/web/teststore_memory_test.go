package web

// testStore memory fact and session-summary stubs used by web handler tests.

import (
	"context"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

func memoryFactMapKey(scope, key string) string {
	return scope + "\x00" + key
}

func (s *testStore) UpsertAgentMemoryFact(_ context.Context, fact store.AgentMemoryFactUpsert) (*gen.AgentMemoryFact, error) {
	if s.agentMemoryFacts == nil {
		s.agentMemoryFacts = make(map[string]*gen.AgentMemoryFact)
	}
	k := memoryFactMapKey(fact.Scope, fact.Key)
	now := time.Now().UTC()
	if existing, ok := s.agentMemoryFacts[k]; ok {
		existing.Value = fact.Value
		existing.Pinned = fact.Pinned
		existing.UpdatedAt = now
		cp := *existing
		return &cp, nil
	}
	s.agentMemoryFactSeq++
	row := &gen.AgentMemoryFact{
		ID:        s.agentMemoryFactSeq,
		Scope:     fact.Scope,
		Key:       fact.Key,
		Value:     fact.Value,
		Pinned:    fact.Pinned,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.agentMemoryFacts[k] = row
	cp := *row
	return &cp, nil
}

func (s *testStore) GetAgentMemoryFact(_ context.Context, scope, key string) (*gen.AgentMemoryFact, error) {
	if s.agentMemoryFacts == nil {
		return nil, types.ErrNotFound
	}
	row, ok := s.agentMemoryFacts[memoryFactMapKey(scope, key)]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *row
	return &cp, nil
}

func (s *testStore) ListAgentMemoryFacts(_ context.Context, scope string) ([]*gen.AgentMemoryFact, error) {
	out := make([]*gen.AgentMemoryFact, 0)
	for _, row := range s.agentMemoryFacts {
		if row.Scope == scope {
			cp := *row
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (s *testStore) DeleteAgentMemoryFact(_ context.Context, scope, key string) error {
	if s.agentMemoryFacts == nil {
		return types.ErrNotFound
	}
	k := memoryFactMapKey(scope, key)
	if _, ok := s.agentMemoryFacts[k]; !ok {
		return types.ErrNotFound
	}
	delete(s.agentMemoryFacts, k)
	return nil
}

func (s *testStore) ListInjectableAgentMemoryFacts(ctx context.Context, params store.AgentMemoryInjectableParams) ([]*gen.AgentMemoryFact, error) {
	return s.ListAgentMemoryFacts(ctx, params.Scope)
}

func (*testStore) GetAgentMemoryFactsFingerprint(_ context.Context, _ string) (store.AgentMemoryFactsFingerprint, error) {
	return store.AgentMemoryFactsFingerprint{}, nil
}

func (s *testStore) UpsertAgentSessionSummaryPending(_ context.Context, sessionFlag, scope, title string) (*gen.AgentSessionSummary, error) {
	sessionFlag = strings.TrimSpace(sessionFlag)
	scope = strings.TrimSpace(scope)
	if sessionFlag == "" || scope == "" {
		return nil, types.ErrNotFound
	}
	if s.agentSessionSummaries == nil {
		s.agentSessionSummaries = make(map[string]*gen.AgentSessionSummary)
	}
	now := time.Now().UTC()
	if existing, ok := s.agentSessionSummaries[sessionFlag]; ok {
		existing.Scope = scope
		existing.Title = strings.TrimSpace(title)
		existing.Status = schema.AgentSessionSummaryPending
		existing.Error = ""
		existing.ClaimToken = ""
		existing.ClaimedAt = nil
		existing.UpdatedAt = now
		cp := *existing
		return &cp, nil
	}
	s.agentSessionSummarySeq++
	row := &gen.AgentSessionSummary{
		ID:          s.agentSessionSummarySeq,
		SessionFlag: sessionFlag,
		Scope:       scope,
		Title:       strings.TrimSpace(title),
		Summary:     "",
		Status:      schema.AgentSessionSummaryPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.agentSessionSummaries[sessionFlag] = row
	cp := *row
	return &cp, nil
}

func (s *testStore) ClaimAgentSessionSummaryPending(_ context.Context, claimToken string) (*gen.AgentSessionSummary, error) {
	claimToken = strings.TrimSpace(claimToken)
	if claimToken == "" {
		return nil, types.ErrNotFound
	}
	now := time.Now().UTC()
	for _, row := range s.agentSessionSummaries {
		if row.Status != schema.AgentSessionSummaryPending || row.ClaimToken != "" {
			continue
		}
		row.ClaimToken = claimToken
		row.ClaimedAt = &now
		row.UpdatedAt = now
		cp := *row
		return &cp, nil
	}
	return nil, types.ErrNotFound
}

func (s *testStore) MarkAgentSessionSummaryReady(_ context.Context, sessionFlag, claimToken, title, summary string) error {
	row, ok := s.agentSessionSummaries[strings.TrimSpace(sessionFlag)]
	if !ok {
		return types.ErrNotFound
	}
	if strings.TrimSpace(claimToken) == "" || row.ClaimToken != strings.TrimSpace(claimToken) {
		return types.ErrNotFound
	}
	if row.Status != schema.AgentSessionSummaryPending {
		return types.ErrNotFound
	}
	row.Title = strings.TrimSpace(title)
	row.Summary = summary
	row.Status = schema.AgentSessionSummaryReady
	row.Error = ""
	row.ClaimToken = ""
	row.ClaimedAt = nil
	row.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *testStore) MarkAgentSessionSummaryFailed(_ context.Context, sessionFlag, claimToken, errMsg string) error {
	row, ok := s.agentSessionSummaries[strings.TrimSpace(sessionFlag)]
	if !ok {
		return types.ErrNotFound
	}
	if strings.TrimSpace(claimToken) == "" || row.ClaimToken != strings.TrimSpace(claimToken) {
		return types.ErrNotFound
	}
	if row.Status != schema.AgentSessionSummaryPending {
		return types.ErrNotFound
	}
	row.Status = schema.AgentSessionSummaryFailed
	row.Error = errMsg
	row.ClaimToken = ""
	row.ClaimedAt = nil
	row.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *testStore) GetAgentSessionSummaryBySession(_ context.Context, sessionFlag string) (*gen.AgentSessionSummary, error) {
	row, ok := s.agentSessionSummaries[strings.TrimSpace(sessionFlag)]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *row
	return &cp, nil
}

func (s *testStore) SearchAgentSessionSummaries(_ context.Context, params store.AgentSessionSummarySearchParams) ([]*gen.AgentSessionSummary, error) {
	q := strings.ToLower(strings.TrimSpace(params.Query))
	if q == "" {
		return nil, nil
	}
	out := make([]*gen.AgentSessionSummary, 0)
	for _, row := range s.agentSessionSummaries {
		if row.Status != schema.AgentSessionSummaryReady {
			continue
		}
		if params.Scope != "" && row.Scope != params.Scope {
			continue
		}
		if !strings.Contains(strings.ToLower(row.Title), q) && !strings.Contains(strings.ToLower(row.Summary), q) {
			continue
		}
		cp := *row
		out = append(out, &cp)
	}
	return out, nil
}

func (s *testStore) ListAgentSessionSummaries(_ context.Context, filter store.AgentSessionSummaryListFilter) ([]*gen.AgentSessionSummary, error) {
	out := make([]*gen.AgentSessionSummary, 0)
	q := strings.ToLower(strings.TrimSpace(filter.Q))
	for _, row := range s.agentSessionSummaries {
		if filter.Scope != "" && row.Scope != filter.Scope {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(row.Title), q) && !strings.Contains(strings.ToLower(row.Summary), q) {
			continue
		}
		cp := *row
		out = append(out, &cp)
	}
	return out, nil
}

func (s *testStore) RequeueStaleAgentSessionSummaryPending(_ context.Context, olderThan time.Duration) (int, error) {
	if olderThan <= 0 {
		olderThan = 10 * time.Minute
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	n := 0
	for _, row := range s.agentSessionSummaries {
		if row.Status != schema.AgentSessionSummaryPending || row.ClaimToken == "" || row.ClaimedAt == nil {
			continue
		}
		if row.ClaimedAt.Before(cutoff) {
			row.ClaimToken = ""
			row.ClaimedAt = nil
			row.UpdatedAt = time.Now().UTC()
			n++
		}
	}
	return n, nil
}
