package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestParseSchema(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "valid slack URL", input: "slack://hooks.slack.com/services/xxx", expect: "slack"},
		{name: "discord bot URL", input: "discord-bot://webhook/xxx", expect: "discord-bot"},
		{name: "plain text no scheme", input: "plain text", expect: ""},
		{name: "empty string", input: "", expect: ""},
		{name: "https URL", input: "https://example.com", expect: "https"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scheme, err := ParseSchema(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, scheme)
		})
	}
}

func TestParseTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		templates []string
		expect    types.KV
	}{
		{
			name:      "single template match",
			input:     "slack://general/abc123",
			templates: []string{"slack://{channel}/{token}"},
			expect:    types.KV{"channel": "general", "token": "abc123"},
		},
		{
			name:      "no match",
			input:     "https://other.com/path",
			templates: []string{"slack://{channel}/{token}"},
			expect:    types.KV{},
		},
		{
			name:      "multiple templates picks first match",
			input:     "slack://general/abc123",
			templates: []string{"discord://{channel}/{token}", "slack://{channel}/{token}"},
			expect:    types.KV{"channel": "general", "token": "abc123"},
		},
		{
			name:      "empty templates",
			input:     "slack://general/abc123",
			templates: nil,
			expect:    types.KV{},
		},
		{
			name:      "empty input",
			input:     "",
			templates: []string{"slack://{channel}"},
			expect:    types.KV{},
		},
		{
			name:      "dashed keys",
			input:     "pushover://ukey123/atoken",
			templates: []string{"pushover://{user_key}/{app_token}"},
			expect:    types.KV{"user_key": "ukey123", "app_token": "atoken"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ParseTemplate(tt.input, tt.templates)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestPriorityConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		priority Priority
		expected Priority
	}{
		{name: "Low", priority: Low, expected: Priority(1)},
		{name: "Moderate", priority: Moderate, expected: Priority(2)},
		{name: "Normal", priority: Normal, expected: Priority(3)},
		{name: "High", priority: High, expected: Priority(4)},
		{name: "Emergency", priority: Emergency, expected: Priority(5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.priority)
		})
	}
}

func TestMessageZeroValue(t *testing.T) {
	t.Parallel()
	t.Run("all fields zero/empty", func(t *testing.T) {
		t.Parallel()
		m := Message{}
		assert.Empty(t, m.Title)
		assert.Empty(t, m.Body)
		assert.Empty(t, m.Url)
		assert.Equal(t, Priority(0), m.Priority)
	})
}
