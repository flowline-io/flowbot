package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/notify/manifest"
)

func TestEngineRender(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		templates  []manifest.Template
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
			templates: []manifest.Template{
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
			templates: []manifest.Template{
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
			templates: []manifest.Template{
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
			templates: []manifest.Template{
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
			templates: []manifest.Template{
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
	templates := []manifest.Template{
		{
			ID:              "test.event",
			Name:            "Test Event",
			Description:     "A test notification",
			DefaultFormat:   "markdown",
			DefaultTemplate: "**{{ .title }}**\n{{ .body }}",
			Overrides: []manifest.Override{
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
			err := e.LoadConfig([]manifest.Template{
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
		err := e.LoadConfig([]manifest.Template{
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
			err := e.LoadConfig([]manifest.Template{
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

func TestEngineListAndHasTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		templates   []manifest.Template
		checkID     string
		wantHas     bool
		wantIDCount int
	}{
		{
			name: "lists loaded templates",
			templates: []manifest.Template{
				{ID: "a.created", Name: "A", DefaultFormat: "markdown", DefaultTemplate: "a"},
				{ID: "b.created", Name: "B", DefaultFormat: "markdown", DefaultTemplate: "b"},
			},
			checkID:     "a.created",
			wantHas:     true,
			wantIDCount: 2,
		},
		{
			name: "missing template returns false",
			templates: []manifest.Template{
				{ID: "a.created", Name: "A", DefaultFormat: "markdown", DefaultTemplate: "a"},
			},
			checkID:     "missing",
			wantHas:     false,
			wantIDCount: 1,
		},
		{
			name:        "empty engine",
			templates:   nil,
			checkID:     "any",
			wantHas:     false,
			wantIDCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			require.NoError(t, e.LoadConfig(tt.templates))
			assert.Equal(t, tt.wantHas, e.HasTemplate(tt.checkID))
			assert.Len(t, e.ListTemplateIDs(), tt.wantIDCount)
		})
	}
}

func TestEngineListTemplates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		templates []manifest.Template
		wantIDs   []string
	}{
		{
			name: "returns manifests sorted by id",
			templates: []manifest.Template{
				{ID: "z.last", Name: "Z", DefaultFormat: "markdown", DefaultTemplate: "z"},
				{ID: "a.first", Name: "A", Description: "first", DefaultFormat: "html", DefaultTemplate: "a"},
			},
			wantIDs: []string{"a.first", "z.last"},
		},
		{
			name: "preserves overrides on listed manifests",
			templates: []manifest.Template{
				{
					ID: "with.override", Name: "O", DefaultFormat: "markdown", DefaultTemplate: "body",
					Overrides: []manifest.Override{{Channel: "telegram", Format: "html", Template: "<b>x</b>"}},
				},
			},
			wantIDs: []string{"with.override"},
		},
		{
			name:      "empty engine returns empty slice",
			templates: nil,
			wantIDs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			require.NoError(t, e.LoadConfig(tt.templates))
			got := e.ListTemplates()
			require.Len(t, got, len(tt.wantIDs))
			for i, id := range tt.wantIDs {
				assert.Equal(t, id, got[i].ID)
			}
			if len(tt.templates) == 1 && len(tt.templates[0].Overrides) > 0 {
				require.Len(t, got[0].Overrides, 1)
				assert.Equal(t, "telegram", got[0].Overrides[0].Channel)
			}
		})
	}
}

func TestRenderString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tmpl      string
		format    string
		data      map[string]any
		wantTitle string
		wantBody  string
		wantErr   bool
	}{
		{
			name:      "renders markdown with sprig",
			tmpl:      "**Hello {{ .name | upper }}**\nBody line",
			format:    "markdown",
			data:      map[string]any{"name": "world"},
			wantTitle: "Hello WORLD",
			wantBody:  "**Hello WORLD**\nBody line",
		},
		{
			name:      "defaults empty format to markdown",
			tmpl:      "Only title",
			format:    "",
			data:      nil,
			wantTitle: "Only title",
			wantBody:  "Only title",
		},
		{
			name:    "invalid template returns error",
			tmpl:    "{{ .name ",
			format:  "markdown",
			data:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RenderString(tt.tmpl, tt.format, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantTitle, got.Title)
			assert.Equal(t, tt.wantBody, got.Body)
			if tt.format == "" {
				assert.Equal(t, "markdown", got.Format)
			} else {
				assert.Equal(t, tt.format, got.Format)
			}
		})
	}
}

func TestEventTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{name: "string time passes through", input: "2026-07-16T00:00:00Z", want: "2026-07-16T00:00:00Z"},
		{name: "non-string returns empty", input: 123, want: ""},
		{name: "nil returns empty", input: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, eventTime(tt.input))
		})
	}
}

func TestEngineConditionalBodyPrefix(t *testing.T) {
	t.Run("urgent flag prefixes body with URGENT", func(t *testing.T) {
		t.Parallel()
		e := New()
		err := e.LoadConfig([]manifest.Template{
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
		err := e.LoadConfig([]manifest.Template{
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
