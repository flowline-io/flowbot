package chatagent

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
)

func TestBuildKnowledgeMetadataPrompt_Truncates(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", knowledgeMetadataContentMaxBytes+100)
	prompt := buildKnowledgeMetadataPrompt("/docs/x.md", long)
	assert.Contains(t, prompt, "Path: /docs/x.md")
	assert.Contains(t, prompt, "[truncated]")
	assert.Less(t, len(prompt), knowledgeMetadataContentMaxBytes+200)
}

func TestParseKnowledgeMetadataJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		want    KnowledgeMetadata
		wantErr bool
	}{
		{
			name: "plain json",
			raw:  `{"title":"Hello","tags":["a","b","c"],"summary":"One line"}`,
			want: KnowledgeMetadata{Title: "Hello", Tags: []string{"a", "b", "c"}, Summary: "One line"},
		},
		{
			name: "fenced json",
			raw:  "```json\n{\"title\":\"T\",\"tags\":[\"x\"],\"summary\":\"S\"}\n```",
			want: KnowledgeMetadata{Title: "T", Tags: []string{"x"}, Summary: "S"},
		},
		{
			name: "wrapped in prose",
			raw:  "Here you go:\n{\"title\":\"T\",\"tags\":[],\"summary\":\"S\"}\nThanks",
			want: KnowledgeMetadata{Title: "T", Tags: []string{}, Summary: "S"},
		},
		{
			name:    "invalid",
			raw:     "not json",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseKnowledgeMetadataJSON(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSanitizeKnowledgeMetadata(t *testing.T) {
	t.Parallel()
	meta := sanitizeKnowledgeMetadata(KnowledgeMetadata{
		Title:   "  Hello\nWorld  ",
		Summary: "A\nB",
		Tags:    []string{" Flowbot ", "flowbot", "homelab", "a", "b", "c", "d", "e"},
	})
	assert.Equal(t, "Hello World", meta.Title)
	assert.Equal(t, "A B", meta.Summary)
	assert.Equal(t, []string{"Flowbot", "homelab", "a", "b", "c", "d"}, meta.Tags)
}

func TestGenerateKnowledgeMetadataWithLLM_RejectsTooFewTags(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"title":"Doc","tags":["only","two"],"summary":"Sum"}`,
	})
	_, err := generateKnowledgeMetadataWithLLM(
		context.Background(),
		"/p.md",
		"# Doc\n\nBody",
		"fake-model",
		func(context.Context, string) (llms.Model, string, error) {
			return fake, "fake-model", nil
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete metadata")
}

func TestGenerateKnowledgeMetadata_RequiresContentAndModel(t *testing.T) {
	orig := config.App.ChatAgent.ChatModel
	t.Cleanup(func() { config.App.ChatAgent.ChatModel = orig })

	config.App.ChatAgent.ChatModel = "test-model"
	_, err := GenerateKnowledgeMetadata(context.Background(), "/x.md", "   ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")

	config.App.ChatAgent.ChatModel = ""
	_, err = GenerateKnowledgeMetadata(context.Background(), "/x.md", "body")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestGenerateKnowledgeMetadata_UsesInjectedLLM(t *testing.T) {
	orig := config.App.ChatAgent.ChatModel
	config.App.ChatAgent.ChatModel = "test-model"
	t.Cleanup(func() { config.App.ChatAgent.ChatModel = orig })

	restore := SetKnowledgeMetadataLLMForTest(func(
		_ context.Context, path, content, chatModel string, _ knowledgeMetadataModelFunc,
	) (KnowledgeMetadata, error) {
		assert.Equal(t, "/scripts/run.md", path)
		assert.Equal(t, "hello body", content)
		assert.Equal(t, "test-model", chatModel)
		return KnowledgeMetadata{
			Title:   "Generated",
			Tags:    []string{"one", "two", "three"},
			Summary: "A summary",
		}, nil
	})
	t.Cleanup(restore)

	got, err := GenerateKnowledgeMetadata(context.Background(), "/scripts/run.md", "hello body")
	require.NoError(t, err)
	assert.Equal(t, "Generated", got.Title)
	assert.Equal(t, []string{"one", "two", "three"}, got.Tags)
	assert.Equal(t, "A summary", got.Summary)
}

func TestGenerateKnowledgeMetadataWithLLM_ParsesFakeModel(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"title":"Doc","tags":["t1","t2","t3"],"summary":"Sum"}`,
	})
	got, err := generateKnowledgeMetadataWithLLM(
		context.Background(),
		"/p.md",
		"# Doc\n\nBody",
		"fake-model",
		func(context.Context, string) (llms.Model, string, error) {
			return fake, "fake-model", nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "Doc", got.Title)
	assert.Equal(t, []string{"t1", "t2", "t3"}, got.Tags)
	assert.Equal(t, "Sum", got.Summary)
}
