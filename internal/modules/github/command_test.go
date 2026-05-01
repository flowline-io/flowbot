package github

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 8)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "github setting")
	assert.Contains(t, defines, "github oauth")
	assert.Contains(t, defines, "github user")
	assert.Contains(t, defines, "github issue [string]")
	assert.Contains(t, defines, "github card [string]")
	assert.Contains(t, defines, "github repo [string]")
	assert.Contains(t, defines, "github user [string]")
	assert.Contains(t, defines, "deploy")
}

func TestCommandRules_Handlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}

func TestCommandRules_TokenParsing(t *testing.T) {
	tests := []struct {
		define string
		input  string
		want   bool
	}{
		{"github setting", "github setting", true},
		{"github oauth", "github oauth", true},
		{"github user", "github user", true},
		{"github issue [string]", "github issue [title]", true},
		{"github card [string]", "github card [text]", true},
		{"github repo [string]", "github repo [owner/repo]", true},
		{"github user [string]", "github user [username]", true},
		{"deploy", "deploy", true},
		{"github setting", "github oauth", false},
		{"github oauth", "github setting", false},
		{"github user", "github user extra", false},
		{"deploy", "deploy extra", false},
	}

	for _, tt := range tests {
		t.Run(tt.define+"_"+tt.input, func(t *testing.T) {
			tokens, err := parser.ParseString(tt.input)
			require.NoError(t, err)

			check, err := parser.SyntaxCheck(tt.define, tokens)
			require.NoError(t, err)
			assert.Equal(t, tt.want, check)
		})
	}
}

func TestCommandRules_ProcessCommand_Unknown(t *testing.T) {
	rs := command.Ruleset(commandRules)
	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

	result, err := rs.ProcessCommand(ctx, "unknown command xyz")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCommandRules_GithubSettingHandler(t *testing.T) {
	t.Skip("requires database connection")

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
}

func TestCommandRules_GithubOAuthHandler_Unauthorized(t *testing.T) {
	t.Skip("requires database connection")

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
}

func TestCommandRules_GithubUserHandler_Unauthorized(t *testing.T) {
	t.Skip("requires database connection")

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
}

func TestCommandRules_GithubIssueHandler(t *testing.T) {
	t.Skip("requires database connection")

	var issueRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "github issue [string]" {
			issueRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, issueRule)

	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	tokens, _ := parser.ParseString("github issue [some issue]")

	payload := issueRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msgType := types.TypeOf(payload)
	assert.Contains(t, []string{"TextMsg", "LinkMsg"}, msgType)
}

func TestCommandRules_GithubCardHandler(t *testing.T) {
	t.Skip("requires database connection")

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
}

func TestCommandRules_GithubRepoHandler(t *testing.T) {
	t.Skip("requires database connection")

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
}

func TestCommandRules_GithubUserStrHandler(t *testing.T) {
	t.Skip("requires database connection")

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
}

func TestCommandRules_DeployHandler(t *testing.T) {
	t.Skip("requires external service")

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
}
