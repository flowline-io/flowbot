package github

import (
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/github"
)

var cronRules = []cron.Rule{
	{
		Name: "github_starred",
		When: "*/10 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			// data
			client := github.NewGithub("", "", "", ctx.Token)
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
				r = append(r, types.RepoMsg{
					ID:               repo.ID,
					NodeID:           repo.NodeID,
					Name:             repo.Name,
					FullName:         repo.FullName,
					Description:      repo.Description,
					Homepage:         repo.Homepage,
					CreatedAt:        repo.CreatedAt,
					PushedAt:         repo.PushedAt,
					UpdatedAt:        repo.UpdatedAt,
					HTMLURL:          repo.HTMLURL,
					Language:         repo.Language,
					Fork:             repo.Fork,
					ForksCount:       repo.ForksCount,
					NetworkCount:     repo.NetworkCount,
					OpenIssuesCount:  repo.OpenIssuesCount,
					StargazersCount:  repo.StargazersCount,
					SubscribersCount: repo.SubscribersCount,
					WatchersCount:    repo.WatchersCount,
					Size:             repo.Size,
					Topics:           repo.Topics,
					Archived:         repo.Archived,
					Disabled:         repo.Disabled,
				})
			}
			return r
		},
	},
	{
		Name: "github_stargazers",
		When: "* * * * *",
		Action: func(types.Context) []types.MsgPayload {
			return nil
		},
	},
}
