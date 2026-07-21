package workflow

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
)

type mockCatalog struct {
	meta    map[string]*types.WorkflowMetadata
	defs    []*gen.Workflow
	getErr  error
	listErr error
}

func (m *mockCatalog) GetMetadata(_ context.Context, name string) (*types.WorkflowMetadata, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	meta, ok := m.meta[name]
	if !ok {
		return nil, types.Errorf(types.ErrNotFound, "workflow %s", name)
	}
	return meta, nil
}

func (*mockCatalog) ApplyDefinition(_ context.Context, _ *types.WorkflowMetadata) (*gen.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCatalog) ListDefinitions(_ context.Context) ([]*gen.Workflow, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.defs, nil
}

func (*mockCatalog) DeleteDefinitionByName(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

func (*mockCatalog) ListRunsByName(_ context.Context, _ string) ([]*gen.WorkflowRun, error) {
	return nil, nil
}

type mockRunStore struct {
	mu      sync.Mutex
	created []*gen.WorkflowRun
	nextID  int64
}

func (m *mockRunStore) CreateRun(_ context.Context, workflowID int64, workflowName, workflowFile, triggerType string, _, inputParams map[string]any) (*gen.WorkflowRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := m.nextID
	var wfID *int64
	if workflowID != 0 {
		wfID = &workflowID
	}
	run := &gen.WorkflowRun{
		ID:           id,
		WorkflowID:   wfID,
		WorkflowName: workflowName,
		WorkflowFile: workflowFile,
		TriggerType:  triggerType,
		InputParams:  inputParams,
	}
	m.created = append(m.created, run)
	return run, nil
}

func (*mockRunStore) UpdateRunStatus(context.Context, int64, int, string) error { return nil }
func (*mockRunStore) CreateStepRun(context.Context, int64, string, string, string, string, map[string]any, int) (*gen.WorkflowStepRun, error) {
	return &gen.WorkflowStepRun{ID: 1}, nil
}
func (*mockRunStore) UpdateStepRun(context.Context, int64, int, map[string]any, string, int) error {
	return nil
}
func (*mockRunStore) SaveCheckpoint(context.Context, int64, any) error { return nil }
func (*mockRunStore) GetIncompleteRuns(context.Context) ([]*gen.WorkflowRun, error) {
	return nil, nil
}
func (*mockRunStore) GetCheckpoint(context.Context, int64, any) error { return nil }
func (*mockRunStore) GetRun(context.Context, int64) (*gen.WorkflowRun, error) {
	return nil, errors.New("not found")
}
func (*mockRunStore) UpdateRunHeartbeat(context.Context, int64) error { return nil }

func sampleMeta(name string) *types.WorkflowMetadata {
	return &types.WorkflowMetadata{
		Name:     name,
		Enabled:  true,
		Pipeline: []string{"step1"},
		Inputs: []types.WorkflowInputDef{
			{Name: "url", Type: types.WorkflowInputTypeString, Required: true},
		},
		Tasks: []types.WorkflowTask{
			{ID: "step1", Action: "mapper:", Params: types.KV{"echo": "{{input.url}}"}},
		},
	}
}

func TestService_StartRunAsync_Validation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		wfName      string
		input       types.KV
		catalog     *mockCatalog
		runs        WorkflowRunStore
		wantErr     bool
		errContain  string
		wantRunID   bool
		waitExecute bool
	}{
		{
			name:   "happy path creates run id",
			wfName: "ok-wf",
			input:  types.KV{"url": "https://example.com"},
			catalog: &mockCatalog{
				meta: map[string]*types.WorkflowMetadata{"ok-wf": sampleMeta("ok-wf")},
				defs: []*gen.Workflow{{ID: 42, Name: "ok-wf", Enabled: true}},
			},
			runs:        &mockRunStore{},
			wantRunID:   true,
			waitExecute: true,
		},
		{
			name:       "empty name rejected",
			wfName:     "  ",
			input:      types.KV{},
			catalog:    &mockCatalog{meta: map[string]*types.WorkflowMetadata{}},
			runs:       &mockRunStore{},
			wantErr:    true,
			errContain: "workflow name is required",
		},
		{
			name:   "missing required input rejected",
			wfName: "need-url",
			input:  types.KV{},
			catalog: &mockCatalog{
				meta: map[string]*types.WorkflowMetadata{"need-url": sampleMeta("need-url")},
				defs: []*gen.Workflow{{ID: 1, Name: "need-url", Enabled: true}},
			},
			runs:       &mockRunStore{},
			wantErr:    true,
			errContain: "input validation failed",
		},
		{
			name:   "wrong input type rejected",
			wfName: "bad-type",
			input:  types.KV{"url": 123},
			catalog: &mockCatalog{
				meta: map[string]*types.WorkflowMetadata{"bad-type": sampleMeta("bad-type")},
				defs: []*gen.Workflow{{ID: 2, Name: "bad-type", Enabled: true}},
			},
			runs:       &mockRunStore{},
			wantErr:    true,
			errContain: "input validation failed",
		},
		{
			name:   "workflow not found",
			wfName: "missing",
			input:  types.KV{"url": "x"},
			catalog: &mockCatalog{
				meta: map[string]*types.WorkflowMetadata{},
				defs: nil,
			},
			runs:       &mockRunStore{},
			wantErr:    true,
			errContain: "workflow missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewService(tt.catalog, tt.runs, nil, nil)
			runID, err := svc.StartRunAsync(context.Background(), tt.wfName, "manual", tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Equal(t, int64(0), runID)
				return
			}
			require.NoError(t, err)
			if tt.wantRunID {
				assert.Positive(t, runID)
			}
			if tt.waitExecute {
				time.Sleep(50 * time.Millisecond)
			}
		})
	}
}

func TestWebhookConfigFromRule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		rule       types.KV
		wantPath   string
		wantMethod string
		wantErr    bool
		errContain string
	}{
		{
			name: "token auth defaults headers and method",
			rule: types.KV{
				"path": "hooks/wf",
				"auth": map[string]any{"token": "secret"},
			},
			wantPath:   "hooks/wf",
			wantMethod: "POST",
		},
		{
			name: "missing auth rejected",
			rule: types.KV{
				"path": "hooks/wf",
			},
			wantErr:    true,
			errContain: "auth.token",
		},
		{
			name: "empty path rejected",
			rule: types.KV{
				"auth": map[string]any{"token": "t"},
			},
			wantErr:    true,
			errContain: "path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := webhookConfigFromRule("wf", tt.rule)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, cfg.Path)
			assert.Equal(t, tt.wantMethod, cfg.Method)
			assert.Equal(t, "X-Webhook-Token", cfg.Auth.TokenHeader)
			assert.Equal(t, "secret", cfg.Auth.Token)
			assert.Equal(t, "workflow.webhook.wf", cfg.EventType)
		})
	}
}
