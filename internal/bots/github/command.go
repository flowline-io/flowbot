package github

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	"gorm.io/gorm"
)

var commandRules = []command.Rule{
	{
		Define: "github setting",
		Help:   `Bot setting`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.SettingMsg(ctx, Name)
		},
	},
	{
		Define: "github oauth",
		Help:   `OAuth`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			// check oauth token
			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, Name)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				flog.Error(err)
			}
			if oauth.Token != "" {
				return types.TextMsg{Text: "App is authorized"}
			}

			flag, err := bots.StoreParameter(types.KV{
				"uid":   ctx.AsUser.String(),
				"topic": ctx.Topic,
			}, time.Now().Add(time.Hour))
			if err != nil {
				flog.Error(err)
				return nil
			}
			id, _ := providers.GetConfig(github.ID, github.ClientIdKey)
			secret, _ := providers.GetConfig(github.ID, github.ClientSecretKey)
			redirectURI := providers.RedirectURI(github.ID, flag)
			provider := github.NewGithub(id.String(), secret.String(), redirectURI, "")
			return types.LinkMsg{Title: "OAuth", Url: provider.GetAuthorizeURL()}
		},
	},
	{
		Define: "github user",
		Help:   `Get current user info`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			// get token
			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, Name)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				flog.Error(err)
			}
			if oauth.Token == "" {
				return types.TextMsg{Text: "App is unauthorized"}
			}

			provider := github.NewGithub("", "", "", oauth.Token)

			user, err := provider.GetAuthenticatedUser()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			if user == nil {
				return types.TextMsg{Text: "user error"}
			}

			return types.InfoMsg{
				Title: "User",
				Model: types.KV{
					"Login":     *user.Login,
					"Followers": *user.Followers,
					"Following": *user.Following,
					"URL":       *user.HTMLURL,
				},
			}
		},
	},
	{
		Define: "github issue [string]",
		Help:   `create issue`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			text, _ := tokens[1].Value.String()

			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, github.ID)
			if err != nil {
				return nil
			}
			if oauth.Token == "" {
				return types.TextMsg{Text: "oauth error"}
			}

			// get user
			client := github.NewGithub("", "", "", oauth.Token)
			user, err := client.GetAuthenticatedUser()
			if err != nil {
				return nil
			}
			if *user.Login == "" {
				return nil
			}

			// repo value
			j, err := bots.SettingGet(ctx, Name, repoSettingKey)
			if err != nil {
				return nil
			}
			repo, _ := j.StringValue()
			if repo == "" {
				return types.TextMsg{Text: "set repo [string]"}
			}

			// create issue
			issue, err := client.CreateIssue(*user.Login, repo, github.Issue{Title: &text})
			if err != nil {
				flog.Error(err)
				return nil
			}
			if *issue.ID == 0 {
				return nil
			}

			return types.LinkMsg{
				Title: fmt.Sprintf("Issue #%d", *issue.Number),
				Url:   *issue.HTMLURL,
			}
		},
	},
	{
		Define: "github card [string]",
		Help:   `create project card`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			text, _ := tokens[1].Value.String()

			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, github.ID)
			if err != nil {
				return nil
			}
			if oauth.Token == "" {
				return types.TextMsg{Text: "oauth error"}
			}

			// get user
			client := github.NewGithub("", "", "", oauth.Token)
			user, err := client.GetAuthenticatedUser()
			if err != nil {
				return nil
			}
			if *user.Login == "" {
				return nil
			}

			// get projects
			projects, err := client.GetUserProjects(*user.Login)
			if err != nil {
				flog.Error(err)
				return nil
			}
			if len(*projects) == 0 {
				return nil
			}

			// get columns
			columns, err := client.GetProjectColumns(*(*projects)[0].ID)
			if err != nil {
				flog.Error(err)
				return nil
			}
			if len(*columns) == 0 {
				return nil
			}

			// create card
			card, err := client.CreateCard(*(*columns)[0].ID, github.ProjectCard{Note: &text})
			if err != nil {
				flog.Error(err)
				return nil
			}
			if *card.ID == 0 {
				return nil
			}

			return types.TextMsg{Text: fmt.Sprintf("Created Project Card #%d", *card.ID)}
		},
	},
	{
		Define: "github repo [string]",
		Help:   "get repo info",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			str, _ := tokens[1].Value.String()

			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, github.ID)
			if err != nil {
				return nil
			}
			if oauth.Token == "" {
				return types.TextMsg{Text: "oauth error"}
			}

			client := github.NewGithub("", "", "", oauth.Token)

			repoArr := strings.Split(str, "/")
			if len(repoArr) != 2 {
				return types.TextMsg{Text: "repo error"}
			}
			repo, err := client.GetRepository(repoArr[0], repoArr[1])
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "repo error"}
			}

			return types.KVMsg{
				"ID":               repo.ID,
				"NodeID":           repo.NodeID,
				"Name":             repo.Name,
				"FullName":         repo.FullName,
				"Description":      repo.Description,
				"Homepage":         repo.Homepage,
				"CreatedAt":        repo.CreatedAt,
				"PushedAt":         repo.PushedAt,
				"UpdatedAt":        repo.UpdatedAt,
				"HTMLURL":          repo.HTMLURL,
				"Language":         repo.Language,
				"Fork":             repo.Fork,
				"ForksCount":       repo.ForksCount,
				"NetworkCount":     repo.NetworkCount,
				"OpenIssuesCount":  repo.OpenIssuesCount,
				"StargazersCount":  repo.StargazersCount,
				"SubscribersCount": repo.SubscribersCount,
				"WatchersCount":    repo.WatchersCount,
				"Size":             repo.Size,
				"Topics":           repo.Topics,
				"Archived":         repo.Archived,
				"Disabled":         repo.Disabled,
			}
		},
	},
	{
		Define: "github user [string]",
		Help:   "get user info",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			username, _ := tokens[1].Value.String()

			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, github.ID)
			if err != nil {
				return nil
			}
			if oauth.Token == "" {
				return types.TextMsg{Text: "oauth error"}
			}

			client := github.NewGithub("", "", "", oauth.Token)

			user, err := client.GetUser(username)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "user error"}
			}

			return types.InfoMsg{
				Title: fmt.Sprintf("User %s", *user.Login),
				Model: user,
			}
		},
	},
	{
		Define: "deploy",
		Help:   `deploy server`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := deploy(ctx)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("deploy failed, error: %v", err)}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
}
