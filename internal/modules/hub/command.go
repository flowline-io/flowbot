package hub

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "hub health",
		Help:   `Hub health status summary`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			checker := hub.NewChecker(hub.Default)
			result := checker.Check(ctx.Context())

			return types.InfoMsg{
				Title: "Hub Health",
				Model: result,
			}
		},
	},
	{
		Define: "hub apps",
		Help:   `List all registered homelab apps`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			apps := homelab.DefaultRegistry.List()
			if len(apps) == 0 {
				return types.TextMsg{Text: "No apps registered"}
			}

			return types.InfoMsg{
				Title: "Homelab Apps",
				Model: apps,
			}
		},
	},
	{
		Define: "hub app [name]",
		Help:   `View app details and health`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			name, _ := tokens[2].Value.String()
			app, ok := homelab.DefaultRegistry.Get(name)
			if !ok {
				return types.TextMsg{Text: fmt.Sprintf("App %q not found", name)}
			}

			return types.InfoMsg{
				Title: fmt.Sprintf("App: %s", name),
				Model: app,
			}
		},
	},
	{
		Define: "hub capabilities",
		Help:   `List all capabilities and their bindings`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			bindings := hub.Default.Bindings()
			if len(bindings) == 0 {
				return types.TextMsg{Text: "No capabilities registered"}
			}

			return types.InfoMsg{
				Title: "Hub Capabilities",
				Model: bindings,
			}
		},
	},
	{
		Define: "hub app start [name]",
		Help:   `Start an app (requires permission)`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			name, _ := tokens[3].Value.String()
			app, err := checkLifecycleOp(name, "start")
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			runCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
			defer cancel()

			if err := homelab.DefaultRuntime.Start(runCtx, app); err != nil {
				return types.TextMsg{Text: fmt.Sprintf("Failed to start %s: %v", name, err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("App %s started", name)}
		},
	},
	{
		Define: "hub app stop [name]",
		Help:   `Stop an app (requires permission)`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			name, _ := tokens[3].Value.String()
			app, err := checkLifecycleOp(name, "stop")
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			runCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
			defer cancel()

			if err := homelab.DefaultRuntime.Stop(runCtx, app); err != nil {
				return types.TextMsg{Text: fmt.Sprintf("Failed to stop %s: %v", name, err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("App %s stopped", name)}
		},
	},
	{
		Define: "hub app restart [name]",
		Help:   `Restart an app (requires permission)`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			name, _ := tokens[3].Value.String()
			app, err := checkLifecycleOp(name, "restart")
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			runCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
			defer cancel()

			if err := homelab.DefaultRuntime.Restart(runCtx, app); err != nil {
				return types.TextMsg{Text: fmt.Sprintf("Failed to restart %s: %v", name, err)}
			}

			return types.TextMsg{Text: fmt.Sprintf("App %s restarted", name)}
		},
	},
}

func checkLifecycleOp(name, operation string) (homelab.App, error) {
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return app, types.Errorf(types.ErrNotFound, "app not found")
	}

	perm := homelab.DefaultRegistry.Permissions()
	if !checkLifecyclePermission(perm, operation) {
		return app, types.Errorf(types.ErrForbidden, "%s not allowed by config for app %s", operation, name)
	}

	return app, nil
}

func checkLifecyclePermission(perm homelab.Permissions, operation string) bool {
	switch operation {
	case "start":
		return perm.Start
	case "stop":
		return perm.Stop
	case "restart":
		return perm.Restart
	default:
		return false
	}
}
