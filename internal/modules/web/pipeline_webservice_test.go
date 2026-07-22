package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/pkg/capability"
	abilityagent "github.com/flowline-io/flowbot/pkg/capability/agent"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/pipeline"
)

type webStubAgentRunner struct {
	replies []string
	prompts []string
	calls   int
}

type pipelineTestStepResult struct {
	Name   string         `json:"name"`
	Status string         `json:"status"`
	Output map[string]any `json:"output"`
}

type pipelineTestStepResponse struct {
	Success bool                     `json:"success"`
	Steps   []pipelineTestStepResult `json:"steps"`
}

func (s *webStubAgentRunner) Run(_ context.Context, params abilityagent.RunParams) (*abilityagent.RunResult, error) {
	s.prompts = append(s.prompts, params.Prompt)
	reply := "agent-reply"
	if s.calls < len(s.replies) {
		reply = s.replies[s.calls]
	}
	s.calls++
	return &abilityagent.RunResult{Reply: reply, SessionID: "sess-web-mock"}, nil
}

func registerWebAgentRunner(t *testing.T, runner abilityagent.Runner) {
	t.Helper()
	abilityagent.SetRunner(runner)
	require.NoError(t, abilityagent.Register())
	t.Cleanup(func() {
		abilityagent.SetRunner(nil)
		capability.UnregisterInvoker(hub.CapAgent, capability.OpAgentRun)
		hub.Default.Unregister(hub.CapAgent)
	})
}

func TestTestPipelineStepAgentMultiStepOutput(t *testing.T) {
	stub := &webStubAgentRunner{replies: []string{"first-agent-reply", "second-agent-reply"}}
	registerWebAgentRunner(t, stub)

	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ps := store.NewPipelineStore(client)
	require.NoError(t, ps.CreateDefinition(ctx, "agent-chain-test", "", ""))
	yamlDraft := `name: agent-chain-test
steps:
  - name: summarize
    capability: agent
    operation: run
    params:
      prompt: "summarize {{.Event.url}}"
  - name: follow-up
    capability: agent
    operation: run
    params:
      prompt: "Previous reply: {{step \"summarize\" \"reply\"}}"`
	require.NoError(t, client.PipelineDefinition.Update().
		SetYamlDraft(yamlDraft).
		Where(pipelinedefinition.Name("agent-chain-test")).
		Exec(ctx))

	body, err := sonic.MarshalString(map[string]any{
		"trigger_source":   "event",
		"mock_payload":     map[string]any{"url": "https://example.com/item"},
		"up_to_step_index": 1,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/service/web/pipelines/agent-chain-test/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var payload pipelineTestStepResponse
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.True(t, payload.Success)
	require.Len(t, payload.Steps, 2)
	assert.Equal(t, "ok", payload.Steps[0].Status)
	assert.Equal(t, "first-agent-reply", payload.Steps[0].Output["reply"])
	assert.Equal(t, "ok", payload.Steps[1].Status)
	assert.Equal(t, "second-agent-reply", payload.Steps[1].Output["reply"])
	require.Len(t, stub.prompts, 2)
	assert.Equal(t, "summarize https://example.com/item", stub.prompts[0])
	assert.Equal(t, "Previous reply: first-agent-reply", stub.prompts[1])
}

func TestTestPipelineStepRejectsWhitespaceAgentPrompt(t *testing.T) {
	registerWebAgentRunner(t, &webStubAgentRunner{})

	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ps := store.NewPipelineStore(client)
	require.NoError(t, ps.CreateDefinition(ctx, "agent-empty-prompt", "", ""))
	require.NoError(t, client.PipelineDefinition.Update().
		SetYamlDraft(`name: agent-empty-prompt
steps:
  - name: bad
    capability: agent
    operation: run
    params:
      prompt: "   "`).
		Where(pipelinedefinition.Name("agent-empty-prompt")).
		Exec(ctx))

	body := bytes.NewBufferString(`{"trigger_source":"event","mock_payload":{},"up_to_step_index":0}`)
	req := httptest.NewRequest(http.MethodPost, "/service/web/pipelines/agent-empty-prompt/test", body)
	req.Header.Set("Content-Type", "application/json")
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"success":false`)
}

func TestSetPipelineEnabledPauseAndResume(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ps := store.NewPipelineStore(client)
	require.NoError(t, ps.CreateDefinition(ctx, "pause-test", "", ""))
	yaml := "name: pause-test\nenabled: true\ntriggers: []\nsteps: []"
	_, err := ps.UpdateDefinitionDraft(ctx, "pause-test", yaml, 1)
	require.NoError(t, err)
	_, err = ps.PublishDefinition(ctx, "pause-test", 2)
	require.NoError(t, err)

	pauseBody := bytes.NewBufferString(`{"enabled":false}`)
	req := httptest.NewRequest(http.MethodPut, "/service/web/pipelines/pause-test/enabled", pauseBody)
	req.Header.Set("Content-Type", "application/json")
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	pauseHTML, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(pauseHTML), "Paused")
	assert.Contains(t, string(pauseHTML), `btn-resume-pause-test`)

	def, err := ps.GetDefinitionByName(ctx, "pause-test")
	require.NoError(t, err)
	require.NotNil(t, def.YamlPublished)
	assert.False(t, pipeline.IsEnabledInYAML(*def.YamlPublished))

	resumeBody := bytes.NewBufferString(`{"enabled":true}`)
	req = httptest.NewRequest(http.MethodPut, "/service/web/pipelines/pause-test/enabled", resumeBody)
	req.Header.Set("Content-Type", "application/json")
	addWebAuth(req)
	resp, err = app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resumeHTML, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(resumeHTML), "Active")
	assert.Contains(t, string(resumeHTML), `btn-pause-pause-test`)
}

func TestSetPipelineEnabledRejectsDraftPipeline(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ps := store.NewPipelineStore(client)
	require.NoError(t, ps.CreateDefinition(ctx, "draft-pause", "", ""))

	body := bytes.NewBufferString(`{"enabled":false}`)
	req := httptest.NewRequest(http.MethodPut, "/service/web/pipelines/draft-pause/enabled", body)
	req.Header.Set("Content-Type", "application/json")
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	_, err = ps.GetDefinitionByName(ctx, "draft-pause")
	require.NoError(t, err)
}

type agentRunOptionsTestSkill struct {
	Name string `json:"name"`
}

type agentRunOptionsTestData struct {
	Tools  []string                   `json:"tools"`
	Skills []agentRunOptionsTestSkill `json:"skills"`
}

type agentRunOptionsTestPayload struct {
	Status string                  `json:"status"`
	Data   agentRunOptionsTestData `json:"data"`
}

func TestGetAgentRunOptions(t *testing.T) {
	app, _, _ := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	req := httptest.NewRequest(http.MethodGet, "/service/web/pipelines/agent-run-options", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var payload agentRunOptionsTestPayload
	require.NoError(t, sonic.Unmarshal(body, &payload))
	assert.Equal(t, "ok", string(payload.Status))
	assert.NotEmpty(t, payload.Data.Tools)
}

func TestCreatePipelineChineseName(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	name := "数据同步"
	body := strings.NewReader(url.Values{
		"name":        {name},
		"description": {"中文描述"},
	}.Encode())
	req := httptest.NewRequest(http.MethodPost, "/service/web/pipelines", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/service/web/pipelines/%E6%95%B0%E6%8D%AE%E5%90%8C%E6%AD%A5", resp.Header.Get("HX-Redirect"))

	ctx := context.Background()
	def, err := store.NewPipelineStore(client).GetDefinitionByName(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, name, def.Name)
}

func TestRenamePipeline(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ps := store.NewPipelineStore(client)

	tests := []struct {
		name       string
		setup      func(t *testing.T) (oldName, newName string)
		wantStatus int
		wantCode   string
	}{
		{
			name: "renames pipeline and returns new name",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				require.NoError(t, ps.CreateDefinition(ctx, "web-rename-src", "", ""))
				_, err := ps.UpdateDefinitionDraft(ctx, "web-rename-src", "name: web-rename-src\ntriggers: []\nsteps: []", 1)
				require.NoError(t, err)
				return "web-rename-src", "web-rename-dst"
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "rejects invalid name",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				require.NoError(t, ps.CreateDefinition(ctx, "web-rename-bad", "", ""))
				return "web-rename-bad", "-bad"
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "rejects duplicate name",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				require.NoError(t, ps.CreateDefinition(ctx, "web-rename-dup-src", "", ""))
				require.NoError(t, ps.CreateDefinition(ctx, "web-rename-dup-dst", "", ""))
				return "web-rename-dup-src", "web-rename-dup-dst"
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "ALREADY_EXISTS",
		},
		{
			name: "missing pipeline returns not found",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				return "web-rename-missing", "web-rename-missing-dst"
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldName, newName := tt.setup(t)
			body, err := sonic.MarshalString(map[string]string{"name": newName})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPut, "/service/web/pipelines/"+url.PathEscape(oldName)+"/rename", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			addWebAuth(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tt.wantStatus, resp.StatusCode)

			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if tt.wantStatus == http.StatusOK {
				var payload struct {
					Name string `json:"name"`
				}
				require.NoError(t, sonic.Unmarshal(raw, &payload))
				assert.Equal(t, newName, payload.Name)
				def, getErr := ps.GetDefinitionByName(ctx, newName)
				require.NoError(t, getErr)
				assert.Equal(t, newName, def.Name)
				return
			}
			assert.Contains(t, string(raw), tt.wantCode)
		})
	}
}

func TestPipelineEditorPageChineseName(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	name := "演示1"
	require.NoError(t, store.NewPipelineStore(client).CreateDefinition(ctx, name, "", ""))

	req := httptest.NewRequest(http.MethodGet, "/service/web/pipelines/"+url.PathEscape(name), http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, name)
	assert.NotContains(t, bodyStr, url.PathEscape(name))
}
