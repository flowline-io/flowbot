package clip

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/capability"
	capclip "github.com/flowline-io/flowbot/pkg/capability/clip"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type memPersister struct {
	bySlug map[string]*capclip.Record
}

func (m *memPersister) CreateClip(_ context.Context, slug, title, description, content, createdBy string) error {
	if m.bySlug == nil {
		m.bySlug = map[string]*capclip.Record{}
	}
	m.bySlug[slug] = &capclip.Record{
		Slug: slug, Title: title, Description: description, Content: content, CreatedBy: createdBy,
	}
	return nil
}

func (m *memPersister) GetClipBySlug(_ context.Context, slug string) (*capclip.Record, error) {
	if m.bySlug == nil {
		return nil, nil
	}
	return m.bySlug[slug], nil
}

func TestAbsoluteURL(t *testing.T) {
	prev := config.App.Flowbot.URL
	t.Cleanup(func() { config.App.Flowbot.URL = prev })

	tests := []struct {
		name   string
		base   string
		cfgURL string
		in     string
		want   string
	}{
		{name: "explicit base with path", base: "https://ex.com/", in: "/c/abc", want: "https://ex.com/c/abc"},
		{name: "slug only with base", base: "https://ex.com", in: "abc", want: "https://ex.com/c/abc"},
		{name: "falls back to config url", base: "", cfgURL: "https://cfg.example", in: "/c/x", want: "https://cfg.example/c/x"},
		{name: "relative when no base", base: "", cfgURL: "", in: "/c/x", want: "/c/x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App.Flowbot.URL = tt.cfgURL
			assert.Equal(t, tt.want, AbsoluteURL(tt.base, tt.in))
		})
	}
}

func TestCreateAndGetTools(t *testing.T) {
	t.Cleanup(func() {
		capability.UnregisterInvoker(hub.CapClip, capclip.OpCreate)
		capability.UnregisterInvoker(hub.CapClip, capclip.OpGet)
		capability.UnregisterInvoker(hub.CapClip, capclip.OpHealth)
		hub.Default.Unregister(hub.CapClip)
		capclip.SetPersister(nil)
		capclip.SetMetaLLMForTest(nil)
	})

	prevModel := config.App.ChatAgent.ChatModel
	config.App.ChatAgent.ChatModel = ""
	t.Cleanup(func() { config.App.ChatAgent.ChatModel = prevModel })

	capclip.SetPersister(&memPersister{})
	require.NoError(t, capclip.Register())

	create := CreateTool{PublicBaseURL: "https://flowbot.test"}
	get := GetTool{}

	tests := []struct {
		name    string
		run     func(t *testing.T) string
		wantSub string
	}{
		{
			name: "create returns absolute url",
			run: func(t *testing.T) string {
				res, err := create.Execute(context.Background(), "c1", map[string]any{
					"content": "# Hello\n\nbody",
				}, nil)
				require.NoError(t, err)
				require.False(t, res.IsError)
				text := toolText(res)
				assert.Contains(t, text, "https://flowbot.test/c/")
				return text
			},
			wantSub: "https://flowbot.test/c/",
		},
		{
			name: "get loads content by slug",
			run: func(t *testing.T) string {
				created, err := create.Execute(context.Background(), "c2", map[string]any{
					"content": "secret-markdown-body",
				}, nil)
				require.NoError(t, err)
				slug := slugFromToolText(toolText(created))
				require.NotEmpty(t, slug)
				res, err := get.Execute(context.Background(), "g1", map[string]any{"slug": slug}, nil)
				require.NoError(t, err)
				require.False(t, res.IsError)
				return toolText(res)
			},
			wantSub: "secret-markdown-body",
		},
		{
			name: "create rejects empty content",
			run: func(t *testing.T) string {
				res, err := create.Execute(context.Background(), "c3", map[string]any{"content": "  "}, nil)
				require.NoError(t, err)
				assert.True(t, res.IsError)
				return toolText(res)
			},
			wantSub: "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.run(t)
			assert.Contains(t, got, tt.wantSub)
		})
	}
}

func TestRegisterActiveToolNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "registers both tools"},
		{name: "active names include create and get"},
		{name: "nil registry errors"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			switch tt.name {
			case "registers both tools":
				reg := tool.NewRegistry()
				require.NoError(t, Register(reg, "https://x.test"))
				_, ok := reg.Get(CreateToolName)
				assert.True(t, ok)
				_, ok = reg.Get(GetToolName)
				assert.True(t, ok)
			case "active names include create and get":
				assert.Equal(t, []string{CreateToolName, GetToolName}, ActiveToolNames())
			default:
				err := Register(nil, "")
				require.Error(t, err)
			}
		})
	}
}

func toolText(res msg.ToolResultMessage) string {
	for _, part := range res.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}

func slugFromToolText(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if after, ok := strings.CutPrefix(line, "slug: "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
