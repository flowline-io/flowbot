package server

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestOpenAIMessagesToLLM(t *testing.T) {
	sys, msgs, err := openAIMessagesToLLM([]openAIChatMsg{
		{Role: "system", Content: openAIContent{Text: "be helpful"}},
		{Role: "user", Content: openAIContent{Text: "hi"}},
		{
			Role:    "assistant",
			Content: openAIContent{Text: "calling"},
			ToolCalls: []openAIToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: &openAIFunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"a.go"}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_1", Name: "read_file", Content: openAIContent{Text: "ok"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "be helpful", sys)
	require.Len(t, msgs, 3)
	assert.Equal(t, llms.ChatMessageTypeHuman, msgs[0].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, msgs[1].Role)
	assert.Equal(t, llms.ChatMessageTypeTool, msgs[2].Role)
}

func TestOpenAIContentUnmarshal(t *testing.T) {
	var c openAIContent
	require.NoError(t, sonic.Unmarshal([]byte(`"hello"`), &c))
	assert.Equal(t, "hello", c.Text)

	require.NoError(t, sonic.Unmarshal([]byte(`[{"type":"text","text":"a"},{"type":"text","text":"b"}]`), &c))
	assert.Equal(t, "ab", c.Text)

	raw, err := sonic.Marshal(openAIContent{Text: "out"})
	require.NoError(t, err)
	assert.JSONEq(t, `"out"`, string(raw))
}

func TestOpenAIResponseFromResult_ToolCalls(t *testing.T) {
	result := agentllm.AssistantResult{
		Content: "",
		ToolCalls: []llms.ToolCall{{
			ID:   "c1",
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      "list_dir",
				Arguments: `{}`,
			},
		}},
		StopReason: "tool_calls",
	}
	resp := openAIResponseFromResult("id1", "m1", 1, result)
	require.Len(t, resp.Choices, 1)
	require.NotNil(t, resp.Choices[0].FinishReason)
	assert.Equal(t, "tool_calls", *resp.Choices[0].FinishReason)
	require.NotNil(t, resp.Choices[0].Message)
	require.Len(t, resp.Choices[0].Message.ToolCalls, 1)
	assert.Equal(t, "list_dir", resp.Choices[0].Message.ToolCalls[0].Function.Name)
}

func TestWriteAgentLLMStreamResult_EmitsDone(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	writeAgentLLMStreamResult(w, "chatcmpl-1", "forced-model", 1, agentllm.AssistantResult{
		Content:    "hello world",
		StopReason: "complete",
	})
	out := buf.String()
	assert.Contains(t, out, `"content":"hello world"`)
	assert.Contains(t, out, `"finish_reason":"stop"`)
	assert.True(t, strings.HasSuffix(strings.TrimSpace(out), "data: [DONE]"), out)
}

func TestAgentLLMChatCompletions_AuthAndModelOverride(t *testing.T) {
	client := sqlitetest.OpenClient(t, "agent_llm_proxy")
	origDB := store.Database
	t.Cleanup(func() { store.Database = origDB })
	store.Database = &tokenTestAdapter{client: client}

	raw := "agent-headless-token"
	require.NoError(t, store.NewModuleDataStore(client).ParameterSet(
		context.Background(),
		auth.HashToken(raw),
		types.KV{"uid": "u1", "topic": "t", "scopes": []string{auth.ScopeAgentHeadless}},
		time.Now().Add(time.Hour),
	))
	t.Cleanup(func() { route.SetAccessTokenStore(nil) })
	WireAccessTokenStore()

	origChat := config.App.ChatAgent
	config.App.ChatAgent.ChatModel = "forced-model"
	t.Cleanup(func() { config.App.ChatAgent = origChat })

	origModels := config.App.Models
	config.App.Models = []config.Model{{
		Provider:   "openai_compatible",
		ModelNames: []string{"forced-model"},
		ApiKey:     "sk-test",
		BaseUrl:    "http://127.0.0.1:9/v1",
	}}
	t.Cleanup(func() { config.App.Models = origModels })

	agentllm.ResetModelPoolForTest()
	t.Cleanup(agentllm.ResetModelPoolForTest)
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "hello-from-proxy"})
	agentllm.SetModelCreatorForTest(func(_ context.Context, modelName string) (llms.Model, string, error) {
		assert.Equal(t, "forced-model", modelName)
		return fake, modelName, nil
	})
	t.Cleanup(func() { agentllm.SetModelCreatorForTest(nil) })

	app := newTestApp()
	RegisterAgentLLMRoutes(app)

	t.Run("rejects missing token", func(t *testing.T) {
		body := `{"model":"client-model","messages":[{"role":"user","content":"hi"}]}`
		req := httptest.NewRequest(http.MethodPost, "/agent/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("forces server model and returns text", func(t *testing.T) {
		body := `{"model":"client-model","messages":[{"role":"user","content":"hi"}],"stream":false}`
		req := httptest.NewRequest(http.MethodPost, "/agent/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-AccessToken", raw)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		rawBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		var out openAIChatResponse
		require.NoError(t, sonic.Unmarshal(rawBody, &out))
		assert.Equal(t, "forced-model", out.Model)
		require.Len(t, out.Choices, 1)
		require.NotNil(t, out.Choices[0].Message)
		assert.Equal(t, "hello-from-proxy", out.Choices[0].Message.Content.Text)
		assert.Equal(t, 1, fake.Calls())
	})
}
