package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestCommandRules_Metadata(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 7 command rules",
			test: func(t *testing.T) {
				assert.Len(t, commandRules, 7)
			},
		},
		{
			name: "should contain all expected defines",
			test: func(t *testing.T) {
				defines := make(map[string]string)
				for _, r := range commandRules {
					defines[r.Define] = r.Help
				}

				assert.Contains(t, defines, "hub health")
				assert.Contains(t, defines, "hub apps")
				assert.Contains(t, defines, "hub app [name]")
				assert.Contains(t, defines, "hub capabilities")
				assert.Contains(t, defines, "hub app start [name]")
				assert.Contains(t, defines, "hub app stop [name]")
				assert.Contains(t, defines, "hub app restart [name]")
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
		t.Run(tt.name, tt.test)
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
		{name: "hub app start with name param", define: "hub app start [name]", input: "hub app start archivebox", want: true},
		{name: "hub app stop with name param", define: "hub app stop [name]", input: "hub app stop archivebox", want: true},
		{name: "hub app restart with name param", define: "hub app restart [name]", input: "hub app restart archivebox", want: true},
		{name: "hub health does not match hub apps", define: "hub health", input: "hub apps", want: false},
		{name: "hub apps does not match hub health", define: "hub apps", input: "hub health", want: false},
		{name: "hub app name with wrong define", define: "hub app [name]", input: "hub apps archivebox", want: false},
		{name: "hub app start with wrong prefix", define: "hub app start [name]", input: "hub start archivebox", want: false},
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
				Type: hub.CapBookmark, Backend: "karakeep", App: "karakeep",
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
				Type: hub.CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true,
			}))
			require.NoError(t, hub.Default.Register(hub.Descriptor{
				Type: hub.CapArchive, Backend: "archivebox", App: "archivebox", Healthy: false,
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
			assert.Equal(t, tt.want, checkLifecyclePermission(tt.perm, tt.action))
		})
	}
}
