package clip

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

type memPersister struct {
	mu     sync.Mutex
	bySlug map[string]struct {
		title, description, content, createdBy string
	}
	failOnce error
}

func (m *memPersister) CreateClip(_ context.Context, slug, title, description, content, createdBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOnce != nil {
		err := m.failOnce
		m.failOnce = nil
		return err
	}
	if _, ok := m.bySlug[slug]; ok {
		return errors.New("UNIQUE constraint failed: clips.slug")
	}
	if m.bySlug == nil {
		m.bySlug = map[string]struct {
			title, description, content, createdBy string
		}{}
	}
	m.bySlug[slug] = struct {
		title, description, content, createdBy string
	}{title, description, content, createdBy}
	return nil
}

func (m *memPersister) GetClipBySlug(_ context.Context, slug string) (*Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.bySlug[slug]
	if !ok {
		return nil, nil
	}
	return &Record{
		Slug:        slug,
		Title:       row.title,
		Description: row.description,
		Content:     row.content,
		CreatedBy:   row.createdBy,
	}, nil
}

func unregisterClipCapability() {
	capability.UnregisterInvoker(hub.CapClip, OpCreate)
	capability.UnregisterInvoker(hub.CapClip, OpGet)
	capability.UnregisterInvoker(hub.CapClip, OpHealth)
	hub.Default.Unregister(hub.CapClip)
	SetPersister(nil)
	SetMetaLLMForTest(nil)
}

func TestRegisterAndCreate(t *testing.T) {
	t.Cleanup(unregisterClipCapability)

	p := &memPersister{}
	SetPersister(p)
	SetMetaLLMForTest(func(_ context.Context, _ string, _ metaModelFunc) (Meta, error) {
		return Meta{Title: "LLM Title", Description: "LLM description for preview"}, nil
	})
	require.NoError(t, Register())

	tests := []struct {
		name      string
		params    map[string]any
		wantErr   bool
		wantTitle string
		errIs     error
	}{
		{
			name:      "creates clip with llm meta",
			params:    map[string]any{"content": "# Hello\n\nbody text here", "created_by": "u1"},
			wantTitle: "LLM Title",
		},
		{
			name:    "rejects empty content",
			params:  map[string]any{"content": "   "},
			wantErr: true,
			errIs:   types.ErrInvalidArgument,
		},
		{
			name:    "rejects missing content",
			params:  map[string]any{},
			wantErr: true,
			errIs:   types.ErrInvalidArgument,
		},
		{
			name:    "rejects oversized content",
			params:  map[string]any{"content": strings.Repeat("a", MaxContentBytes+1)},
			wantErr: true,
			errIs:   types.ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := capability.Invoke(context.Background(), hub.CapClip, OpCreate, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			data, ok := res.Data.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.wantTitle, data["title"])
			assert.NotEmpty(t, data["slug"])
			assert.Equal(t, "/c/"+data["slug"].(string), data["url"])
			assert.Equal(t, "LLM description for preview", data["description"])
		})
	}
}

func TestCreateFallsBackWhenLLMFails(t *testing.T) {
	t.Cleanup(unregisterClipCapability)

	SetPersister(&memPersister{})
	SetMetaLLMForTest(func(_ context.Context, _ string, _ metaModelFunc) (Meta, error) {
		return Meta{}, errors.New("llm down")
	})
	require.NoError(t, Register())

	tests := []struct {
		name      string
		content   string
		wantTitle string
	}{
		{name: "heading title", content: "# Fuselage Auth\n\nDetails about sessions.", wantTitle: "Fuselage Auth"},
		{name: "plain first line", content: "Plain note without heading\n\nmore", wantTitle: "Plain note without heading"},
		{name: "empty-ish still untitled", content: "###\n\n", wantTitle: "Untitled clip"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := capability.Invoke(context.Background(), hub.CapClip, OpCreate, map[string]any{
				"content": tt.content,
			})
			require.NoError(t, err)
			data, ok := res.Data.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.wantTitle, data["title"])
			assert.NotEmpty(t, data["description"])
		})
	}
}

func TestCreateUnavailableWithoutPersister(t *testing.T) {
	t.Cleanup(unregisterClipCapability)

	SetPersister(nil)
	require.NoError(t, Register())

	tests := []struct {
		name string
	}{
		{name: "create without persister"},
		{name: "health reports unavailable"},
		{name: "invoke still typed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "create without persister":
				_, err := capability.Invoke(context.Background(), hub.CapClip, OpCreate, map[string]any{
					"content": "x",
				})
				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrUnavailable)
			case "health reports unavailable":
				res, err := capability.Invoke(context.Background(), hub.CapClip, OpHealth, nil)
				require.NoError(t, err)
				data, ok := res.Data.(map[string]any)
				require.True(t, ok)
				assert.Equal(t, false, data["ready"])
			default:
				assert.NotNil(t, hub.CapClip)
			}
		})
	}
}

func TestGetClipBySlug(t *testing.T) {
	t.Cleanup(unregisterClipCapability)

	p := &memPersister{}
	SetPersister(p)
	SetMetaLLMForTest(func(_ context.Context, _ string, _ metaModelFunc) (Meta, error) {
		return Meta{Title: "T", Description: "D"}, nil
	})
	require.NoError(t, Register())

	created, err := capability.Invoke(context.Background(), hub.CapClip, OpCreate, map[string]any{
		"content": "# Hello\n\nsecret body",
	})
	require.NoError(t, err)
	data, ok := created.Data.(map[string]any)
	require.True(t, ok)
	slug, ok := data["slug"].(string)
	require.True(t, ok)
	require.NotEmpty(t, slug)

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errIs   error
	}{
		{name: "loads existing clip", params: map[string]any{"slug": slug}},
		{name: "missing slug", params: map[string]any{}, wantErr: true, errIs: types.ErrInvalidArgument},
		{name: "unknown slug", params: map[string]any{"slug": "missing1"}, wantErr: true, errIs: types.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := capability.Invoke(context.Background(), hub.CapClip, OpGet, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			got, ok := res.Data.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, slug, got["slug"])
			assert.Contains(t, got["content"], "secret body")
			assert.Equal(t, "/c/"+slug, got["url"])
		})
	}
}

func TestParseMetaJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		want    Meta
		wantErr bool
	}{
		{
			name: "plain json",
			raw:  `{"title":"A","description":"B"}`,
			want: Meta{Title: "A", Description: "B"},
		},
		{
			name: "fenced json",
			raw:  "```json\n{\"title\":\"T\",\"description\":\"D\"}\n```",
			want: Meta{Title: "T", Description: "D"},
		},
		{
			name:    "invalid json",
			raw:     "not-json",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseMetaJSON(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWordCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{name: "empty", content: "", want: 0},
		{name: "simple", content: "one two three", want: 3},
		{name: "multiline", content: "hello\nworld", want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WordCount(tt.content))
		})
	}
}

func TestNewSlug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{name: "eight chars", n: 8},
		{name: "zero length", n: 0, wantErr: true},
		{name: "negative", n: -1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := newSlug(tt.n)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.n)
			for _, ch := range got {
				assert.Contains(t, slugAlphabet, string(ch))
			}
		})
	}
}

func TestGenerateMetaWithLLMUsesResolver(t *testing.T) {
	t.Parallel()
	// Ensure the function signature stays injectable; empty chat model path returns error.
	_, err := generateMetaWithLLM(context.Background(), "x", func(context.Context, string) (llms.Model, string, error) {
		return nil, "", errors.New("unused")
	})
	require.Error(t, err)
}
