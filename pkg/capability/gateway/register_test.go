package gateway_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	capgw "github.com/flowline-io/flowbot/pkg/capability/gateway"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

type memStore struct {
	mu      sync.Mutex
	jobs    map[string]*types.GatewayJob
	fresh   bool
	created int
}

func newMemStore(fresh bool) *memStore {
	return &memStore{jobs: map[string]*types.GatewayJob{}, fresh: fresh}
}

func (m *memStore) Create(_ context.Context, in types.GatewayCreateJob) (*types.GatewayJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.created++
	id := types.Id()
	j := &types.GatewayJob{
		JobID: id, UID: in.UID, CLI: in.CLI, Prompt: in.Prompt, Cwd: in.Cwd,
		Status: types.GatewayJobPending, CreatedAt: time.Now(),
	}
	m.jobs[id] = j
	go func() {
		time.Sleep(20 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		if cur, ok := m.jobs[id]; ok && cur.Status == types.GatewayJobPending {
			cur.Status = types.GatewayJobSucceeded
			cur.Output = "done"
		}
	}()
	return j, nil
}

func (m *memStore) Get(_ context.Context, jobID string) (*types.GatewayJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j := m.jobs[jobID]
	if j == nil {
		return nil, nil
	}
	cp := *j
	return &cp, nil
}

func (m *memStore) Cancel(_ context.Context, jobID string) (*types.GatewayJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j := m.jobs[jobID]
	if j == nil {
		return nil, types.ErrNotFound
	}
	j.Status = types.GatewayJobCanceled
	cp := *j
	return &cp, nil
}

func (m *memStore) HasFreshWorker(context.Context, time.Duration) (bool, error) {
	return m.fresh, nil
}

func (*memStore) ReclaimExpired(context.Context) error {
	return nil
}

func TestRunFailFastWithoutWorker(t *testing.T) {
	hub.Default.Unregister(hub.CapGateway)
	t.Cleanup(func() { hub.Default.Unregister(hub.CapGateway) })

	config.App.Gateway = config.GatewayConfig{Enabled: true, RunTimeout: time.Minute, WorkerStaleAfter: time.Minute}
	capgw.SetJobStore(newMemStore(false))
	require.NoError(t, capgw.Register())

	_, err := capability.Invoke(context.Background(), hub.CapGateway, capgw.OpRun, map[string]any{
		"prompt": "x",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrUnavailable)
}

func TestRunWaitsForTerminal(t *testing.T) {
	hub.Default.Unregister(hub.CapGateway)
	t.Cleanup(func() { hub.Default.Unregister(hub.CapGateway) })

	config.App.Gateway = config.GatewayConfig{Enabled: true, RunTimeout: time.Minute, WorkerStaleAfter: time.Minute}
	capgw.SetJobStore(newMemStore(true))
	require.NoError(t, capgw.Register())

	res, err := capability.Invoke(context.Background(), hub.CapGateway, capgw.OpRun, map[string]any{
		"prompt": "hello",
	})
	require.NoError(t, err)
	job, ok := res.Data.(*types.GatewayJob)
	require.True(t, ok)
	assert.Equal(t, types.GatewayJobSucceeded, job.Status)
	assert.Equal(t, "done", job.Output)
}

func TestRunRejectsOpenCode(t *testing.T) {
	hub.Default.Unregister(hub.CapGateway)
	t.Cleanup(func() { hub.Default.Unregister(hub.CapGateway) })

	config.App.Gateway = config.GatewayConfig{Enabled: true}
	capgw.SetJobStore(newMemStore(true))
	require.NoError(t, capgw.Register())

	_, err := capability.Invoke(context.Background(), hub.CapGateway, capgw.OpRun, map[string]any{
		"prompt": "x", "cli": "opencode",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidArgument)
}
