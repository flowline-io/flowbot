package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestCommandRules_Metadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 13 command rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, commandRules, 13)
			},
		},
		{
			name: "should contain all expected defines",
			test: func(t *testing.T) {
				t.Parallel()
				expected := []string{
					"dev setting", "id", "form test", "queue test",
					"page test", "docker test", "torrent test",
					"slash test", "llm test", "notify test", "fs test",
					"event test", "test",
				}
				defines := make(map[string]bool)
				for _, r := range commandRules {
					defines[r.Define] = true
				}
				for _, e := range expected {
					assert.True(t, defines[e], "expected define %q to exist", e)
				}
			},
		},
		{
			name: "all command rules should have non-nil handlers",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range commandRules {
					assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
				}
			},
		},
		{
			name: "all command rules should have non-empty help",
			test: func(t *testing.T) {
				t.Parallel()
				for _, r := range commandRules {
					assert.NotEmpty(t, r.Help, "help for %q should not be empty", r.Define)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestCommandRules_TokenParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{name: "id exact match", define: "id", input: "id", want: true},
		{name: "dev setting exact match", define: "dev setting", input: "dev setting", want: true},
		{name: "form test exact match", define: "form test", input: "form test", want: true},
		{name: "queue test exact match", define: "queue test", input: "queue test", want: true},
		{name: "page test exact match", define: "page test", input: "page test", want: true},
		{name: "docker test exact match", define: "docker test", input: "docker test", want: true},
		{name: "torrent test exact match", define: "torrent test", input: "torrent test", want: true},
		{name: "slash test exact match", define: "slash test", input: "slash test", want: true},
		{name: "llm test exact match", define: "llm test", input: "llm test", want: true},
		{name: "notify test exact match", define: "notify test", input: "notify test", want: true},
		{name: "fs test exact match", define: "fs test", input: "fs test", want: true},
		{name: "event test exact match", define: "event test", input: "event test", want: true},
		{name: "test exact match", define: "test", input: "test", want: true},
		{name: "id does not match dev setting", define: "id", input: "dev setting", want: false},
		{name: "dev setting does not match id", define: "dev setting", input: "id", want: false},
		{name: "form test with extra tokens", define: "form test", input: "form test extra", want: false},
		{name: "id with extra tokens", define: "id", input: "id extra", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tokens, err := parser.ParseString(tt.input)
			require.NoError(t, err)

			check, err := parser.SyntaxCheck(tt.define, tokens)
			require.NoError(t, err)
			assert.Equal(t, tt.want, check)
		})
	}
}

func TestCommandRules_IDHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should generate unique non-empty UUIDs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var idRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "id" {
					idRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, idRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("id")

			results := make(map[string]bool)
			for range 10 {
				payload := idRule.Handler(ctx, tokens)
				require.NotNil(t, payload)

				msg, ok := payload.(types.TextMsg)
				require.True(t, ok, "should be TextMsg")

				assert.NotEmpty(t, msg.Text, "ID should not be empty")
				assert.Greater(t, len(msg.Text), 10, "ID should be reasonably long")
				results[msg.Text] = true
			}

			assert.GreaterOrEqual(t, len(results), 2, "should generate different UUIDs")
		})
	}
}

func TestCommandRules_ProcessCommand_Unknown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "unknown command should return nil result"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rs := command.Ruleset(commandRules)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

			result, err := rs.ProcessCommand(ctx, "unknown command xyz")
			require.NoError(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestCommandRules_FormTestHandler(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "form test handler should return FormMsg or TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var formRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "form test" {
					formRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, formRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("form test")

			payload := formRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			if msgType == "FormMsg" {
				msg, ok := payload.(types.FormMsg)
				require.True(t, ok)
				assert.Equal(t, devFormID, msg.ID)
				assert.NotEmpty(t, msg.Title)
				assert.NotEmpty(t, msg.Field)
			} else {
				assert.Equal(t, "TextMsg", msgType)
			}
		})
	}
}

func TestCommandRules_DevSettingHandler(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "dev setting handler should return LinkMsg or TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var settingRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "dev setting" {
					settingRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, settingRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("dev setting")

			payload := settingRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			if msgType == "LinkMsg" {
				msg, ok := payload.(types.LinkMsg)
				require.True(t, ok)
				assert.Contains(t, msg.Title, "Setting")
			} else {
				assert.Equal(t, "TextMsg", msgType)
			}
		})
	}
}

func TestCommandRules_PageTestHandler(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "page test handler should return LinkMsg or TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var pageRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "page test" {
					pageRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, pageRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("page test")

			payload := pageRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			if msgType == "LinkMsg" {
				link, ok := payload.(types.LinkMsg)
				require.True(t, ok)
				assert.Contains(t, link.Url, "/page/dev/")
			} else {
				assert.Equal(t, "TextMsg", msgType)
			}
		})
	}
}

func TestCommandRules_SlashTestHandler(t *testing.T) {
	t.Skip("requires external service")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "slash test handler should return non-empty TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var slashRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "slash test" {
					slashRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, slashRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("slash test")

			payload := slashRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Text)
		})
	}
}

func TestCommandRules_LLMTestHandler(t *testing.T) {
	t.Skip("requires agent service")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "llm test handler should return non-empty TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var llmRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "llm test" {
					llmRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, llmRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("llm test")

			payload := llmRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Text)
		})
	}
}
