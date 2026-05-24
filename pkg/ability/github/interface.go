// Package github implements the GitHub capability.
package github

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Capability is the GitHub capability type constant.
const Capability hub.CapabilityType = hub.CapGithub

// ListIssuesQuery wraps pagination and filtering for listing issues.
type ListIssuesQuery struct {
	Page  ability.PageRequest
	State string // open, closed, all
}

// PageQuery wraps pagination for list operations.
type PageQuery struct {
	Page ability.PageRequest
}

// Service defines the GitHub capability contract.
// Provider adapters implement this interface to bridge the GitHub provider and invokers.
type Service interface {
	// GetUser returns the authenticated user's profile.
	GetUser(ctx context.Context) (*ability.ForgeUser, error)
	// GetUserByLogin returns a GitHub user's profile by login name.
	GetUserByLogin(ctx context.Context, login string) (*ability.ForgeUser, error)
	// GetRepo returns a single repository by owner and name.
	GetRepo(ctx context.Context, owner, repo string) (*ability.ForgeRepo, error)
	// ListIssues returns a paginated list of issues for the given owner.
	ListIssues(ctx context.Context, owner string, q *ListIssuesQuery) (*ability.ListResult[ability.ForgeIssue], error)
	// GetIssue returns a single issue by owner, repo name, and issue number.
	GetIssue(ctx context.Context, owner, repo string, number int64) (*ability.ForgeIssue, error)
	// GetCommitDiff returns the diff for a specific commit.
	GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*ability.ForgeCommitDiff, error)
	// GetFileContent returns file content at a specific commit with line range.
	GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error)
	// ListNotifications returns the authenticated user's notifications.
	ListNotifications(ctx context.Context, q *PageQuery) (*ability.ListResult[ability.Notification], error)
	// ListReleases returns releases for a repository.
	ListReleases(ctx context.Context, owner, repo string, q *PageQuery) (*ability.ListResult[ability.Release], error)
}
