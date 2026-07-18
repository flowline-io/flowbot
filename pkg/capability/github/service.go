package github

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Capability is the GitHub capability type constant.
const Capability hub.CapabilityType = hub.CapGithub

// ListIssuesQuery wraps pagination and filtering for listing issues.
type ListIssuesQuery = capability.GithubListIssuesQuery

// PageQuery wraps pagination for list operations.
type PageQuery = capability.GithubPageQuery

// Service defines the GitHub capability contract.
type Service interface {
	GetUser(ctx context.Context) (*capability.ForgeUser, error)
	GetUserByLogin(ctx context.Context, login string) (*capability.ForgeUser, error)
	GetRepo(ctx context.Context, owner, repo string) (*capability.ForgeRepo, error)
	ListIssues(ctx context.Context, owner string, q *ListIssuesQuery) (*capability.ListResult[capability.ForgeIssue], error)
	GetIssue(ctx context.Context, owner, repo string, number int64) (*capability.ForgeIssue, error)
	GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*capability.ForgeCommitDiff, error)
	GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error)
	ListNotifications(ctx context.Context, q *PageQuery) (*capability.ListResult[capability.Notification], error)
	ListReleases(ctx context.Context, owner, repo string, q *PageQuery) (*capability.ListResult[capability.Release], error)
	HealthCheck(ctx context.Context) (bool, error)
}
