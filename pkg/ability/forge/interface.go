// Package forge implements the software forge capability (Gitea, Gogs, Forgejo).
package forge

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// ListIssuesQuery wraps pagination and filtering for listing issues.
type ListIssuesQuery struct {
	Page  ability.PageRequest
	State string // open, closed, all
}

// Service defines the forge capability contract.
// Provider adapters implement this interface to bridge forge providers and invokers.
type Service interface {
	// GetUser returns the authenticated user's profile.
	GetUser(ctx context.Context) (*ability.ForgeUser, error)
	// GetRepo returns a single repository by owner and name.
	GetRepo(ctx context.Context, owner, repo string) (*ability.ForgeRepo, error)
	// ListIssues returns a paginated list of issues for the given owner.
	ListIssues(ctx context.Context, owner string, q *ListIssuesQuery) (*ability.ListResult[ability.ForgeIssue], error)
	// GetIssue returns a single issue by owner, repo name, and issue index.
	GetIssue(ctx context.Context, owner, repo string, index int64) (*ability.ForgeIssue, error)
	// GetCommitDiff returns the diff for a specific commit.
	GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*ability.ForgeCommitDiff, error)
	// GetFileContent returns file content at a specific commit with line range.
	GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error)
}
