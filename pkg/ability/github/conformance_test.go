package github

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

type conformanceService struct {
	cfg Config
}

func (c *conformanceService) GetUser(ctx context.Context) (*ability.ForgeUser, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if c.cfg.UserErr != nil {
		return nil, types.WrapError(types.ErrProvider, "provider error", c.cfg.UserErr)
	}
	return c.cfg.User, nil
}

func (c *conformanceService) GetRepo(ctx context.Context, owner, repo string) (*ability.ForgeRepo, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "repo is required")
	}
	if c.cfg.RepoErr != nil {
		return nil, types.WrapError(types.ErrProvider, "provider error", c.cfg.RepoErr)
	}
	return c.cfg.Repo, nil
}

func (c *conformanceService) ListIssues(ctx context.Context, owner string, _ *ListIssuesQuery) (*ability.ListResult[ability.ForgeIssue], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if c.cfg.IssuesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "provider error", c.cfg.IssuesErr)
	}
	items := c.cfg.Issues
	if items == nil {
		items = []*ability.ForgeIssue{}
	}
	return &ability.ListResult[ability.ForgeIssue]{Items: items, Page: &ability.PageInfo{Limit: 20}}, nil
}

func (c *conformanceService) GetIssue(ctx context.Context, owner, _ string, _ int64) (*ability.ForgeIssue, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if c.cfg.IssuesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "provider error", c.cfg.IssuesErr)
	}
	if len(c.cfg.Issues) > 0 {
		return c.cfg.Issues[0], nil
	}
	return nil, types.Errorf(types.ErrNotFound, "issue not found")
}

func (c *conformanceService) GetCommitDiff(ctx context.Context, owner, _ string, commitID string) (*ability.ForgeCommitDiff, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if commitID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "commit_id is required")
	}
	if c.cfg.DiffErr != nil {
		return nil, types.WrapError(types.ErrProvider, "provider error", c.cfg.DiffErr)
	}
	return c.cfg.Diff, nil
}

func (c *conformanceService) GetFileContent(ctx context.Context, owner, _, _, filePath string, _, _ int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if filePath == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "file_path is required")
	}
	if c.cfg.FileContentErr != nil {
		return nil, types.WrapError(types.ErrProvider, "provider error", c.cfg.FileContentErr)
	}
	return c.cfg.FileContent, nil
}

func (c *conformanceService) ListNotifications(ctx context.Context, _ *PageQuery) (*ability.ListResult[ability.Notification], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if c.cfg.NotificationsErr != nil {
		return nil, types.WrapError(types.ErrProvider, "provider error", c.cfg.NotificationsErr)
	}
	items := c.cfg.Notifications
	if items == nil {
		items = []*ability.Notification{}
	}
	return &ability.ListResult[ability.Notification]{Items: items, Page: &ability.PageInfo{Limit: 10}}, nil
}

func (c *conformanceService) ListReleases(ctx context.Context, owner, _ string, _ *PageQuery) (*ability.ListResult[ability.Release], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if c.cfg.ReleasesErr != nil {
		return nil, types.WrapError(types.ErrProvider, "provider error", c.cfg.ReleasesErr)
	}
	items := c.cfg.Releases
	if items == nil {
		items = []*ability.Release{}
	}
	return &ability.ListResult[ability.Release]{Items: items, Page: &ability.PageInfo{Limit: 10}}, nil
}

func TestRunGithubConformance(t *testing.T) {
	t.Run("runs github conformance test suite", func(t *testing.T) {
		RunGithubConformance(t, func(_ *testing.T, cfg Config) Service {
			return &conformanceService{cfg: cfg}
		})
	})
}
