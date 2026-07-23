package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func (s *testStore) ListAgentKnowledge(_ context.Context, filter store.AgentKnowledgeListFilter) ([]*gen.AgentKnowledge, error) {
	if s.agentKnowledgeErr != nil {
		return nil, s.agentKnowledgeErr
	}
	q := strings.ToLower(strings.TrimSpace(filter.Q))
	rows := make([]*gen.AgentKnowledge, 0, len(s.agentKnowledge))
	for _, doc := range s.agentKnowledge {
		if q != "" {
			if !strings.Contains(strings.ToLower(doc.Path), q) && !strings.Contains(strings.ToLower(doc.Title), q) {
				continue
			}
		}
		rows = append(rows, doc)
	}
	return rows, nil
}

func (*testStore) SearchAgentKnowledge(_ context.Context, _ store.AgentKnowledgeSearchParams) ([]*gen.AgentKnowledge, error) {
	return nil, types.Errorf(types.ErrUnavailable, "not used in web tests")
}

func (s *testStore) GetAgentKnowledgeByPath(_ context.Context, path string) (*gen.AgentKnowledge, error) {
	for _, doc := range s.agentKnowledge {
		if doc.Path == path {
			return doc, nil
		}
	}
	return nil, types.ErrNotFound
}

func (s *testStore) GetAgentKnowledgeByID(_ context.Context, id int64) (*gen.AgentKnowledge, error) {
	doc, ok := s.agentKnowledge[id]
	if !ok {
		return nil, types.ErrNotFound
	}
	return doc, nil
}

func (s *testStore) CreateAgentKnowledge(_ context.Context, doc *gen.AgentKnowledge) error {
	if s.agentKnowledge == nil {
		s.agentKnowledge = make(map[int64]*gen.AgentKnowledge)
	}
	for _, existing := range s.agentKnowledge {
		if existing.Path == doc.Path {
			return types.Errorf(types.ErrInvalidArgument, "duplicate path")
		}
	}
	s.agentKnowledgeSeq++
	doc.ID = s.agentKnowledgeSeq
	if doc.Tags == nil {
		doc.Tags = []string{}
	}
	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	if doc.UpdatedAt.IsZero() {
		doc.UpdatedAt = now
	}
	cp := *doc
	s.agentKnowledge[doc.ID] = &cp
	return nil
}

func (s *testStore) UpdateAgentKnowledge(_ context.Context, doc *gen.AgentKnowledge) error {
	if s.agentKnowledge == nil {
		return types.ErrNotFound
	}
	if _, ok := s.agentKnowledge[doc.ID]; !ok {
		return types.ErrNotFound
	}
	for id, existing := range s.agentKnowledge {
		if id != doc.ID && existing.Path == doc.Path {
			return types.Errorf(types.ErrInvalidArgument, "duplicate path")
		}
	}
	doc.UpdatedAt = time.Now().UTC()
	cp := *doc
	s.agentKnowledge[doc.ID] = &cp
	return nil
}

func (s *testStore) DeleteAgentKnowledge(_ context.Context, id int64) error {
	if s.agentKnowledge == nil {
		return types.ErrNotFound
	}
	if _, ok := s.agentKnowledge[id]; !ok {
		return types.ErrNotFound
	}
	delete(s.agentKnowledge, id)
	return nil
}

func TestValidateAgentKnowledgeForm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		item    model.AgentKnowledge
		wantKey string
	}{
		{
			name: "valid document",
			item: model.AgentKnowledge{Path: "/docs/api.md", Title: "API", Content: "body"},
		},
		{
			name:    "invalid path",
			item:    model.AgentKnowledge{Path: "docs/api.md", Title: "API", Content: "body"},
			wantKey: "path",
		},
		{
			name:    "missing title",
			item:    model.AgentKnowledge{Path: "/docs/api.md", Content: "body"},
			wantKey: "title",
		},
		{
			name:    "missing content",
			item:    model.AgentKnowledge{Path: "/docs/api.md", Title: "API"},
			wantKey: "content",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := validateAgentKnowledgeForm(tt.item)
			if tt.wantKey == "" {
				require.Empty(t, errs)
				return
			}
			_, ok := errs[tt.wantKey]
			assert.True(t, ok, "want error for %q, got %v", tt.wantKey, errs)
		})
	}
}

func TestAgentKnowledgePageUnauthenticated(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "page redirects to login", method: http.MethodGet, path: "/service/web/agent-knowledge"},
		{name: "list redirects to login", method: http.MethodGet, path: "/service/web/agent-knowledge/list"},
		{name: "create redirects to login", method: http.MethodPost, path: "/service/web/agent-knowledge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
		})
	}
}

func TestAgentKnowledgeCreateAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		form       map[string]string
		wantStatus int
		wantBody   string
		wantPath   string
	}{
		{
			name: "creates document",
			form: map[string]string{
				"path":    "/docs/ops/backup.md",
				"title":   "Backup",
				"tags":    "ops, db",
				"summary": "how to backup",
				"content": "# Backup\n\nsteps",
			},
			wantStatus: http.StatusOK,
			wantBody:   "/docs/ops/backup.md",
			wantPath:   "/docs/ops/backup.md",
		},
		{
			name: "rejects invalid path",
			form: map[string]string{
				"path":    "relative.md",
				"title":   "Bad",
				"content": "body",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "path must start with /",
		},
		{
			name: "rejects duplicate path",
			form: map[string]string{
				"path":    "/docs/existing.md",
				"title":   "Dup",
				"content": "body",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "path already exists",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentKnowledge: map[int64]*gen.AgentKnowledge{
					1: {ID: 1, Path: "/docs/existing.md", Title: "Existing", Content: "x", Tags: []string{}},
				},
				agentKnowledgeSeq: 1,
			}
			app := setupAuthenticatedApp(t, ts)
			body := buildFormBody(tt.form)
			req := httptest.NewRequest(http.MethodPost, "/service/web/agent-knowledge", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			respBody, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(respBody), tt.wantBody)
			if tt.wantPath != "" {
				found := false
				for _, doc := range ts.agentKnowledge {
					if doc.Path == tt.wantPath {
						found = true
						assert.Equal(t, []string{"ops", "db"}, doc.Tags)
					}
				}
				assert.True(t, found)
			}
		})
	}
}

func TestAgentKnowledgeUpdateAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		id         int64
		form       map[string]string
		wantStatus int
		wantBody   string
	}{
		{
			name: "updates title",
			id:   1,
			form: map[string]string{
				"path":    "/docs/ops/backup.md",
				"title":   "Updated Backup",
				"content": "new body",
			},
			wantStatus: http.StatusOK,
			wantBody:   "Updated Backup",
		},
		{
			name: "missing document",
			id:   99,
			form: map[string]string{
				"path":    "/docs/missing.md",
				"title":   "Missing",
				"content": "body",
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Knowledge document not found",
		},
		{
			name: "invalid path on update",
			id:   1,
			form: map[string]string{
				"path":    "bad.md",
				"title":   "Bad",
				"content": "body",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "path must start with /",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentKnowledge: map[int64]*gen.AgentKnowledge{
					1: {ID: 1, Path: "/docs/ops/backup.md", Title: "Backup", Content: "old", Tags: []string{}},
				},
				agentKnowledgeSeq: 1,
			}
			app := setupAuthenticatedApp(t, ts)
			body := buildFormBody(tt.form)
			req := httptest.NewRequest(http.MethodPut, "/service/web/agent-knowledge/"+strconv.FormatInt(tt.id, 10), strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			respBody, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(respBody), tt.wantBody)
		})
	}
}

func TestAgentKnowledgeDeleteAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantEmpty  bool
	}{
		{name: "deletes last document", id: "1", wantStatus: http.StatusOK, wantEmpty: true},
		{name: "missing document returns toast no content", id: "99", wantStatus: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentKnowledge: map[int64]*gen.AgentKnowledge{
					1: {ID: 1, Path: "/docs/a.md", Title: "A", Content: "body", Tags: []string{}},
				},
				agentKnowledgeSeq: 1,
			}
			app := setupAuthenticatedApp(t, ts)
			req := httptest.NewRequest(http.MethodDelete, "/service/web/agent-knowledge/"+tt.id, http.NoBody)
			req.Header.Set("Cookie", "accessToken=test-token")
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantEmpty {
				assert.Empty(t, ts.agentKnowledge)
			}
		})
	}
}

func TestAgentKnowledgeListFilterAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		q          string
		wantBody   string
		wantAbsent string
	}{
		{name: "filters by title", q: "Alpha", wantBody: "/docs/alpha.md", wantAbsent: "/docs/beta.md"},
		{name: "empty filter lists all", q: "", wantBody: "/docs/alpha.md"},
		{name: "no match shows empty state", q: "zzzz", wantBody: "No knowledge documents yet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentKnowledge: map[int64]*gen.AgentKnowledge{
					1: {ID: 1, Path: "/docs/alpha.md", Title: "Alpha", Content: "a", Tags: []string{}},
					2: {ID: 2, Path: "/docs/beta.md", Title: "Beta", Content: "b", Tags: []string{}},
				},
			}
			app := setupAuthenticatedApp(t, ts)
			req := httptest.NewRequest(http.MethodGet, "/service/web/agent-knowledge/list?q="+tt.q, http.NoBody)
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			respBody, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(respBody), tt.wantBody)
			if tt.wantAbsent != "" {
				assert.NotContains(t, string(respBody), tt.wantAbsent)
			}
		})
	}
}
