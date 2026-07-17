package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/parser"
	providergithub "github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/flowline-io/flowbot/version"
)

func TestCommandRules_Metadata(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should contain all expected hub defines",
			test: func(t *testing.T) {
				defines := make(map[string]string)
				for _, r := range commandRules {
					defines[r.Define] = r.Help
				}

				assert.NotEmpty(t, commandRules)
				assert.Contains(t, defines, "hub health")
				assert.Contains(t, defines, "hub apps")
				assert.Contains(t, defines, "hub app [name]")
				assert.Contains(t, defines, "hub capabilities")
				assert.Contains(t, defines, "version")
				assert.Contains(t, defines, "hub app start [name]")
				assert.Contains(t, defines, "hub app stop [name]")
				assert.Contains(t, defines, "hub app restart [name]")
				assert.Contains(t, defines, "kanban status")
				assert.Equal(t, "Show kanban status", defines["kanban status"])
				assert.Contains(t, defines, "reader")
				assert.Equal(t, "show reader id", defines["reader"])
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
				for _, r := range commandRules {
					assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
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
		{name: "hub health exact match", define: "hub health", input: "hub health", want: true},
		{name: "hub apps exact match", define: "hub apps", input: "hub apps", want: true},
		{name: "hub app with name param", define: "hub app [name]", input: "hub app archivebox", want: true},
		{name: "hub capabilities exact match", define: "hub capabilities", input: "hub capabilities", want: true},
		{name: "version exact match", define: "version", input: "version", want: true},
		{name: "hub app start with name param", define: "hub app start [name]", input: "hub app start archivebox", want: true},
		{name: "hub app stop with name param", define: "hub app stop [name]", input: "hub app stop archivebox", want: true},
		{name: "hub app restart with name param", define: "hub app restart [name]", input: "hub app restart archivebox", want: true},
		{name: "hub health does not match hub apps", define: "hub health", input: "hub apps", want: false},
		{name: "hub apps does not match hub health", define: "hub apps", input: "hub health", want: false},
		{name: "hub app name with wrong define", define: "hub app [name]", input: "hub apps archivebox", want: false},
		{name: "hub app start with wrong prefix", define: "hub app start [name]", input: "hub start archivebox", want: false},
		{name: "kanban status exact match", define: "kanban status", input: "kanban status", want: true},
		{name: "kanban status with extra tokens", define: "kanban status", input: "kanban status extra", want: false},
		{name: "kanban partial match fails", define: "kanban status", input: "kanban", want: false},
		{name: "reader exact match", define: "reader", input: "reader", want: true},
		{name: "reader with extra tokens fails", define: "reader", input: "reader extra", want: false},
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
		{name: "unknown command should return nil result"},
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

func TestHubHealthHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "hub health handler returns InfoMsg with title and model"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldHubList := hub.Default.List()
			oldHlList := homelab.DefaultRegistry.List()
			defer func() {
				for _, d := range oldHubList {
					_ = hub.Default.Register(d)
				}
				homelab.DefaultRegistry.Replace(oldHlList)
			}()

			require.NoError(t, hub.Default.Register(hub.Descriptor{
				Type: hub.CapKarakeep, App: "karakeep",
				Instance: "ok", Healthy: true,
			}))

			var healthRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "hub health" {
					healthRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, healthRule)

			tokens, _ := parser.ParseString("hub health")
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			payload := healthRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.InfoMsg)
			require.True(t, ok)
			assert.Equal(t, "Hub Health", msg.Title)
			assert.NotNil(t, msg.Model)
		})
	}
}

func TestHubAppsHandler(t *testing.T) {
	tests := []struct {
		name      string
		apps      []homelab.App
		wantTitle string
		wantText  string
		isTextMsg bool
	}{
		{
			name: "with registered apps returns InfoMsg",
			apps: []homelab.App{
				{Name: "archivebox", Status: homelab.AppStatusRunning, Health: homelab.HealthHealthy},
				{Name: "karakeep", Status: homelab.AppStatusStopped, Health: homelab.HealthUnhealthy},
			},
			wantTitle: "Homelab Apps",
			isTextMsg: false,
		},
		{
			name:      "empty registry returns TextMsg",
			apps:      []homelab.App{},
			wantText:  "No apps registered",
			isTextMsg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldList := homelab.DefaultRegistry.List()
			defer homelab.DefaultRegistry.Replace(oldList)

			homelab.DefaultRegistry.Replace(tt.apps)

			var appsRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "hub apps" {
					appsRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, appsRule)

			tokens, _ := parser.ParseString("hub apps")
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			payload := appsRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			if tt.isTextMsg {
				msg, ok := payload.(types.TextMsg)
				require.True(t, ok)
				assert.Equal(t, tt.wantText, msg.Text)
			} else {
				msg, ok := payload.(types.InfoMsg)
				require.True(t, ok)
				assert.Equal(t, tt.wantTitle, msg.Title)
			}
		})
	}
}

func TestHubAppHandler(t *testing.T) {
	tests := []struct {
		name      string
		appName   string
		apps      []homelab.App
		wantTitle string
		wantText  string
		isTextMsg bool
	}{
		{
			name:    "existing app returns InfoMsg",
			appName: "archivebox",
			apps: []homelab.App{
				{Name: "archivebox", Path: "/apps/archivebox", Status: homelab.AppStatusRunning},
			},
			wantTitle: "App: archivebox",
			isTextMsg: false,
		},
		{
			name:      "nonexistent app returns not found TextMsg",
			appName:   "nonexistent",
			apps:      []homelab.App{},
			wantText:  "not found",
			isTextMsg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldList := homelab.DefaultRegistry.List()
			defer homelab.DefaultRegistry.Replace(oldList)

			homelab.DefaultRegistry.Replace(tt.apps)

			var appRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "hub app [name]" {
					appRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, appRule)

			tokens, _ := parser.ParseString("hub app " + tt.appName)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			payload := appRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			if tt.isTextMsg {
				msg, ok := payload.(types.TextMsg)
				require.True(t, ok)
				assert.Contains(t, msg.Text, tt.wantText)
			} else {
				msg, ok := payload.(types.InfoMsg)
				require.True(t, ok)
				assert.Equal(t, tt.wantTitle, msg.Title)
			}
		})
	}
}

func TestVersionHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "version handler returns InfoMsg with version fields"},
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

			tokens, _ := parser.ParseString("version")
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			payload := versionRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.InfoMsg)
			require.True(t, ok)
			assert.Equal(t, "Flowbot Version", msg.Title)

			model, ok := msg.Model.(types.KV)
			require.True(t, ok)
			assert.Equal(t, version.Buildtags, model["Version"])
			assert.Equal(t, version.Buildstamp, model["Build"])
		})
	}
}

func TestHubCapabilitiesHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "hub capabilities handler returns InfoMsg with title"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldList := hub.Default.List()
			defer func() {
				for _, d := range oldList {
					_ = hub.Default.Register(d)
				}
			}()

			require.NoError(t, hub.Default.Register(hub.Descriptor{
				Type: hub.CapKarakeep, App: "karakeep", Healthy: true,
			}))
			require.NoError(t, hub.Default.Register(hub.Descriptor{
				Type: hub.CapExample, App: "example", Healthy: false,
			}))

			var capRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "hub capabilities" {
					capRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, capRule)

			tokens, _ := parser.ParseString("hub capabilities")
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			payload := capRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.InfoMsg)
			require.True(t, ok)
			assert.Equal(t, "Hub Capabilities", msg.Title)
		})
	}
}

func TestHubAppStartHandler_PermissionDenied(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "start denied when permissions disable start"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldList := homelab.DefaultRegistry.List()
			oldPerm := homelab.DefaultRegistry.Permissions()
			defer func() {
				homelab.DefaultRegistry.Replace(oldList)
				homelab.DefaultRegistry.SetPermissions(oldPerm)
			}()

			homelab.DefaultRegistry.Replace([]homelab.App{
				{Name: "archivebox", Path: "/apps/archivebox"},
			})
			homelab.DefaultRegistry.SetPermissions(homelab.Permissions{Start: false})

			var startRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "hub app start [name]" {
					startRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, startRule)

			tokens, _ := parser.ParseString("hub app start archivebox")
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			payload := startRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Contains(t, msg.Text, "start not allowed")
		})
	}
}

func TestHubAppStopHandler_NotFound(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "stopping nonexistent app returns not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldList := homelab.DefaultRegistry.List()
			defer homelab.DefaultRegistry.Replace(oldList)

			homelab.DefaultRegistry.Replace([]homelab.App{})

			var stopRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "hub app stop [name]" {
					stopRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, stopRule)

			tokens, _ := parser.ParseString("hub app stop nonexistent")
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			payload := stopRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Contains(t, msg.Text, "not found")
		})
	}
}

func TestHubAppRestartHandler_PermissionDenied(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "restart denied when permissions disable restart"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldList := homelab.DefaultRegistry.List()
			oldPerm := homelab.DefaultRegistry.Permissions()
			defer func() {
				homelab.DefaultRegistry.Replace(oldList)
				homelab.DefaultRegistry.SetPermissions(oldPerm)
			}()

			homelab.DefaultRegistry.Replace([]homelab.App{
				{Name: "archivebox", Path: "/apps/archivebox"},
			})
			homelab.DefaultRegistry.SetPermissions(homelab.Permissions{Restart: false})

			var restartRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "hub app restart [name]" {
					restartRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, restartRule)

			tokens, _ := parser.ParseString("hub app restart archivebox")
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			payload := restartRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Contains(t, msg.Text, "restart not allowed")
		})
	}
}

func TestKanbanStatusHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "status handler returns empty message type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var statusRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "kanban status" {
					statusRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, statusRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("kanban status")

			payload := statusRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Equal(t, "EmptyMsg", msgType)
		})
	}
}

func TestReaderHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "reader handler returns miniflux id as text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var readerRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "reader" {
					readerRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, readerRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("reader")

			payload := readerRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Equal(t, miniflux.ID, msg.Text)
		})
	}
}

func TestGithubOAuthUsesGithubProvider(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "github oauth provider id is github not hub module name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, "github", providergithub.ID)
			assert.NotEqual(t, Name, providergithub.ID)
		})
	}
}

func TestGithubSettingHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "github setting handler should return LinkMsg or TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestGithubOAuthHandler_Unauthorized(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "github oauth handler should return LinkMsg or authorized TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestGithubUserHandler_Unauthorized(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "github user handler should return InfoMsg or unauthorized TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestGithubCardHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "github card handler should return TextMsg or EmptyMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestGithubRepoHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "github repo handler should return TextMsg or KVMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestGithubUserStrHandler(t *testing.T) {
	t.Skip("requires database connection")

	tests := []struct {
		name string
	}{
		{name: "github user [string] handler should return TextMsg or InfoMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestDeployHandler(t *testing.T) {
	t.Skip("requires external service")

	tests := []struct {
		name string
	}{
		{name: "deploy handler should return non-empty TextMsg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestFormRules_Empty(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Empty(t, formRules)
		})
	}
}

func TestCheckLifecyclePermission(t *testing.T) {
	tests := []struct {
		name   string
		perm   homelab.Permissions
		action string
		want   bool
	}{
		{name: "start action allowed", perm: homelab.Permissions{Start: true, Stop: false, Restart: true}, action: "start", want: true},
		{name: "stop action denied", perm: homelab.Permissions{Start: true, Stop: false, Restart: true}, action: "stop", want: false},
		{name: "restart action allowed", perm: homelab.Permissions{Start: true, Stop: false, Restart: true}, action: "restart", want: true},
		{name: "unknown action denied", perm: homelab.Permissions{Start: true, Stop: false, Restart: true}, action: "unknown", want: false},
		{name: "all false - start denied", perm: homelab.Permissions{}, action: "start", want: false},
		{name: "all false - stop denied", perm: homelab.Permissions{}, action: "stop", want: false},
		{name: "all false - restart denied", perm: homelab.Permissions{}, action: "restart", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, homelab.AllowsLifecycle(tt.perm, tt.action))
		})
	}
}
