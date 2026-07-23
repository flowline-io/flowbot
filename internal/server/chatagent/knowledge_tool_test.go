package chatagent_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type knowledgeTestStore struct {
	store.Adapter
	docs map[string]*gen.AgentKnowledge
}

func (s *knowledgeTestStore) SearchAgentKnowledge(_ context.Context, params store.AgentKnowledgeSearchParams) ([]*gen.AgentKnowledge, error) {
	if strings.TrimSpace(params.Query) == "" && strings.TrimSpace(params.PathPrefix) == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "query or path_prefix is required")
	}
	out := make([]*gen.AgentKnowledge, 0)
	q := strings.ToLower(strings.TrimSpace(params.Query))
	for _, doc := range s.docs {
		if params.PathPrefix != "" && !strings.HasPrefix(doc.Path, params.PathPrefix) {
			continue
		}
		if q != "" {
			hay := strings.ToLower(doc.Path + "\n" + doc.Title + "\n" + doc.Summary + "\n" + doc.Content + "\n" + strings.Join(doc.Tags, "\n"))
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, doc)
	}
	return out, nil
}

func (s *knowledgeTestStore) GetAgentKnowledgeByPath(_ context.Context, path string) (*gen.AgentKnowledge, error) {
	doc, ok := s.docs[path]
	if !ok {
		return nil, types.ErrNotFound
	}
	return doc, nil
}

func TestSearchKnowledgeTool(t *testing.T) {
	orig := store.Database
	t.Cleanup(func() { store.Database = orig })

	tests := []struct {
		name      string
		args      map[string]any
		wantErr   bool
		wantEmpty bool
		wantPath  string
	}{
		{
			name:     "query finds document",
			args:     map[string]any{"query": "postgres"},
			wantPath: "/docs/ops/postgres.md",
		},
		{
			name:     "query finds tag-only document",
			args:     map[string]any{"query": "flowbot"},
			wantPath: "/scripts/run.md",
		},
		{
			name:    "missing query and prefix",
			args:    map[string]any{},
			wantErr: true,
		},
		{
			name:      "no matches",
			args:      map[string]any{"query": "zzzz-missing"},
			wantEmpty: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.Database = &knowledgeTestStore{
				docs: map[string]*gen.AgentKnowledge{
					"/docs/ops/postgres.md": {
						Path:      "/docs/ops/postgres.md",
						Title:     "Postgres Backup",
						Tags:      []string{"ops"},
						Summary:   "Backup guide",
						Content:   "How to backup postgres",
						UpdatedAt: time.Now(),
					},
					"/scripts/run.md": {
						Path:      "/scripts/run.md",
						Title:     "Homelab Data Hub",
						Tags:      []string{"flowbot", "homelab"},
						Summary:   "",
						Content:   "Overview without product name in body",
						UpdatedAt: time.Now(),
					},
				},
			}
			res, err := (chatagent.SearchKnowledgeTool{}).Execute(context.Background(), "call-1", tt.args, nil)
			require.NoError(t, err)
			require.Equal(t, "search_knowledge", res.Name)
			text := knowledgeResultText(res)
			if tt.wantErr {
				assert.True(t, res.IsError)
				return
			}
			assert.False(t, res.IsError)
			if tt.wantEmpty {
				assert.Contains(t, text, "no matches")
				return
			}
			assert.Contains(t, text, tt.wantPath)
		})
	}
}

func TestGetKnowledgeTool(t *testing.T) {
	orig := store.Database
	t.Cleanup(func() { store.Database = orig })

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
		want    string
	}{
		{
			name: "reads by path",
			args: map[string]any{"path": "/docs/ops/postgres.md"},
			want: "Backup body",
		},
		{
			name:    "invalid path",
			args:    map[string]any{"path": "relative.md"},
			wantErr: true,
		},
		{
			name:    "not found",
			args:    map[string]any{"path": "/docs/missing.md"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.Database = &knowledgeTestStore{
				docs: map[string]*gen.AgentKnowledge{
					"/docs/ops/postgres.md": {
						Path:    "/docs/ops/postgres.md",
						Title:   "Postgres",
						Content: "Backup body",
					},
				},
			}
			res, err := (chatagent.GetKnowledgeTool{MaxOutput: 8192}).Execute(context.Background(), "call-2", tt.args, nil)
			require.NoError(t, err)
			if tt.wantErr {
				assert.True(t, res.IsError)
				return
			}
			assert.False(t, res.IsError)
			assert.Contains(t, knowledgeResultText(res), tt.want)
		})
	}
}

func TestActiveToolNamesIncludesKnowledge(t *testing.T) {
	tests := []struct {
		name string
		tool string
	}{
		{name: "search_knowledge", tool: "search_knowledge"},
		{name: "get_knowledge", tool: "get_knowledge"},
		{name: "read_skill still present", tool: "read_skill"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, chatagent.ActiveToolNames(), tt.tool)
		})
	}
}

func knowledgeResultText(result msg.ToolResultMessage) string {
	var out strings.Builder
	for _, part := range result.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = out.WriteString(tp.Text)
		}
	}
	return out.String()
}
