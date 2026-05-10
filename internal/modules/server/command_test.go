package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestCommandRules_Count(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "nine command rules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, commandRules, 9)
		})
	}
}

func TestCommandRules_Defines(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all expected commands defined with help"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}

			assert.Contains(t, defines, "version")
			assert.Contains(t, defines, "mem stats")
			assert.Contains(t, defines, "golang stats")
			assert.Contains(t, defines, "server stats")
			assert.Contains(t, defines, "online stats")
			assert.Contains(t, defines, "adguard status")
			assert.Contains(t, defines, "adguard stats")
			assert.Contains(t, defines, "queue stats")
			assert.Contains(t, defines, "check")
		})
	}
}

func TestCommandRules_Handlers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all command rules have handlers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		})
	}
}

func TestCommandRules_TokenParsing(t *testing.T) {
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{
			name:   "version exact match",
			define: "version",
			input:  "version",
			want:   true,
		},
		{
			name:   "mem stats exact match",
			define: "mem stats",
			input:  "mem stats",
			want:   true,
		},
		{
			name:   "golang stats exact match",
			define: "golang stats",
			input:  "golang stats",
			want:   true,
		},
		{
			name:   "server stats exact match",
			define: "server stats",
			input:  "server stats",
			want:   true,
		},
		{
			name:   "online stats exact match",
			define: "online stats",
			input:  "online stats",
			want:   true,
		},
		{
			name:   "adguard status exact match",
			define: "adguard status",
			input:  "adguard status",
			want:   true,
		},
		{
			name:   "adguard stats exact match",
			define: "adguard stats",
			input:  "adguard stats",
			want:   true,
		},
		{
			name:   "queue stats exact match",
			define: "queue stats",
			input:  "queue stats",
			want:   true,
		},
		{
			name:   "check exact match",
			define: "check",
			input:  "check",
			want:   true,
		},
		{
			name:   "version mismatched with mem stats",
			define: "version",
			input:  "mem stats",
			want:   false,
		},
		{
			name:   "mem stats mismatched with version",
			define: "mem stats",
			input:  "version",
			want:   false,
		},
		{
			name:   "version with extra tokens fails",
			define: "version",
			input:  "version extra",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parser.ParseString(tt.input)
			require.NoError(t, err)

			check, err := parser.SyntaxCheck(tt.define, tokens)
			require.NoError(t, err)
			assert.Equal(t, tt.want, check)
		})
	}
}

func TestCommandRules_ProcessCommand_Unknown(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "unknown command returns nil result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := command.Ruleset(commandRules)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

			result, err := rs.ProcessCommand(ctx, "unknown command xyz")
			require.NoError(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestCommandRules_VersionHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "version handler returns non-empty text message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var versionRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "version" {
					versionRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, versionRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("version")

			payload := versionRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Text)
		})
	}
}

func TestCommandRules_MemStatsHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "mem stats handler returns info message with non-empty title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "mem stats" {
					memRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, memRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("mem stats")

			payload := memRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.InfoMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Title)
		})
	}
}

func TestCommandRules_GolangStatsHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "golang stats handler returns info message with non-empty title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var golangRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "golang stats" {
					golangRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, golangRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("golang stats")

			payload := golangRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.InfoMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Title)
		})
	}
}

func TestCommandRules_ServerStatsHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "server stats handler returns text or info message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "server stats" {
					serverRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, serverRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("server stats")

			payload := serverRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Contains(t, []string{"TextMsg", "InfoMsg"}, msgType)
		})
	}
}

func TestCommandRules_OnlineStatsHandler(t *testing.T) {
	t.Skip("requires redis service")

	tests := []struct {
		name string
	}{
		{name: "online stats handler returns text or kv message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var onlineRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "online stats" {
					onlineRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, onlineRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("online stats")

			payload := onlineRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Contains(t, []string{"TextMsg", "KVMsg"}, msgType)
		})
	}
}

func TestCommandRules_AdguardStatusHandler(t *testing.T) {
	t.Skip("requires adguard service")

	tests := []struct {
		name string
	}{
		{name: "adguard status handler returns non-empty text message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var adguardRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "adguard status" {
					adguardRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, adguardRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("adguard status")

			payload := adguardRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Text)
		})
	}
}

func TestCommandRules_QueueStatsHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "queue stats handler returns link message with queue url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var queueRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "queue stats" {
					queueRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, queueRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("queue stats")

			payload := queueRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.LinkMsg)
			require.True(t, ok)
			assert.Contains(t, msg.Url, "/queue/stats")
		})
	}
}

func TestCommandRules_CheckHandler(t *testing.T) {
	t.Skip("requires external services")

	tests := []struct {
		name string
	}{
		{name: "check handler returns non-empty text message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var checkRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "check" {
					checkRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, checkRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("check")

			payload := checkRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Text)
		})
	}
}
