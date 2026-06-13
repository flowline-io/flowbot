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

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func (s *testStore) ListAgentSkills(_ context.Context, enabledOnly bool) ([]*gen.AgentSkill, error) {
	if s.agentSkillsErr != nil {
		return nil, s.agentSkillsErr
	}
	rows := make([]*gen.AgentSkill, 0, len(s.agentSkills))
	for _, skill := range s.agentSkills {
		if enabledOnly && !skill.Enabled {
			continue
		}
		rows = append(rows, skill)
	}
	return rows, nil
}

func (s *testStore) GetAgentSkillByFlag(_ context.Context, flag string) (*gen.AgentSkill, error) {
	skill, ok := s.agentSkills[flag]
	if !ok {
		return nil, types.ErrNotFound
	}
	return skill, nil
}

func (s *testStore) CreateAgentSkill(_ context.Context, skill *gen.AgentSkill) error {
	if s.createAgentSkillFn != nil {
		return s.createAgentSkillFn(skill)
	}
	if s.agentSkills == nil {
		s.agentSkills = make(map[string]*gen.AgentSkill)
	}
	if _, exists := s.agentSkills[skill.Flag]; exists {
		return types.Errorf(types.ErrInvalidArgument, "agent_skills_flag_key")
	}
	for _, existing := range s.agentSkills {
		if existing.Name == skill.Name {
			return types.Errorf(types.ErrInvalidArgument, "agent_skills_name_key")
		}
	}
	s.agentSkills[skill.Flag] = skill
	return nil
}

func (s *testStore) UpdateAgentSkill(_ context.Context, skill *gen.AgentSkill) error {
	if s.updateAgentSkillFn != nil {
		return s.updateAgentSkillFn(skill)
	}
	if s.agentSkills == nil {
		return types.ErrNotFound
	}
	if _, ok := s.agentSkills[skill.Flag]; !ok {
		return types.ErrNotFound
	}
	for flag, existing := range s.agentSkills {
		if flag != skill.Flag && existing.Name == skill.Name {
			return types.Errorf(types.ErrInvalidArgument, "agent_skills_name_key")
		}
	}
	skill.UpdatedAt = time.Now().UTC()
	s.agentSkills[skill.Flag] = skill
	return nil
}

func (s *testStore) DeleteAgentSkill(_ context.Context, flag string) error {
	if s.deleteAgentSkillFn != nil {
		return s.deleteAgentSkillFn(flag)
	}
	if s.agentSkills == nil {
		return types.ErrNotFound
	}
	if _, ok := s.agentSkills[flag]; !ok {
		return types.ErrNotFound
	}
	delete(s.agentSkills, flag)
	return nil
}

func TestValidateAgentSkillForm(t *testing.T) {
	tests := []struct {
		name    string
		item    model.AgentSkill
		isNew   bool
		wantKey string
	}{
		{name: "empty flag rejected on create", item: model.AgentSkill{Name: "demo", Description: "d", Content: "c"}, isNew: true, wantKey: "flag"},
		{name: "invalid slug rejected", item: model.AgentSkill{Flag: "Bad Flag", Name: "demo", Description: "d", Content: "c"}, isNew: true, wantKey: "flag"},
		{name: "empty name rejected", item: model.AgentSkill{Flag: "demo", Description: "d", Content: "c"}, isNew: true, wantKey: "name"},
		{name: "empty description rejected", item: model.AgentSkill{Flag: "demo", Name: "demo", Content: "c"}, isNew: true, wantKey: "description"},
		{name: "empty content rejected", item: model.AgentSkill{Flag: "demo", Name: "demo", Description: "d"}, isNew: true, wantKey: "content"},
		{name: "valid update passes without flag", item: model.AgentSkill{Name: "demo", Description: "d", Content: "body"}, isNew: false, wantKey: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateAgentSkillForm(tt.item, tt.isNew)
			if tt.wantKey == "" {
				require.Empty(t, errs)
				return
			}
			_, ok := errs[tt.wantKey]
			assert.True(t, ok, "want error for %q, got %v", tt.wantKey, errs)
		})
	}
}

func TestMapAgentSkillUniqueError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantKey string
	}{
		{name: "flag constraint", err: types.Errorf(types.ErrInvalidArgument, "agent_skills_flag_key"), wantKey: "flag"},
		{name: "name constraint", err: types.Errorf(types.ErrInvalidArgument, "agent_skills_name_key"), wantKey: "name"},
		{name: "unrelated error", err: types.Errorf(types.ErrInvalidArgument, "invalid agent skill flag"), wantKey: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := mapAgentSkillUniqueError(tt.err)
			if tt.wantKey == "" {
				assert.Nil(t, errs)
				return
			}
			_, ok := errs[tt.wantKey]
			assert.True(t, ok, "want error for %q, got %v", tt.wantKey, errs)
		})
	}
}

func TestAgentSkillsPageUnauthenticated(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "page redirects to login", path: "/service/web/agent-skills"},
		{name: "list redirects to login", path: "/service/web/agent-skills/list"},
		{name: "create redirects to login", path: "/service/web/agent-skills"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			method := http.MethodGet
			if tt.name == "create redirects to login" {
				method = http.MethodPost
			}
			req := httptest.NewRequest(method, tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			if len(body) > 0 {
				assert.NotContains(t, string(body), "Agent Skills")
			}
		})
	}
}

func TestAgentSkillCreateAuthenticated(t *testing.T) {
	tests := []struct {
		name        string
		form        map[string]string
		wantStatus  int
		wantBody    string
		wantEnabled bool
		checkStore  bool
	}{
		{
			name: "creates skill and returns row html",
			form: map[string]string{
				"flag":        "demo-skill",
				"name":        "demo-skill",
				"description": "Demo skill",
				"content":     "# Demo\nBody",
				"enabled":     "true",
			},
			wantStatus:  http.StatusOK,
			wantBody:    "demo-skill",
			wantEnabled: true,
			checkStore:  true,
		},
		{
			name: "creates disabled skill when enabled unchecked",
			form: map[string]string{
				"flag":        "disabled-skill",
				"name":        "disabled-skill",
				"description": "Disabled skill",
				"content":     "body",
			},
			wantStatus:  http.StatusOK,
			wantBody:    "disabled-skill",
			wantEnabled: false,
			checkStore:  true,
		},
		{
			name: "validation error returns form",
			form: map[string]string{
				"flag":        "demo-skill",
				"name":        "demo-skill",
				"description": "",
				"content":     "body",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "Description is required",
		},
		{
			name: "duplicate flag rejected",
			form: map[string]string{
				"flag":        "existing",
				"name":        "existing",
				"description": "Demo skill",
				"content":     "body",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "Flag already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentSkills: map[string]*gen.AgentSkill{
					"existing": {
						Flag:        "existing",
						Name:        "existing",
						Description: "Existing",
						Content:     "body",
						Enabled:     true,
					},
				},
			}
			app := setupAuthenticatedApp(t, ts)

			body := buildFormBody(tt.form)
			req := httptest.NewRequest(http.MethodPost, "/service/web/agent-skills", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			respBody, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(respBody), tt.wantBody)
			if tt.checkStore {
				skill := ts.agentSkills[tt.form["flag"]]
				require.NotNil(t, skill)
				assert.Equal(t, tt.wantEnabled, skill.Enabled)
			}
		})
	}
}

func TestAgentSkillCreateInvalidatesPromptCache(t *testing.T) {
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

			ts := &testStore{agentSkills: map[string]*gen.AgentSkill{}}
			app := setupAuthenticatedApp(t, ts)

			body := buildFormBody(map[string]string{
				"flag":        "cache-skill",
				"name":        "cache-skill",
				"description": "Cache test",
				"content":     "body",
				"enabled":     "true",
			})
			req := httptest.NewRequest(http.MethodPost, "/service/web/agent-skills", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Greater(t, chatagent.PromptCacheVersion(), before)
		})
	}
}

func TestAgentSkillUpdateAuthenticated(t *testing.T) {
	tests := []struct {
		name        string
		form        map[string]string
		wantStatus  int
		wantBody    string
		wantEnabled bool
	}{
		{
			name: "updates description and disables skill",
			form: map[string]string{
				"name":        "demo-skill",
				"description": "Updated description",
				"content":     "updated body",
			},
			wantStatus:  http.StatusOK,
			wantBody:    "Updated description",
			wantEnabled: false,
		},
		{
			name: "keeps skill enabled when checked",
			form: map[string]string{
				"name":        "demo-skill",
				"description": "Still enabled",
				"content":     "body",
				"enabled":     "true",
			},
			wantStatus:  http.StatusOK,
			wantBody:    "Still enabled",
			wantEnabled: true,
		},
		{
			name: "missing skill returns not found",
			form: map[string]string{
				"name":        "missing",
				"description": "Missing",
				"content":     "body",
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Agent skill not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentSkills: map[string]*gen.AgentSkill{
					"demo-skill": {
						Flag:        "demo-skill",
						Name:        "demo-skill",
						Description: "Original",
						Content:     "body",
						Enabled:     true,
					},
				},
			}
			app := setupAuthenticatedApp(t, ts)

			flag := "demo-skill"
			if tt.name == "missing skill returns not found" {
				flag = "missing"
			}

			body := buildFormBody(tt.form)
			req := httptest.NewRequest(http.MethodPut, "/service/web/agent-skills/"+flag, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			respBody, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(respBody), tt.wantBody)
			if tt.wantStatus == http.StatusOK {
				skill := ts.agentSkills["demo-skill"]
				require.NotNil(t, skill)
				assert.Equal(t, tt.wantEnabled, skill.Enabled)
			}
		})
	}
}

func TestAgentSkillDeleteAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		flag       string
		wantStatus int
		wantEmpty  bool
	}{
		{name: "deletes existing skill", flag: "demo-skill", wantStatus: http.StatusOK, wantEmpty: true},
		{name: "returns not found for missing skill", flag: "missing", wantStatus: http.StatusNotFound},
		{name: "returns empty body on success", flag: "other-skill", wantStatus: http.StatusOK, wantEmpty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{
				agentSkills: map[string]*gen.AgentSkill{
					"demo-skill": {
						Flag:        "demo-skill",
						Name:        "demo-skill",
						Description: "Demo",
						Content:     "body",
						Enabled:     true,
					},
					"other-skill": {
						Flag:        "other-skill",
						Name:        "other-skill",
						Description: "Other",
						Content:     "body",
						Enabled:     true,
					},
				},
			}
			app := setupAuthenticatedApp(t, ts)

			req := httptest.NewRequest(http.MethodDelete, "/service/web/agent-skills/"+tt.flag, nil)
			req.Header.Set("Cookie", "accessToken=test-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantEmpty {
				_, ok := ts.agentSkills[tt.flag]
				assert.False(t, ok)
				respBody, _ := io.ReadAll(resp.Body)
				assert.Empty(t, string(respBody))
			}
		})
	}
}

func setupAuthenticatedApp(t *testing.T, ts *testStore) *fiber.App {
	t.Helper()
	store.Database = ts
	handler = moduleHandler{authConfig: AuthConfig{Username: "admin", Password: "admin"}}
	config = configType{Enabled: true, Auth: AuthConfig{Username: "admin", Password: "admin"}}
	ts.paramGetFn = func(_ context.Context, flag string) (gen.Parameter, error) {
		return gen.Parameter{
			ID:        1,
			Flag:      flag,
			Params:    map[string]any{"uid": "testuser", "topic": "test"},
			ExpiredAt: time.Now().Add(time.Hour),
		}, nil
	}
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app
}

func buildFormBody(values map[string]string) string {
	form := url.Values{}
	for key, value := range values {
		form.Set(key, value)
	}
	return form.Encode()
}
