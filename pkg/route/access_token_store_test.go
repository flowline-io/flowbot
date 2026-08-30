package route

import (
	"context"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

type memoryAccessTokenStore struct {
	mu   sync.Mutex
	rows map[string]AccessToken
	next int64
}

func newMemoryAccessTokenStore() *memoryAccessTokenStore {
	return &memoryAccessTokenStore{rows: make(map[string]AccessToken), next: 1}
}

func (m *memoryAccessTokenStore) Get(_ context.Context, flag string) (AccessToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.rows[flag]
	if !ok {
		return AccessToken{}, types.ErrNotFound
	}
	return p, nil
}

func (m *memoryAccessTokenStore) Set(_ context.Context, flag string, params types.KV, expiredAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.rows[flag]
	id := existing.ID
	if !ok {
		id = m.next
		m.next++
	}
	cp := map[string]any{}
	maps.Copy(cp, params)
	m.rows[flag] = AccessToken{ID: id, Flag: flag, Params: cp, ExpiredAt: expiredAt}
	return nil
}

func (m *memoryAccessTokenStore) SetParams(_ context.Context, flag string, params types.KV) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.rows[flag]
	if !ok {
		return types.ErrNotFound
	}
	cp := map[string]any{}
	maps.Copy(cp, params)
	existing.Params = cp
	m.rows[flag] = existing
	return nil
}

func (m *memoryAccessTokenStore) Delete(_ context.Context, flag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, flag)
	return nil
}

func withTestAccessTokenStore(t *testing.T) *memoryAccessTokenStore {
	t.Helper()
	mem := newMemoryAccessTokenStore()
	prev := getAccessTokenStore()
	SetAccessTokenStore(mem)
	t.Cleanup(func() { SetAccessTokenStore(prev) })
	return mem
}
