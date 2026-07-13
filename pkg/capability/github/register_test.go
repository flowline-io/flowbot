package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockGithubService struct{}

func (*mockGithubService) GetUser(_ context.Context) (*capability.ForgeUser, error) { return nil, nil }
func (*mockGithubService) GetUserByLogin(_ context.Context, _ string) (*capability.ForgeUser, error) {
	return nil, nil
}
func (*mockGithubService) GetRepo(_ context.Context, _, _ string) (*capability.ForgeRepo, error) {
	return nil, nil
}
func (*mockGithubService) ListIssues(_ context.Context, _ string, _ *ListIssuesQuery) (*capability.ListResult[capability.ForgeIssue], error) {
	return nil, nil
}
func (*mockGithubService) GetIssue(_ context.Context, _, _ string, _ int64) (*capability.ForgeIssue, error) {
	return nil, nil
}
func (*mockGithubService) GetCommitDiff(_ context.Context, _, _, _ string) (*capability.ForgeCommitDiff, error) {
	return nil, nil
}
func (*mockGithubService) GetFileContent(_ context.Context, _, _, _, _ string, _, _ int) ([]byte, error) {
	return nil, nil
}
func (*mockGithubService) ListNotifications(_ context.Context, _ *PageQuery) (*capability.ListResult[capability.Notification], error) {
	return nil, nil
}
func (*mockGithubService) ListReleases(_ context.Context, _, _ string, _ *PageQuery) (*capability.ListResult[capability.Release], error) {
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
		{name: "valid service", app: "app1", svc: &mockGithubService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockGithubService{}, wantErr: false},
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
	require.NoError(t, Register("github", &mockGithubService{}))
	desc, ok := hub.Default.Get(hub.CapGithub)
	require.True(t, ok)
	assert.Equal(t, hub.CapGithub, desc.Type)
	assert.Equal(t, "github", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 9)
	assert.Len(t, desc.Events, 1)

	tests := []struct {
		name string
		op   string
	}{
		{"has get_user operation", OpGetUser},
		{"has get_user_by_login operation", OpGetUserByLogin},
		{"has get_repo operation", OpGetRepo},
		{"has list_issues operation", OpListIssues},
		{"has get_issue operation", OpGetIssue},
		{"has get_commit_diff operation", OpGetCommitDiff},
		{"has get_file_content operation", OpGetFileContent},
		{"has list_notifications operation", OpListNotifications},
		{"has list_releases operation", OpListReleases},
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
	require.NoError(t, Register("github", &mockGithubService{}))
	desc, ok := hub.Default.Get(hub.CapGithub)
	require.True(t, ok)

	tests := []struct {
		name  string
		event string
	}{
		{"has forge.push event", "forge.push"},
		{"has exactly one event", "forge.push"},
		{"event list not empty", "forge.push"},
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
