package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func (s *testStore) ListAgentSubagents(_ context.Context, enabledOnly bool) ([]*gen.AgentSubagent, error) {
	if s.agentSubagentsErr != nil {
		return nil, s.agentSubagentsErr
	}
	rows := make([]*gen.AgentSubagent, 0, len(s.agentSubagents))
	for _, subagent := range s.agentSubagents {
		if enabledOnly && !subagent.Enabled {
			continue
		}
		rows = append(rows, subagent)
	}
	return rows, nil
}

func (*testStore) GetAgentSubagentsMaxUpdatedAt(_ context.Context) (time.Time, error) {
	return time.Time{}, nil
}

func (s *testStore) GetAgentSubagentByFlag(_ context.Context, flag string) (*gen.AgentSubagent, error) {
	subagent, ok := s.agentSubagents[flag]
	if !ok {
		return nil, types.ErrNotFound
	}
	return subagent, nil
}

func (s *testStore) CreateAgentSubagent(_ context.Context, subagent *gen.AgentSubagent) error {
	if s.createAgentSubagentFn != nil {
		return s.createAgentSubagentFn(subagent)
	}
	if s.agentSubagents == nil {
		s.agentSubagents = make(map[string]*gen.AgentSubagent)
	}
	if _, exists := s.agentSubagents[subagent.Flag]; exists {
		return types.Errorf(types.ErrInvalidArgument, "agent_subagents_flag_key")
	}
	for _, existing := range s.agentSubagents {
		if existing.Name == subagent.Name {
			return types.Errorf(types.ErrInvalidArgument, "agent_subagents_name_key")
		}
	}
	s.agentSubagents[subagent.Flag] = subagent
	return nil
}

func (s *testStore) UpdateAgentSubagent(_ context.Context, subagent *gen.AgentSubagent) error {
	if s.updateAgentSubagentFn != nil {
		return s.updateAgentSubagentFn(subagent)
	}
	if s.agentSubagents == nil {
		return types.ErrNotFound
	}
	if _, ok := s.agentSubagents[subagent.Flag]; !ok {
		return types.ErrNotFound
	}
	for flag, existing := range s.agentSubagents {
		if flag != subagent.Flag && existing.Name == subagent.Name {
			return types.Errorf(types.ErrInvalidArgument, "agent_subagents_name_key")
		}
	}
	subagent.UpdatedAt = time.Now().UTC()
	s.agentSubagents[subagent.Flag] = subagent
	return nil
}

func (s *testStore) DeleteAgentSubagent(_ context.Context, flag string) error {
	if s.deleteAgentSubagentFn != nil {
		return s.deleteAgentSubagentFn(flag)
	}
	if s.agentSubagents == nil {
		return types.ErrNotFound
	}
	if _, ok := s.agentSubagents[flag]; !ok {
		return types.ErrNotFound
	}
	delete(s.agentSubagents, flag)
	return nil
}

func (s *testStore) CreateAgentSubagentTask(_ context.Context, task *gen.AgentSubagentTask) error {
	if s.createAgentSubagentTaskFn != nil {
		return s.createAgentSubagentTaskFn(task)
	}
	if s.agentSubagentTasks == nil {
		s.agentSubagentTasks = make(map[int64]*gen.AgentSubagentTask)
	}
	if task.ID == 0 {
		task.ID = int64(len(s.agentSubagentTasks) + 1)
	}
	s.agentSubagentTasks[task.ID] = task
	return nil
}

func (s *testStore) UpdateAgentSubagentTask(_ context.Context, task *gen.AgentSubagentTask) error {
	if s.updateAgentSubagentTaskFn != nil {
		return s.updateAgentSubagentTaskFn(task)
	}
	if s.agentSubagentTasks == nil {
		return types.ErrNotFound
	}
	if _, ok := s.agentSubagentTasks[task.ID]; !ok {
		return types.ErrNotFound
	}
	task.UpdatedAt = time.Now().UTC()
	s.agentSubagentTasks[task.ID] = task
	return nil
}

func (s *testStore) ListAgentSubagentTasks(_ context.Context, sessionID string, limit int) ([]*gen.AgentSubagentTask, error) {
	if s.agentSubagentTasksErr != nil {
		return nil, s.agentSubagentTasksErr
	}
	rows := make([]*gen.AgentSubagentTask, 0, len(s.agentSubagentTasks))
	for _, task := range s.agentSubagentTasks {
		if sessionID != "" && task.SessionID != sessionID {
			continue
		}
		rows = append(rows, task)
	}
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (s *testStore) GetAgentSubagentTask(_ context.Context, id int64) (*gen.AgentSubagentTask, error) {
	if s.agentSubagentTasks == nil {
		return nil, types.ErrNotFound
	}
	task, ok := s.agentSubagentTasks[id]
	if !ok {
		return nil, types.ErrNotFound
	}
	return task, nil
}

func TestValidateAgentSubagentForm(t *testing.T) {
	tests := []struct {
		name    string
		item    model.AgentSubagent
		isNew   bool
		wantKey string
	}{
		{name: "empty flag rejected on create", item: model.AgentSubagent{Name: "demo", Description: "d", SystemPrompt: "p"}, isNew: true, wantKey: "flag"},
		{name: "invalid slug rejected", item: model.AgentSubagent{Flag: "Bad Flag", Name: "demo", Description: "d", SystemPrompt: "p"}, isNew: true, wantKey: "flag"},
		{name: "empty name rejected", item: model.AgentSubagent{Flag: "demo", Description: "d", SystemPrompt: "p"}, isNew: true, wantKey: "name"},
		{name: "empty description rejected", item: model.AgentSubagent{Flag: "demo", Name: "demo", SystemPrompt: "p"}, isNew: true, wantKey: "description"},
		{name: "empty system prompt rejected", item: model.AgentSubagent{Flag: "demo", Name: "demo", Description: "d"}, isNew: true, wantKey: "system_prompt"},
		{name: "empty tools rejected", item: model.AgentSubagent{Flag: "demo", Name: "demo", Description: "d", SystemPrompt: "p"}, isNew: true, wantKey: "tools"},
		{name: "invalid tool rejected", item: model.AgentSubagent{Flag: "demo", Name: "demo", Description: "d", SystemPrompt: "p", Tools: []string{"evil_tool"}}, isNew: true, wantKey: "tools"},
		{name: "valid update passes without flag", item: model.AgentSubagent{Name: "demo", Description: "d", SystemPrompt: "body", Tools: []string{"read_file"}}, isNew: false, wantKey: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateAgentSubagentForm(tt.item, tt.isNew)
			if tt.wantKey == "" {
				require.Empty(t, errs)
				return
			}
			_, ok := errs[tt.wantKey]
			assert.True(t, ok, "want error for %q, got %v", tt.wantKey, errs)
		})
	}
}

func TestParseAgentSubagentMultiValues(t *testing.T) {
	tests := []struct {
		name string
		raw  [][]byte
		want []string
	}{
		{name: "multiple values", raw: [][]byte{[]byte("read_file"), []byte("run_terminal")}, want: []string{"read_file", "run_terminal"}},
		{name: "deduplicates and trims", raw: [][]byte{[]byte("read_file"), []byte(" read_file "), []byte("web_search")}, want: []string{"read_file", "web_search"}},
		{name: "empty yields empty", raw: [][]byte{}, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAgentSubagentMultiValues(tt.raw)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseAgentSubagentSkills(t *testing.T) {
	tests := []struct {
		name string
		raw  [][]byte
		want []string
	}{
		{name: "multiple skills", raw: [][]byte{[]byte("demo-skill"), []byte("other-skill")}, want: []string{"demo-skill", "other-skill"}},
		{name: "deduplicates", raw: [][]byte{[]byte("demo-skill"), []byte("demo-skill")}, want: []string{"demo-skill"}},
		{name: "empty yields empty", raw: [][]byte{}, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAgentSubagentMultiValues(tt.raw)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAgentSubagentTasksTableAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		tasks      map[int64]*gen.AgentSubagentTask
		wantStatus int
		wantBody   string
	}{
		{
			name: "renders task table",
			tasks: map[int64]*gen.AgentSubagentTask{
				1: {
					ID:           1,
					SessionID:    "session-1",
					SubagentName: "explore",
					Description:  "Find auth code",
					Prompt:       "Locate authentication middleware",
					Status:       "completed",
					StartedAt:    time.Now().UTC(),
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "explore",
		},
		{
			name:       "renders empty state",
			tasks:      map[int64]*gen.AgentSubagentTask{},
			wantStatus: http.StatusOK,
			wantBody:   "No delegated tasks found",
		},
		{
			name: "renders failed status badge",
			tasks: map[int64]*gen.AgentSubagentTask{
				2: {
					ID:           2,
					SubagentName: "general",
					Description:  "Retry task",
					Status:       "failed",
					ErrorText:    "model unavailable",
					StartedAt:    time.Now().UTC(),
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{agentSubagentTasks: tt.tasks}
			app := setupAuthenticatedApp(t, ts)

			req := httptest.NewRequest(http.MethodGet, "/service/web/agent-subagents/tasks", http.NoBody)
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

func TestAgentSubagentCreateAuthenticated(t *testing.T) {
	tests := []struct {
		name        string
		form        map[string]string
		tools       []string
		skills      []string
		wantStatus  int
		wantBody    string
		wantEnabled bool
		checkStore  bool
	}{
		{
			name: "creates subagent and returns row html",
			form: map[string]string{
				"flag":          "code-reviewer",
				"name":          "code-reviewer",
				"description":   "Reviews code",
				"system_prompt": "You review code.",
				"enabled":       "true",
			},
			tools:       []string{"read_file", "run_terminal"},
			skills:      []string{"code-review", "lint-rules"},
			wantStatus:  http.StatusOK,
			wantBody:    "code-reviewer",
			wantEnabled: true,
			checkStore:  true,
		},
		{
			name: "validation error returns form",
			form: map[string]string{
				"flag":          "code-reviewer",
				"name":          "code-reviewer",
				"description":   "",
				"system_prompt": "p",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "Description is required",
		},
		{
			name: "duplicate flag rejected",
			form: map[string]string{
				"flag":          "existing",
				"name":          "existing",
				"description":   "Existing",
				"system_prompt": "p",
			},
			tools:      []string{"read_file"},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "Flag already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentSubagents: map[string]*gen.AgentSubagent{
					"existing": {
						Flag:         "existing",
						Name:         "existing",
						Description:  "Existing",
						SystemPrompt: "p",
						Enabled:      true,
					},
				},
			}
			app := setupAuthenticatedApp(t, ts)

			body := buildSubagentFormBody(tt.form, tt.tools, tt.skills)
			req := httptest.NewRequest(http.MethodPost, "/service/web/agent-subagents", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			respBody, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(respBody), tt.wantBody)
			if tt.checkStore {
				subagent := ts.agentSubagents[tt.form["flag"]]
				require.NotNil(t, subagent)
				assert.Equal(t, tt.wantEnabled, subagent.Enabled)
				assert.Equal(t, []string{"read_file", "run_terminal"}, subagent.Tools)
				assert.Equal(t, []string{"code-review", "lint-rules"}, subagent.Skills)
			}
		})
	}
}

func buildSubagentFormBody(values map[string]string, tools, skills []string) string {
	form := url.Values{}
	for key, value := range values {
		form.Set(key, value)
	}
	for _, tool := range tools {
		form.Add("tools", tool)
	}
	for _, skill := range skills {
		form.Add("skills", skill)
	}
	return form.Encode()
}

func TestAgentSubagentDeleteAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		flag       string
		wantStatus int
		wantEmpty  bool
	}{
		{name: "deletes existing subagent", flag: "code-reviewer", wantStatus: http.StatusOK, wantEmpty: true},
		{name: "returns not found for missing subagent", flag: "missing", wantStatus: http.StatusNotFound},
		{name: "returns empty body on success", flag: "planner", wantStatus: http.StatusOK, wantEmpty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentSubagents: map[string]*gen.AgentSubagent{
					"code-reviewer": {Flag: "code-reviewer", Name: "code-reviewer", Description: "d", SystemPrompt: "p", Enabled: true},
					"planner":       {Flag: "planner", Name: "planner", Description: "d", SystemPrompt: "p", Enabled: true},
				},
			}
			app := setupAuthenticatedApp(t, ts)

			req := httptest.NewRequest(http.MethodDelete, "/service/web/agent-subagents/"+tt.flag, http.NoBody)
			req.Header.Set("Cookie", "accessToken=test-token")
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantEmpty {
				_, ok := ts.agentSubagents[tt.flag]
				assert.False(t, ok)
				respBody, _ := io.ReadAll(resp.Body)
				assert.Empty(t, string(respBody))
			}
		})
	}
}

func TestAgentSubagentCreateInvalidatesPromptCache(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "bumps prompt cache version"},
		{name: "clears stale prompt after create"},
		{name: "invalidates on successful mutation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatagent.ResetPromptCacheForTest()
			before := chatagent.PromptCacheVersion()

			ts := &testStore{agentSubagents: map[string]*gen.AgentSubagent{}}
			app := setupAuthenticatedApp(t, ts)

			body := buildSubagentFormBody(map[string]string{
				"flag":          "cache-subagent",
				"name":          "cache-subagent",
				"description":   "Cache test",
				"system_prompt": "p",
				"enabled":       "true",
			}, []string{"read_file"}, nil)
			req := httptest.NewRequest(http.MethodPost, "/service/web/agent-subagents", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Greater(t, chatagent.PromptCacheVersion(), before)
		})
	}
}
