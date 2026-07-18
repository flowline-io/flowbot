package gitea

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ListIssuesQuery wraps pagination and filtering for listing issues.
type ListIssuesQuery = capability.ForgeListIssuesQuery

// Service defines the forge capability contract.
type Service interface {
	GetUser(ctx context.Context) (*capability.ForgeUser, error)
	GetRepo(ctx context.Context, owner, repo string) (*capability.ForgeRepo, error)
	ListIssues(ctx context.Context, owner string, q *ListIssuesQuery) (*capability.ListResult[capability.ForgeIssue], error)
	GetIssue(ctx context.Context, owner, repo string, index int64) (*capability.ForgeIssue, error)
	GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*capability.ForgeCommitDiff, error)
	GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error)
	HealthCheck(ctx context.Context) (bool, error)
}
