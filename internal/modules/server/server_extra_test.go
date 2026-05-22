package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestCommandRules_HandlerContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		define    string
		input     string
		wantType  string
		wantTitle string
		wantURL   string
	}{
		{
			name:     "version handler returns text message with content",
			define:   "version",
			input:    "version",
			wantType: "TextMsg",
		},
		{
			name:      "mem stats handler returns info message with title",
			define:    "mem stats",
			input:     "mem stats",
			wantType:  "InfoMsg",
			wantTitle: "Memory stats",
		},
		{
			name:      "golang stats handler returns info message with title",
			define:    "golang stats",
			input:     "golang stats",
			wantType:  "InfoMsg",
			wantTitle: "Golang stats",
		},
		{
			name:      "queue stats handler returns link message with url and title",
			define:    "queue stats",
			input:     "queue stats",
			wantType:  "LinkMsg",
			wantTitle: "Queue Stats",
			wantURL:   "/queue/stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == tt.define {
					rule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, rule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString(tt.input)

			payload := rule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			switch tt.wantType {
			case "TextMsg":
				msg, ok := payload.(types.TextMsg)
				require.True(t, ok)
				assert.NotEmpty(t, msg.Text)
			case "InfoMsg":
				msg, ok := payload.(types.InfoMsg)
				require.True(t, ok)
				assert.Equal(t, tt.wantTitle, msg.Title)
			case "LinkMsg":
				msg, ok := payload.(types.LinkMsg)
				require.True(t, ok)
				assert.Contains(t, msg.Url, tt.wantURL)
				if tt.wantTitle != "" {
					assert.Equal(t, tt.wantTitle, msg.Title)
				}
			}
		})
	}
}

func TestWebserviceRulesEndpoints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "at least two webservice rules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.GreaterOrEqual(t, len(webserviceRules), 2)
		})
	}
}
