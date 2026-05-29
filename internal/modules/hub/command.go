// Package hub implements the hub management module providing chat commands
// for health checks, app management, and resource tag query endpoints.
// It consolidates the bookmark, github, kanban, note, and reader capabilities.
package hub

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	// --- Hub management ---
	{
		Define: "hub health",
		Help:   `Hub health status summary`,
		Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
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
		Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
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
		Handler: func(_ types.Context, tokens []*parser.Token) types.MsgPayload {
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
		Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
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

	// --- Bookmark ---
	{
		Define: "bookmark list",
		Help:   `newest 10`,
		Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, ability.OpBookmarkList, map[string]any{"limit": 10})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			var header []string
			var row [][]any
			bookmarks, ok := res.Data.([]*ability.Bookmark)
			if !ok {
				bookmarks = nil
			}
			if len(bookmarks) > 0 {
				header = []string{"Id", "Title", "URL"}
				for _, v := range bookmarks {
					row = append(row, []any{v.ID, v.Title, v.URL})
				}
			}

			return types.TableMsg{
				Title:  "Newest Bookmark List",
				Header: header,
				Row:    row,
			}
		},
	},

	// --- GitHub ---
	{
		Define: "github setting",
		Help:   `Bot setting`,
		Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
			return module.SettingMsg(ctx, Name)
		},
	},
	{
		Define: "github oauth",
		Help:   `OAuth`,
		Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
			oauth, err := providers.GetOrRefreshToken(ctx.Context(), ctx.AsUser, ctx.Topic, Name)
			if err != nil && !errors.Is(err, types.ErrNotFound) {
				flog.Error(err)
			}
			if oauth != nil && oauth.AccessToken != "" {
				return types.TextMsg{Text: "App is authorized"}
			}

			flag, err := module.StoreParameter(types.KV{
				"uid":   ctx.AsUser.String(),
				"topic": ctx.Topic,
			}, time.Now().Add(time.Hour))
			if err != nil {
				flog.Error(err)
				return nil
			}

			p, err := providers.GetOAuthProvider(Name)
			if err != nil {
				flog.Error(err)
				return nil
			}
			authorizeURL := p.GetAuthorizeURL(flag)
			return types.LinkMsg{Title: "OAuth", Url: authorizeURL}
		},
	},
	{
		Define: "github user",
		Help:   `Get current user info`,
		Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
			res, err := ability.Invoke(ctx.Context(), hub.CapGithub, ability.OpGithubGetUser, nil)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			user, ok := res.Data.(*ability.ForgeUser)
			if !ok || user == nil {
				return types.TextMsg{Text: "user error"}
			}
			return types.InfoMsg{
				Title: "User",
				Model: types.KV{
					"Login":  user.UserName,
					"URL":    user.AvatarURL,
					"UserID": user.ID,
					"Email":  user.Email,
				},
			}
		},
	},
	{
		Define: "github card [string]",
		Help:   `create project card`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			text, _ := tokens[1].Value.String()

			oauth, err := providers.GetOrRefreshToken(ctx.Context(), ctx.AsUser, ctx.Topic, Name)
			if err != nil {
				return nil
			}
			if oauth == nil || oauth.AccessToken == "" {
				return types.TextMsg{Text: "oauth error"}
			}

			_ = text
			return types.TextMsg{Text: "project card creation requires ability layer project operations"}
		},
	},
	{
		Define: "github repo [string]",
		Help:   "get repo info",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			str, _ := tokens[1].Value.String()

			repoArr := strings.Split(str, "/")
			if len(repoArr) != 2 {
				return types.TextMsg{Text: "repo error"}
			}

			res, err := ability.Invoke(ctx.Context(), hub.CapGithub, ability.OpGithubGetRepo, map[string]any{
				"owner": repoArr[0],
				"repo":  repoArr[1],
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			repo, ok := res.Data.(*ability.ForgeRepo)
			if !ok || repo == nil {
				return types.TextMsg{Text: "repo error"}
			}

			return types.KVMsg{
				"ID":          repo.ID,
				"Name":        repo.Name,
				"FullName":    repo.FullName,
				"Description": repo.Description,
				"HTMLURL":     repo.HTMLURL,
				"CloneURL":    repo.CloneURL,
			}
		},
	},
	{
		Define: "github user [string]",
		Help:   "get user info",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			username, _ := tokens[1].Value.String()

			res, err := ability.Invoke(ctx.Context(), hub.CapGithub, ability.OpGithubGetUserByLogin, map[string]any{
				"login": username,
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			user, ok := res.Data.(*ability.ForgeUser)
			if !ok || user == nil {
				return types.TextMsg{Text: "user error"}
			}

			return types.InfoMsg{
				Title: fmt.Sprintf("User %s", user.UserName),
				Model: user,
			}
		},
	},
	{
		Define: "deploy",
		Help:   `deploy server`,
		Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
			err := deploy(ctx)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("deploy failed, error: %v", err)}
			}

			return types.TextMsg{Text: "ok"}
		},
	},

	// --- Kanban ---
	{
		Define: "kanban status",
		Help:   `Show kanban status`,
		Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
			return types.EmptyMsg{}
		},
	},

	// --- Reader ---
	{
		Define: "reader",
		Help:   `show reader id`,
		Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: miniflux.ID}
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
