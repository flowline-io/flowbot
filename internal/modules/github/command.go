// Package github implements the GitHub integration module.
package github

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
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
			// check oauth token
			oauth, err := store.Database.OAuthGet(ctx.Context(), ctx.AsUser, ctx.Topic, Name)
			if err != nil && !errors.Is(err, types.ErrNotFound) {
				flog.Error(err)
			}
			if oauth.Token != "" {
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
			id, _ := providers.GetConfig("github", "id")

			redirectURI := providers.RedirectURI("github", flag)
			authorizeURL := fmt.Sprintf(
				"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=repo",
				id.String(), redirectURI,
			)
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

			oauth, err := store.Database.OAuthGet(ctx.Context(), ctx.AsUser, ctx.Topic, Name)
			if err != nil {
				return nil
			}
			if oauth.Token == "" {
				return types.TextMsg{Text: "oauth error"}
			}

			// TODO: migrate to ability layer when project management operations are available.
			// The ability layer currently does not expose GitHub Projects (classic) operations.
			// These require: GetAuthenticatedUser, GetUserProjects, GetProjectColumns, CreateCard.
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
}
