package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockGithubService struct{}

func (*mockGithubService) GetUser(_ context.Context) (*ability.ForgeUser, error) { return nil, nil }
func (*mockGithubService) GetUserByLogin(_ context.Context, _ string) (*ability.ForgeUser, error) {
	return nil, nil
}
func (*mockGithubService) GetRepo(_ context.Context, _, _ string) (*ability.ForgeRepo, error) {
	return nil, nil
}
func (*mockGithubService) ListIssues(_ context.Context, _ string, _ *ListIssuesQuery) (*ability.ListResult[ability.ForgeIssue], error) {
	return nil, nil
}
func (*mockGithubService) GetIssue(_ context.Context, _, _ string, _ int64) (*ability.ForgeIssue, error) {
	return nil, nil
}
func (*mockGithubService) GetCommitDiff(_ context.Context, _, _, _ string) (*ability.ForgeCommitDiff, error) {
	return nil, nil
}
func (*mockGithubService) GetFileContent(_ context.Context, _, _, _, _ string, _, _ int) ([]byte, error) {
	return nil, nil
}
func (*mockGithubService) ListNotifications(_ context.Context, _ *PageQuery) (*ability.ListResult[ability.Notification], error) {
	return nil, nil
}
func (*mockGithubService) ListReleases(_ context.Context, _, _ string, _ *PageQuery) (*ability.ListResult[ability.Release], error) {
	return nil, nil
}

func TestDescriptor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		backend     string
		app         string
		svc         Service
		wantHealthy bool
	}{
		{"nil service produces unhealthy descriptor", "github", "github", nil, false},
		{"non-nil service produces healthy descriptor", "github", "github", &mockGithubService{}, true},
		{"different backend and app names produce correct descriptor", "github-enterprise", "ghe-instance", &mockGithubService{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor(tt.backend, tt.app, tt.svc)
			assert.Equal(t, hub.CapGithub, desc.Type)
			assert.Equal(t, tt.backend, desc.Backend)
			assert.Equal(t, tt.app, desc.App)
			assert.Equal(t, tt.wantHealthy, desc.Healthy)
			assert.Equal(t, "GitHub capability", desc.Description)
			assert.Len(t, desc.Operations, 9)
			assert.Len(t, desc.Events, 1)
		})
	}
}

func TestDescriptor_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   string
	}{
		{"has get_user operation", ability.OpGithubGetUser},
		{"has get_user_by_login operation", ability.OpGithubGetUserByLogin},
		{"has get_repo operation", ability.OpGithubGetRepo},
		{"has list_issues operation", ability.OpGithubListIssues},
		{"has get_issue operation", ability.OpGithubGetIssue},
		{"has get_commit_diff operation", ability.OpGithubGetCommitDiff},
		{"has get_file_content operation", ability.OpGithubGetFileContent},
		{"has list_notifications operation", ability.OpGithubListNotifications},
		{"has list_releases operation", ability.OpGithubListReleases},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor("g", "g", nil)
			opNames := make([]string, len(desc.Operations))
			for i, op := range desc.Operations {
				opNames[i] = op.Name
			}
			assert.Contains(t, opNames, tt.op)
		})
	}
}

func TestDescriptor_Events(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		event string
	}{
		{"has forge.push event", "forge.push"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor("g", "g", nil)
			eventNames := make([]string, len(desc.Events))
			for i, ev := range desc.Events {
				eventNames[i] = ev.Name
			}
			assert.Contains(t, eventNames, tt.event)
		})
	}
}

func TestRegisterService_NilService(t *testing.T) {
	tests := []struct {
		name string
		svc  Service
	}{
		{name: "nil service returns nil", svc: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterService("github", "app1", tt.svc)
			assert.NoError(t, err)
		})
	}
}
