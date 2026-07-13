// Package gitea implements the Gitea adapter for the forge capability.
package gitea

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/types"

	giteasdk "code.gitea.io/sdk/gitea"
)

var defaultCursorSecret = []byte("flowbot-ability-forge-gitea-cursor-v1")

// client defines the subset of provider.Gitea methods used by this adapter.
type client interface {
	GetMyUserInfo() (*giteasdk.User, error)
	GetRepositories(owner, reponame string) (*giteasdk.Repository, error)
	ListIssues(owner string, page, pageSize int) ([]*giteasdk.Issue, error)
	GetDiff(owner, repo, commitID string) (*provider.CommitDiff, error)
	GetFileContent(owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error)
}

// Adapter implements Service using the Gitea provider client.
type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
}

// New creates an Adapter using the default provider client.
// Returns nil when the provider is not configured or unavailable.
func New() Service {
	client, err := provider.GetClient()
	if err != nil {
		flog.Error(fmt.Errorf("gitea forge adapter: %w", err))
		return nil
	}
	if client == nil {
		return nil
	}
	return NewWithClient(client)
}

// NewWithClient creates an Adapter with a specific client, useful for testing.
func NewWithClient(c client) Service {
	return &Adapter{
		client:       c,
		cursorSecret: defaultCursorSecret,
		now:          time.Now,
	}
}

// SetCursorSecret sets the cursor signing secret (for testing).
func (a *Adapter) SetCursorSecret(secret []byte) {
	a.cursorSecret = secret
}

// GetUser returns the authenticated user's profile from the Gitea API.
func (a *Adapter) GetUser(ctx context.Context) (*capability.ForgeUser, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "forge get user canceled", err)
	}
	user, err := a.client.GetMyUserInfo()
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "gitea get user", err)
	}
	return toForgeUser(user), nil
}

// GetRepo returns a single repository by owner and name from the Gitea API.
func (a *Adapter) GetRepo(ctx context.Context, owner, repo string) (*capability.ForgeRepo, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "forge get repo canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "repo is required")
	}
	r, err := a.client.GetRepositories(owner, repo)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "gitea get repo", err)
	}
	if r == nil {
		return nil, types.Errorf(types.ErrNotFound, "repo %s/%s not found", owner, repo)
	}
	return toForgeRepo(r), nil
}

// ListIssues returns a paginated list of issues for the given owner from the Gitea API.

func (a *Adapter) ListIssues(ctx context.Context, owner string, q *ListIssuesQuery) (*capability.ListResult[capability.ForgeIssue], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "forge list issues canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if q == nil {
		q = &ListIssuesQuery{}
	}
	limit := normalizedLimit(q.Page.Limit)
	page := 1
	if q.Page.Cursor != "" {
		payload, err := capability.DecodeCursor(a.cursorSecret, q.Page.Cursor, a.now())
		if err != nil {
			return nil, err
		}
		if p, err := strconv.Atoi(payload.ProviderCursor); err == nil {
			page = p
		}
	}
	issues, err := a.client.ListIssues(owner, page, limit)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "gitea list issues", err)
	}
	items := make([]*capability.ForgeIssue, 0, len(issues))
	for i := range issues {
		items = append(items, toForgeIssue(issues[i]))
	}
	result := &capability.ListResult[capability.ForgeIssue]{
		Items: items,
		Page: &capability.PageInfo{
			Limit:   limit,
			HasMore: len(issues) >= limit,
		},
	}
	if result.Page.HasMore {
		nextCursor, err := capability.EncodeCursor(a.cursorSecret, capability.CursorPayload{
			Capability:     "gitea",
			Strategy:       "offset",
			ProviderCursor: strconv.Itoa(page + 1),
			Limit:          limit,
		})
		if err != nil {
			return nil, err
		}
		result.Page.NextCursor = nextCursor
	}
	return result, nil
}

// GetIssue returns a single issue by owner, repo name, and index from the Gitea API.
// Because the Gitea SDK does not expose a direct single-issue endpoint, this method
// fetches issues for the owner and filters by repository name and index.
func (a *Adapter) GetIssue(ctx context.Context, owner, repo string, index int64) (*capability.ForgeIssue, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "forge get issue canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "repo is required")
	}
	issues, err := a.client.ListIssues(owner, 1, 100)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "gitea get issue", err)
	}
	for _, issue := range issues {
		if issue.Index == index && issue.Repository != nil && issue.Repository.Name == repo {
			return toForgeIssue(issue), nil
		}
	}
	return nil, types.Errorf(types.ErrNotFound, "issue #%d not found in %s/%s", index, owner, repo)
}

// GetCommitDiff returns the diff for a specific commit from the Gitea API.
func (a *Adapter) GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*capability.ForgeCommitDiff, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "forge get commit diff canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "repo is required")
	}
	if commitID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "commit_id is required")
	}
	diff, err := a.client.GetDiff(owner, repo, commitID)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "gitea get commit diff", err)
	}
	if diff == nil {
		return nil, types.Errorf(types.ErrNotFound, "commit %s not found in %s/%s", commitID, owner, repo)
	}
	return &capability.ForgeCommitDiff{
		CommitID:      diff.CommitID,
		CommitMessage: diff.CommitMessage,
		Files:         diff.Files,
		DiffContent:   diff.DiffContent,
	}, nil
}

// GetFileContent returns file content at a specific commit with line range from the Gitea API.
func (a *Adapter) GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "forge get file content canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "repo is required")
	}
	if commitID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "commit_id is required")
	}
	if filePath == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "file_path is required")
	}
	content, err := a.client.GetFileContent(owner, repo, commitID, filePath, lineStart, lineCount)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "gitea get file content", err)
	}
	return content, nil
}

func normalizedLimit(limit int) int {
	const defaultLimit = 50
	const maxLimit = 100
	if limit <= 0 || limit > maxLimit {
		return defaultLimit
	}
	return limit
}

func toForgeUser(user *giteasdk.User) *capability.ForgeUser {
	if user == nil {
		return nil
	}
	return &capability.ForgeUser{
		ID:        user.ID,
		UserName:  user.UserName,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
	}
}

func toForgeRepo(repo *giteasdk.Repository) *capability.ForgeRepo {
	if repo == nil {
		return nil
	}
	ownerName := ""
	if repo.Owner != nil {
		ownerName = repo.Owner.UserName
	}
	return &capability.ForgeRepo{
		ID:          repo.ID,
		Name:        repo.Name,
		FullName:    repo.FullName,
		Description: repo.Description,
		Private:     repo.Private,
		HTMLURL:     repo.HTMLURL,
		CloneURL:    repo.CloneURL,
		Owner:       ownerName,
	}
}

func toForgeIssue(issue *giteasdk.Issue) *capability.ForgeIssue {
	if issue == nil {
		return nil
	}
	author := ""
	if issue.Poster != nil {
		author = issue.Poster.UserName
	}
	return &capability.ForgeIssue{
		ID:      issue.ID,
		Index:   issue.Index,
		Title:   issue.Title,
		Body:    issue.Body,
		State:   string(issue.State),
		HTMLURL: issue.HTMLURL,
		Author:  author,
	}
}
