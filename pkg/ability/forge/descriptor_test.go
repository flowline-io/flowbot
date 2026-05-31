package forge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockForgeService struct{}

func (*mockForgeService) GetUser(_ context.Context) (*ability.ForgeUser, error) { return nil, nil }
func (*mockForgeService) GetRepo(_ context.Context, _, _ string) (*ability.ForgeRepo, error) {
	return nil, nil
}
func (*mockForgeService) ListIssues(_ context.Context, _ string, _ *ListIssuesQuery) (*ability.ListResult[ability.ForgeIssue], error) {
	return nil, nil
}
func (*mockForgeService) GetIssue(_ context.Context, _, _ string, _ int64) (*ability.ForgeIssue, error) {
	return nil, nil
}
func (*mockForgeService) GetCommitDiff(_ context.Context, _, _, _ string) (*ability.ForgeCommitDiff, error) {
	return nil, nil
}
func (*mockForgeService) GetFileContent(_ context.Context, _, _, _, _ string, _, _ int) ([]byte, error) {
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
		{"nil service produces unhealthy descriptor", "gitea", "gitea", nil, false},
		{"non-nil service produces healthy descriptor", "gitea", "gitea", &mockForgeService{}, true},
		{"different backend and app names produce correct descriptor", "gogs", "gogs-instance", &mockForgeService{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor(tt.backend, tt.app, tt.svc)
			assert.Equal(t, hub.CapForge, desc.Type)
			assert.Equal(t, tt.backend, desc.Backend)
			assert.Equal(t, tt.app, desc.App)
			assert.Equal(t, tt.wantHealthy, desc.Healthy)
			assert.Equal(t, "Forge capability", desc.Description)
			assert.Len(t, desc.Operations, 6)
			assert.Len(t, desc.Events, 5)
		})
	}
}

func TestDescriptor_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   string
	}{
		{"has get_user operation", ability.OpForgeGetUser},
		{"has get_repo operation", ability.OpForgeGetRepo},
		{"has list_issues operation", ability.OpForgeListIssues},
		{"has get_issue operation", ability.OpForgeGetIssue},
		{"has get_commit_diff operation", ability.OpForgeGetCommitDiff},
		{"has get_file_content operation", ability.OpForgeGetFileContent},
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
		{"has forge.issue.opened event", "forge.issue.opened"},
		{"has forge.issue.closed event", "forge.issue.closed"},
		{"has forge.issue.reopened event", "forge.issue.reopened"},
		{"has forge.issue.edited event", "forge.issue.edited"},
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
			err := RegisterService("gitea", "app1", tt.svc)
			assert.NoError(t, err)
		})
	}
}
