package github

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
			name: "should have exactly 7 command rules",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, commandRules, 7)
			},
		},
		{
			name: "should contain all expected defines",
			test: func(t *testing.T) {
				t.Parallel()
				defines := make(map[string]string)
				for _, r := range commandRules {
					defines[r.Define] = r.Help
				}

				assert.Contains(t, defines, "github setting")
				assert.Contains(t, defines, "github oauth")
				assert.Contains(t, defines, "github user")
				assert.Contains(t, defines, "github card [string]")
				assert.Contains(t, defines, "github repo [string]")
				assert.Contains(t, defines, "github user [string]")
				assert.Contains(t, defines, "deploy")
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
		{name: "github setting exact match", define: "github setting", input: "github setting", want: true},
		{name: "github oauth exact match", define: "github oauth", input: "github oauth", want: true},
		{name: "github user exact match", define: "github user", input: "github user", want: true},
		{name: "github card with param", define: "github card [string]", input: "github card [text]", want: true},
		{name: "github repo with param", define: "github repo [string]", input: "github repo [owner/repo]", want: true},
		{name: "github user with param", define: "github user [string]", input: "github user [username]", want: true},
		{name: "deploy exact match", define: "deploy", input: "deploy", want: true},
		{name: "github setting does not match github oauth", define: "github setting", input: "github oauth", want: false},
		{name: "github oauth does not match github setting", define: "github oauth", input: "github setting", want: false},
		{name: "github user with extra tokens", define: "github user", input: "github user extra", want: false},
		{name: "deploy with extra tokens", define: "deploy", input: "deploy extra", want: false},
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

func TestCommandRules_GithubSettingHandler(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "github setting handler should return LinkMsg or TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var settingRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "github setting" {
					settingRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, settingRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("github setting")

			payload := settingRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Contains(t, []string{"LinkMsg", "TextMsg"}, msgType)
		})
	}
}

func TestCommandRules_GithubOAuthHandler_Unauthorized(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "github oauth handler should return LinkMsg or authorized TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var oauthRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "github oauth" {
					oauthRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, oauthRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("github oauth")

			payload := oauthRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			if msgType == "LinkMsg" {
				msg, ok := payload.(types.LinkMsg)
				require.True(t, ok)
				assert.Contains(t, msg.Url, "github.com")
			} else {
				msg, ok := payload.(types.TextMsg)
				require.True(t, ok)
				assert.Contains(t, msg.Text, "authorized")
			}
		})
	}
}

func TestCommandRules_GithubUserHandler_Unauthorized(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "github user handler should return InfoMsg or unauthorized TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var userRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "github user" {
					userRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, userRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("github user")

			payload := userRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			if msgType == "InfoMsg" {
				msg, ok := payload.(types.InfoMsg)
				require.True(t, ok)
				assert.NotEmpty(t, msg.Title)
			} else {
				msg, ok := payload.(types.TextMsg)
				require.True(t, ok)
				assert.Contains(t, msg.Text, "unauthorized")
			}
		})
	}
}

func TestCommandRules_GithubCardHandler(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "github card handler should return TextMsg or EmptyMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var cardRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "github card [string]" {
					cardRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, cardRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("github card [some card]")

			payload := cardRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Contains(t, []string{"TextMsg", "EmptyMsg"}, msgType)
		})
	}
}

func TestCommandRules_GithubRepoHandler(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "github repo handler should return TextMsg or KVMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var repoRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "github repo [string]" {
					repoRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, repoRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("github repo [owner/repo]")

			payload := repoRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Contains(t, []string{"TextMsg", "KVMsg"}, msgType)
		})
	}
}

func TestCommandRules_GithubUserStrHandler(t *testing.T) {
	t.Skip("requires database connection")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "github user [string] handler should return TextMsg or InfoMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var userStrRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "github user [string]" {
					userStrRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, userStrRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("github user [username]")

			payload := userStrRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Contains(t, []string{"TextMsg", "InfoMsg"}, msgType)
		})
	}
}

func TestCommandRules_DeployHandler(t *testing.T) {
	t.Skip("requires external service")

	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "deploy handler should return non-empty TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var deployRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "deploy" {
					deployRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, deployRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("deploy")

			payload := deployRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.NotEmpty(t, msg.Text)
		})
	}
}
