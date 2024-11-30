package github

import (
	"errors"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
	"gorm.io/gorm"
)

var cronRules = []cron.Rule{
	{
		Name:  "github_starred",
		Scope: cron.CronScopeUser,
		When:  "*/10 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			// get oauth token
			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, Name)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				flog.Error(err)
				return nil
			}
			if oauth.Token == "" {
				return nil
			}

			// data
			client := github.NewGithub("", "", "", oauth.Token)
			user, err := client.GetAuthenticatedUser()
			if err != nil {
				flog.Error(err)
				return []types.MsgPayload{}
			}
			if *user.Login == "" {
				return []types.MsgPayload{}
			}

			repos, err := client.GetStarred(*user.Login)
			if err != nil {
				flog.Error(err)
				return []types.MsgPayload{}
			}
			reposList := *repos
			var r []types.MsgPayload
			for i := range reposList {
				repo := reposList[i]
				r = append(r, types.InfoMsg{
					Title: *repo.FullName,
					Model: types.KV{
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
					},
				})
			}
			return r
		},
	},
}
