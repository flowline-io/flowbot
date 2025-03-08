package gitea

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
)

func hookIssueOpened(ctx types.Context, issue *gitea.IssuePayload) error {
	return nil
}

func hookIssueCreated(ctx types.Context, issue *gitea.IssuePayload) error {
	return nil
}

func hookIssueClosed(ctx types.Context, issue *gitea.IssuePayload) error {
	return nil
}

func hookPush(ctx types.Context, payload *gitea.RepoPayload) error {
	ctx.SetTimeout(10 * time.Minute)
	owner := payload.Repository.Owner.UserName
	repo := payload.Repository.Name
	for _, commit := range payload.Commits {
		comment, err := reviewCommit(ctx.Context(), owner, repo, commit.Id)
		if err != nil {
			return fmt.Errorf("failed to review commit: %w", err)
		}

		err = event.BotEventFire(ctx, types.TaskCreateBotEventID, types.KV{
			"title":       fmt.Sprintf("Code Review: %s", comment.Path),
			"project_id":  kanboard.DefaultProjectId,
			"priority":    kanboard.DefaultPriority,
			"reference":   fmt.Sprintf("%s:commit:%s", gitea.ID, commit.Id),
			"description": fmt.Sprintf("%s/%s/%s/commit/%s\n\n%s", config.App.Search.UrlBaseMap[gitea.ID], owner, repo, commit.Id, comment.Body),
			"tags": []string{
				gitea.ID,
				"CodeReview",
			},
		})
		if err != nil {
			return fmt.Errorf("failed to fire event: %w", err)
		}
	}

	return nil
}
