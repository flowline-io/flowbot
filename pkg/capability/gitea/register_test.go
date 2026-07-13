package gitea

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockForgeService struct{}

func (*mockForgeService) GetUser(_ context.Context) (*capability.ForgeUser, error) { return nil, nil }
func (*mockForgeService) GetRepo(_ context.Context, _, _ string) (*capability.ForgeRepo, error) {
	return nil, nil
}
func (*mockForgeService) ListIssues(_ context.Context, _ string, _ *ListIssuesQuery) (*capability.ListResult[capability.ForgeIssue], error) {
	return nil, nil
}
func (*mockForgeService) GetIssue(_ context.Context, _, _ string, _ int64) (*capability.ForgeIssue, error) {
	return nil, nil
}
func (*mockForgeService) GetCommitDiff(_ context.Context, _, _, _ string) (*capability.ForgeCommitDiff, error) {
	return nil, nil
}
func (*mockForgeService) GetFileContent(_ context.Context, _, _, _, _ string, _, _ int) ([]byte, error) {
	return nil, nil
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: &mockForgeService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockForgeService{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Register(tt.app, tt.svc)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegister_Operations(t *testing.T) {
	require.NoError(t, Register("gitea", &mockForgeService{}))
	desc, ok := hub.Default.Get(hub.CapGitea)
	require.True(t, ok)
	assert.Equal(t, hub.CapGitea, desc.Type)
	assert.Equal(t, "gitea", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 6)
	assert.Len(t, desc.Events, 5)

	tests := []struct {
		name string
		op   string
	}{
		{"has get_user operation", OpGetUser},
		{"has get_repo operation", OpGetRepo},
		{"has list_issues operation", OpListIssues},
		{"has get_issue operation", OpGetIssue},
		{"has get_commit_diff operation", OpGetCommitDiff},
		{"has get_file_content operation", OpGetFileContent},
	}
	opNames := make([]string, len(desc.Operations))
	for i, op := range desc.Operations {
		opNames[i] = op.Name
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, opNames, tt.op)
		})
	}
}

func TestRegister_Events(t *testing.T) {
	require.NoError(t, Register("gitea", &mockForgeService{}))
	desc, ok := hub.Default.Get(hub.CapGitea)
	require.True(t, ok)

	tests := []struct {
		name  string
		event string
	}{
		{"has forge.issue.opened event", "forge.issue.opened"},
		{"has forge.issue.closed event", "forge.issue.closed"},
		{"has forge.issue.reopened event", "forge.issue.reopened"},
		{"has forge.issue.edited event", "forge.issue.edited"},
		{"has forge.push event", "forge.push"},
	}
	eventNames := make([]string, len(desc.Events))
	for i, ev := range desc.Events {
		eventNames[i] = ev.Name
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, eventNames, tt.event)
		})
	}
}
