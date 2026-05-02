package hub

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 7)
}

func TestCommandRules_Defines(t *testing.T) {
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
		{"hub health", "hub health", true},
		{"hub apps", "hub apps", true},
		{"hub app [name]", "hub app archivebox", true},
		{"hub capabilities", "hub capabilities", true},
		{"hub app start [name]", "hub app start archivebox", true},
		{"hub app stop [name]", "hub app stop archivebox", true},
		{"hub app restart [name]", "hub app restart archivebox", true},
		{"hub health", "hub apps", false},
		{"hub apps", "hub health", false},
		{"hub app [name]", "hub apps archivebox", false},
		{"hub app start [name]", "hub start archivebox", false},
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

func TestHubHealthHandler(t *testing.T) {
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
}

func TestHubAppsHandler(t *testing.T) {
	oldList := homelab.DefaultRegistry.List()
	defer homelab.DefaultRegistry.Replace(oldList)

	homelab.DefaultRegistry.Replace([]homelab.App{
		{Name: "archivebox", Status: homelab.AppStatusRunning, Health: homelab.HealthHealthy},
		{Name: "karakeep", Status: homelab.AppStatusStopped, Health: homelab.HealthUnhealthy},
	})

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

	msg, ok := payload.(types.InfoMsg)
	require.True(t, ok)
	assert.Equal(t, "Homelab Apps", msg.Title)
}

func TestHubAppsHandler_Empty(t *testing.T) {
	oldList := homelab.DefaultRegistry.List()
	defer homelab.DefaultRegistry.Replace(oldList)

	homelab.DefaultRegistry.Replace([]homelab.App{})

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

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "No apps registered", msg.Text)
}

func TestHubAppHandler(t *testing.T) {
	oldList := homelab.DefaultRegistry.List()
	defer homelab.DefaultRegistry.Replace(oldList)

	homelab.DefaultRegistry.Replace([]homelab.App{
		{Name: "archivebox", Path: "/apps/archivebox", Status: homelab.AppStatusRunning},
	})

	var appRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "hub app [name]" {
			appRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, appRule)

	tokens, _ := parser.ParseString("hub app archivebox")
	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	payload := appRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.InfoMsg)
	require.True(t, ok)
	assert.Equal(t, "App: archivebox", msg.Title)
}

func TestHubAppHandler_NotFound(t *testing.T) {
	oldList := homelab.DefaultRegistry.List()
	defer homelab.DefaultRegistry.Replace(oldList)

	homelab.DefaultRegistry.Replace([]homelab.App{})

	var appRule *command.Rule
	for i := range commandRules {
		if commandRules[i].Define == "hub app [name]" {
			appRule = &commandRules[i]
			break
		}
	}
	require.NotNil(t, appRule)

	tokens, _ := parser.ParseString("hub app nonexistent")
	ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
	payload := appRule.Handler(ctx, tokens)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "not found")
}

func TestHubCapabilitiesHandler(t *testing.T) {
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
}

func TestHubAppStartHandler_PermissionDenied(t *testing.T) {
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
}

func TestHubAppStopHandler_NotFound(t *testing.T) {
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
}

func TestHubAppRestartHandler_PermissionDenied(t *testing.T) {
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
}

func TestCheckLifecyclePermission(t *testing.T) {
	perm := homelab.Permissions{
		Start:   true,
		Stop:    false,
		Restart: true,
	}

	assert.True(t, checkLifecyclePermission(perm, "start"))
	assert.False(t, checkLifecyclePermission(perm, "stop"))
	assert.True(t, checkLifecyclePermission(perm, "restart"))
	assert.False(t, checkLifecyclePermission(perm, "unknown"))
}

func TestCheckLifecyclePermission_AllFalse(t *testing.T) {
	var perm homelab.Permissions

	assert.False(t, checkLifecyclePermission(perm, "start"))
	assert.False(t, checkLifecyclePermission(perm, "stop"))
	assert.False(t, checkLifecyclePermission(perm, "restart"))
}
