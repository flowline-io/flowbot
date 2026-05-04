package template

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineLoadConfig(t *testing.T) {
	e := New()
	err := e.LoadConfig([]config.NotifyTemplate{
		{
			ID:              "test.event",
			Name:            "Test Event",
			Description:     "A test notification",
			DefaultFormat:   "markdown",
			DefaultTemplate: "**{{ .title }}**\n{{ .body }}",
		},
	})
	require.NoError(t, err)

	result, err := e.Render("test.event", "slack", map[string]any{
		"title": "Hello World",
		"body":  "This is a test",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Hello World", result.Title)
	assert.Equal(t, "**Hello World**\nThis is a test", result.Body)
	assert.Equal(t, "markdown", result.Format)
}

func TestEngineChannelOverride(t *testing.T) {
	e := New()
	err := e.LoadConfig([]config.NotifyTemplate{
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
	})
	require.NoError(t, err)

	// default channel uses default template
	result, err := e.Render("test.event", "slack", map[string]any{
		"title": "Hello",
		"body":  "World",
	})
	require.NoError(t, err)
	assert.Equal(t, "markdown", result.Format)
	assert.Contains(t, result.Body, "**Hello**")

	// telegram channel uses override
	result, err = e.Render("test.event", "telegram", map[string]any{
		"title": "Hello",
		"body":  "World",
	})
	require.NoError(t, err)
	assert.Equal(t, "html", result.Format)
	assert.Contains(t, result.Body, "<b>Hello</b>")
}

func TestEngineSpriteFunctions(t *testing.T) {
	e := New()
	err := e.LoadConfig([]config.NotifyTemplate{
		{
			ID:              "sprig.test",
			Name:            "Sprig Test",
			Description:     "Test sprig functions",
			DefaultFormat:   "markdown",
			DefaultTemplate: "{{ .name | upper }}\n{{ .count | default 0 }}\n{{ join \", \" .tags }}",
		},
	})
	require.NoError(t, err)

	result, err := e.Render("sprig.test", "slack", map[string]any{
		"name": "hello",
		"tags": []string{"a", "b", "c"},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Body, "HELLO")
	assert.Contains(t, result.Body, "0") // default
	assert.Contains(t, result.Body, "a, b, c")
}

func TestEngineMissingTemplate(t *testing.T) {
	e := New()
	result, err := e.Render("nonexistent", "slack", nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestEngineShorten(t *testing.T) {
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

	result, err := e.Render("shorten.test", "slack", map[string]any{
		"text": "this is a very long string",
	})
	require.NoError(t, err)
	assert.Equal(t, "this is...", result.Body)
}

func TestEngineShortenShort(t *testing.T) {
	e := New()
	err := e.LoadConfig([]config.NotifyTemplate{
		{
			ID:              "shorten.short",
			Name:            "Shorten Short",
			Description:     "Test shorten short",
			DefaultFormat:   "markdown",
			DefaultTemplate: "{{ shorten .text 10 }}",
		},
	})
	require.NoError(t, err)

	result, err := e.Render("shorten.short", "slack", map[string]any{
		"text": "hi",
	})
	require.NoError(t, err)
	assert.Equal(t, "hi", result.Body)
}

func TestEngineExtractTitle(t *testing.T) {
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
}

func TestEngineLoadConfigError(t *testing.T) {
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
}

func TestEngineGetTemplateID(t *testing.T) {
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

	assert.Equal(t, "bookmark.created", e.GetTemplateID("bookmark.created"))
	assert.Equal(t, "", e.GetTemplateID("nonexistent"))
}

func TestRenderResultNilOnMissingTemplate(t *testing.T) {
	e := New()
	result, err := e.Render("does.not.exist", "slack", map[string]any{})
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestEngineConditionalTemplate(t *testing.T) {
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

	result, err = e.Render("conditional.test", "slack", map[string]any{
		"title":  "Task",
		"urgent": false,
	})
	require.NoError(t, err)
	assert.Equal(t, "Task", result.Body)
}
