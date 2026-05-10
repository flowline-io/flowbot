package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
)

func TestEngineRender(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		templates  []config.NotifyTemplate
		templateID string
		channel    string
		data       map[string]any
		wantTitle  string
		wantBody   string
		wantFormat string
		wantNil    bool
	}{
		{
			name: "basic markdown template",
			templates: []config.NotifyTemplate{
				{
					ID:              "test.event",
					Name:            "Test Event",
					Description:     "A test notification",
					DefaultFormat:   "markdown",
					DefaultTemplate: "**{{ .title }}**\n{{ .body }}",
				},
			},
			templateID: "test.event",
			channel:    "slack",
			data: map[string]any{
				"title": "Hello World",
				"body":  "This is a test",
			},
			wantTitle:  "Hello World",
			wantBody:   "**Hello World**\nThis is a test",
			wantFormat: "markdown",
		},
		{
			name: "sprig functions",
			templates: []config.NotifyTemplate{
				{
					ID:              "sprig.test",
					Name:            "Sprig Test",
					Description:     "Test sprig functions",
					DefaultFormat:   "markdown",
					DefaultTemplate: "{{ .name | upper }}\n{{ .count | default 0 }}\n{{ join \", \" .tags }}",
				},
			},
			templateID: "sprig.test",
			channel:    "slack",
			data: map[string]any{
				"name": "hello",
				"tags": []string{"a", "b", "c"},
			},
			wantTitle:  "HELLO",
			wantBody:   "HELLO\n0\na, b, c",
			wantFormat: "markdown",
		},
		{
			name: "title extraction from markdown heading",
			templates: []config.NotifyTemplate{
				{
					ID:              "title.test",
					Name:            "Title Test",
					Description:     "Test title extraction",
					DefaultFormat:   "markdown",
					DefaultTemplate: "# My Title\n\nBody content here",
				},
			},
			templateID: "title.test",
			channel:    "slack",
			data:       nil,
			wantTitle:  "My Title",
			wantBody:   "# My Title\n\nBody content here",
			wantFormat: "markdown",
		},
		{
			name: "conditional template urgent true",
			templates: []config.NotifyTemplate{
				{
					ID:              "conditional.test",
					Name:            "Conditional Test",
					DefaultFormat:   "markdown",
					DefaultTemplate: "{{ if .urgent }}URGENT: {{ end }}{{ .title }}",
				},
			},
			templateID: "conditional.test",
			channel:    "slack",
			data: map[string]any{
				"title":  "Task",
				"urgent": true,
			},
			wantTitle:  "URGENT: Task",
			wantFormat: "markdown",
		},
		{
			name: "conditional template urgent false",
			templates: []config.NotifyTemplate{
				{
					ID:              "conditional.test",
					Name:            "Conditional Test",
					DefaultFormat:   "markdown",
					DefaultTemplate: "{{ if .urgent }}URGENT: {{ end }}{{ .title }}",
				},
			},
			templateID: "conditional.test",
			channel:    "slack",
			data: map[string]any{
				"title":  "Task",
				"urgent": false,
			},
			wantTitle:  "Task",
			wantBody:   "Task",
			wantFormat: "markdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			err := e.LoadConfig(tt.templates)
			require.NoError(t, err)

			result, err := e.Render(tt.templateID, tt.channel, tt.data)
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.wantTitle != "" {
				assert.Equal(t, tt.wantTitle, result.Title)
			}
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, result.Body)
			}
			if tt.wantFormat != "" {
				assert.Equal(t, tt.wantFormat, result.Format)
			}
		})
	}
}

func TestEngineChannelOverride(t *testing.T) {
	t.Parallel()
	templates := []config.NotifyTemplate{
		{
			ID:              "test.event",
			Name:            "Test Event",
			Description:     "A test notification",
			DefaultFormat:   "markdown",
			DefaultTemplate: "**{{ .title }}**\n{{ .body }}",
			Overrides: []config.NotifyOverride{
				{
					Channel:  "telegram",
					Format:   "html",
					Template: "<b>{{ .title }}</b>\n{{ .body }}",
				},
			},
		},
	}

	tests := []struct {
		name        string
		channel     string
		wantFormat  string
		wantContain string
	}{
		{
			name:        "default channel uses default template",
			channel:     "slack",
			wantFormat:  "markdown",
			wantContain: "**Hello**",
		},
		{
			name:        "telegram channel uses override",
			channel:     "telegram",
			wantFormat:  "html",
			wantContain: "<b>Hello</b>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			err := e.LoadConfig(templates)
			require.NoError(t, err)

			result, err := e.Render("test.event", tt.channel, map[string]any{
				"title": "Hello",
				"body":  "World",
			})
			require.NoError(t, err)
			assert.Equal(t, tt.wantFormat, result.Format)
			assert.Contains(t, result.Body, tt.wantContain)
		})
	}
}

func TestEngineShorten(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		text     string
		wantBody string
	}{
		{
			name:     "long text is truncated",
			text:     "this is a very long string",
			wantBody: "this is...",
		},
		{
			name:     "short text is unchanged",
			text:     "hi",
			wantBody: "hi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			err := e.LoadConfig([]config.NotifyTemplate{
				{
					ID:              "shorten.test",
					Name:            "Shorten Test",
					Description:     "Test shorten",
					DefaultFormat:   "markdown",
					DefaultTemplate: "{{ shorten .text 10 }}",
				},
			})
			require.NoError(t, err)

			result, err := e.Render("shorten.test", "slack", map[string]any{"text": tt.text})
			require.NoError(t, err)
			assert.Equal(t, tt.wantBody, result.Body)
		})
	}
}

func TestEngineMissingTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		templateID string
		data       map[string]any
	}{
		{
			name:       "nonexistent template returns nil",
			templateID: "nonexistent",
			data:       nil,
		},
		{
			name:       "does not exist returns nil",
			templateID: "does.not.exist",
			data:       map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			result, err := e.Render(tt.templateID, "slack", tt.data)
		require.NoError(t, err)
		assert.Nil(t, result)
		})
	}
}

func TestEngineLoadConfigError(t *testing.T) {
	t.Run("bad template parse error", func(t *testing.T) {
		t.Parallel()
		e := New()
		err := e.LoadConfig([]config.NotifyTemplate{
			{
				ID:              "bad.template",
				Name:            "Bad Template",
				Description:     "A bad template",
				DefaultFormat:   "markdown",
				DefaultTemplate: "{{ .nonexistent | invalid_func }}",
			},
		})
		assert.Error(t, err)
	})
}

func TestEngineGetTemplateID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventType string
		wantEmpty bool
	}{
		{name: "existing template", eventType: "bookmark.created", wantEmpty: false},
		{name: "nonexistent template", eventType: "nonexistent", wantEmpty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			err := e.LoadConfig([]config.NotifyTemplate{
				{
					ID:              "bookmark.created",
					Name:            "Bookmark Created",
					DefaultFormat:   "markdown",
					DefaultTemplate: "test",
				},
			})
			require.NoError(t, err)

			got := e.GetTemplateID(tt.eventType)
			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.eventType, got)
			}
		})
	}
}

func TestEngineConditionalBodyPrefix(t *testing.T) {
	t.Run("urgent flag prefixes body with URGENT", func(t *testing.T) {
		t.Parallel()
		e := New()
		err := e.LoadConfig([]config.NotifyTemplate{
			{
				ID:              "conditional.test",
				Name:            "Conditional Test",
				DefaultFormat:   "markdown",
				DefaultTemplate: "{{ if .urgent }}URGENT: {{ end }}{{ .title }}",
			},
		})
		require.NoError(t, err)

		result, err := e.Render("conditional.test", "slack", map[string]any{
			"title":  "Task",
			"urgent": true,
		})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(result.Body, "URGENT: "))
	})
}

func TestEngineRenderExtractTitle(t *testing.T) {
	t.Run("markdown heading becomes plain title", func(t *testing.T) {
		t.Parallel()
		e := New()
		err := e.LoadConfig([]config.NotifyTemplate{
			{
				ID:              "title.test",
				Name:            "Title Test",
				Description:     "Test title extraction",
				DefaultFormat:   "markdown",
				DefaultTemplate: "# My Title\n\nBody content here",
			},
		})
		require.NoError(t, err)

		result, err := e.Render("title.test", "slack", nil)
		require.NoError(t, err)
		assert.Equal(t, "My Title", result.Title)
		assert.Contains(t, result.Body, "Body content here")
	})
}
