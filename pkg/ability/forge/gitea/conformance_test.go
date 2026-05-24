package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	"github.com/flowline-io/flowbot/pkg/ability/forge"
	provider "github.com/flowline-io/flowbot/pkg/providers/gitea"

	giteasdk "code.gitea.io/sdk/gitea"
)

func TestGiteaForgeConformance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{"runs forge conformance test suite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			conformance.RunForgeConformance(t, func(_ *testing.T, cfg conformance.ForgeConfig) forge.Service {
				c := &fakeClient{
					user:           cfgToSDKUser(cfg.User),
					userErr:        cfg.UserErr,
					repo:           cfgToSDKRepo(cfg.Repo),
					repoErr:        cfg.RepoErr,
					issues:         cfgToSDKIssues(cfg.Issues),
					issuesErr:      cfg.IssuesErr,
					diff:           cfgToDiff(cfg.Diff),
					diffErr:        cfg.DiffErr,
					fileContent:    cfg.FileContent,
					fileContentErr: cfg.FileContentErr,
				}
				a, ok := NewWithClient(c).(*Adapter)
				if !ok {
					t.Fatal("unexpected type")
				}
				a.cursorSecret = conformance.CursorSecret
				a.now = conformance.TestTime
				return a
			})
		})
	}
}

func cfgToSDKUser(user *ability.ForgeUser) *giteasdk.User {
	if user == nil {
		return nil
	}
	return &giteasdk.User{
		ID:        user.ID,
		UserName:  user.UserName,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
	}
}

func cfgToSDKRepo(repo *ability.ForgeRepo) *giteasdk.Repository {
	if repo == nil {
		return nil
	}
	return &giteasdk.Repository{
		ID:          repo.ID,
		Name:        repo.Name,
		FullName:    repo.FullName,
		Description: repo.Description,
		Private:     repo.Private,
		HTMLURL:     repo.HTMLURL,
		CloneURL:    repo.CloneURL,
		Owner:       &giteasdk.User{UserName: repo.Owner},
	}
}

func cfgToSDKIssues(issues []*ability.ForgeIssue) []*giteasdk.Issue {
	if issues == nil {
		return nil
	}
	result := make([]*giteasdk.Issue, 0, len(issues))
	for _, issue := range issues {
		state := giteasdk.StateType(issue.State)
		result = append(result, &giteasdk.Issue{
			ID:         issue.ID,
			Index:      issue.Index,
			Title:      issue.Title,
			Body:       issue.Body,
			State:      state,
			Poster:     &giteasdk.User{UserName: issue.Author},
			Repository: &giteasdk.RepositoryMeta{Name: "repo"},
		})
	}
	return result
}

func cfgToDiff(diff *ability.ForgeCommitDiff) *provider.CommitDiff {
	if diff == nil {
		return nil
	}
	return &provider.CommitDiff{
		CommitID:      diff.CommitID,
		CommitMessage: diff.CommitMessage,
		Files:         diff.Files,
		DiffContent:   diff.DiffContent,
	}
}

func TestFakeClientSatisfiesInterface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		assert func(t *testing.T)
	}{
		{
			name: "fake client satisfies client interface",
			assert: func(_ *testing.T) {
				var _ client = (*fakeClient)(nil)
			},
		},
		{
			name: "nil check does not panic",
			assert: func(t *testing.T) {
				var c client
				assert.Nil(t, c)
			},
		},
		{
			name: "NewWithClient accepts fakeClient",
			assert: func(t *testing.T) {
				a := NewWithClient(&fakeClient{})
				require.NotNil(t, a)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t)
		})
	}
}

func TestTestIssueHelper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		index      int64
		title      string
		wantIndex  int64
		wantTitle  string
		wantState  giteasdk.StateType
	}{
		{name: "basic issue", index: 42, title: "Test Issue", wantIndex: 42, wantTitle: "Test Issue", wantState: giteasdk.StateOpen},
		{name: "issue with special chars", index: 1, title: "Fix: panic on nil", wantIndex: 1, wantTitle: "Fix: panic on nil", wantState: giteasdk.StateOpen},
		{name: "large index", index: 99999, title: "Batch cleanup", wantIndex: 99999, wantTitle: "Batch cleanup", wantState: giteasdk.StateOpen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			issue := testIssue(tt.index, tt.title)
			assert.Equal(t, tt.wantIndex, issue.Index)
			assert.Equal(t, tt.wantTitle, issue.Title)
			assert.Equal(t, tt.wantState, issue.State)
		})
	}
}

func TestDecodeTestCursor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		capability     string
		backend        string
		strategy       string
		providerCursor string
		limit          int
	}{
		{name: "page 1", capability: "forge", backend: provider.ID, strategy: "offset", providerCursor: "1", limit: 50},
		{name: "page 5", capability: "forge", backend: provider.ID, strategy: "offset", providerCursor: "5", limit: 20},
		{name: "different limit", capability: "forge", backend: provider.ID, strategy: "offset", providerCursor: "10", limit: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &fakeClient{}
			a, ok := NewWithClient(c).(*Adapter)
			if !ok {
				t.Fatal("unexpected type")
			}
			a.cursorSecret = conformance.CursorSecret
			a.now = conformance.TestTime
			cursor, err := ability.EncodeCursor(a.cursorSecret, ability.CursorPayload{
				Capability:     tt.capability,
				Backend:        tt.backend,
				Strategy:       tt.strategy,
				ProviderCursor: tt.providerCursor,
				Limit:          tt.limit,
			})
			require.NoError(t, err)
			payload := decodeTestCursor(t, a, cursor)
			assert.Equal(t, tt.providerCursor, payload.ProviderCursor)
			assert.Equal(t, tt.limit, payload.Limit)
		})
	}
}
