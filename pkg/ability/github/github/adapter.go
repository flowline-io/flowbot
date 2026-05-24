// Package github implements the GitHub adapter for the github capability.
package github

import (
	"context"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	githubsvc "github.com/flowline-io/flowbot/pkg/ability/github"
	provider "github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/types"
)

var defaultCursorSecret = []byte("flowbot-ability-github-cursor-v1")

// client defines the subset of provider.Github methods used by this adapter.
type client interface {
	GetAuthenticatedUser() (*provider.User, error)
	GetUser(username string) (*provider.User, error)
	GetRepository(owner, repo string) (*provider.Repository, error)
	ListIssues(owner string, page, pageSize int, state string) ([]*provider.Issue, error)
	GetDiff(owner, repo, commitID string) (*provider.CommitDiff, error)
	GetFileContent(owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error)
	GetNotifications() ([]*provider.Notification, error)
	GetReleases(owner, repo string, page, perPage int) ([]*provider.RepositoryRelease, error)
}

// Adapter implements githubsvc.Service using the GitHub provider client.
type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
}

// New creates an Adapter using the default provider client.
func New() githubsvc.Service {
	client := provider.GetClient()
	return NewWithClient(client)
}

// NewWithClient creates an Adapter with a specific client, useful for testing.
func NewWithClient(c client) githubsvc.Service {
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

// GetUser returns the authenticated user's profile from the GitHub API.
func (a *Adapter) GetUser(ctx context.Context) (*ability.ForgeUser, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github get user canceled", err)
	}
	user, err := a.client.GetAuthenticatedUser()
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "github get user", err)
	}
	return toForgeUser(user), nil
}

// GetUserByLogin returns a GitHub user's profile by login name from the GitHub API.
func (a *Adapter) GetUserByLogin(ctx context.Context, login string) (*ability.ForgeUser, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github get user by login canceled", err)
	}
	if login == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "login is required")
	}
	user, err := a.client.GetUser(login)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "github get user by login", err)
	}
	if user == nil {
		return nil, types.Errorf(types.ErrNotFound, "user %s not found", login)
	}
	return toForgeUser(user), nil
}

// GetRepo returns a single repository by owner and name from the GitHub API.
func (a *Adapter) GetRepo(ctx context.Context, owner, repo string) (*ability.ForgeRepo, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github get repo canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "repo is required")
	}
	r, err := a.client.GetRepository(owner, repo)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "github get repo", err)
	}
	if r == nil {
		return nil, types.Errorf(types.ErrNotFound, "repo %s/%s not found", owner, repo)
	}
	return toForgeRepo(r), nil
}

// ListIssues returns a paginated list of issues for the given owner from the GitHub API.
func (a *Adapter) ListIssues(ctx context.Context, owner string, q *githubsvc.ListIssuesQuery) (*ability.ListResult[ability.ForgeIssue], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github list issues canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if q == nil {
		q = &githubsvc.ListIssuesQuery{}
	}
	limit := normalizedLimit(q.Page.Limit)
	page := 1
	if q.Page.Cursor != "" {
		payload, err := ability.DecodeCursor(a.cursorSecret, q.Page.Cursor, a.now())
		if err != nil {
			return nil, err
		}
		if p, err := strconv.Atoi(payload.ProviderCursor); err == nil {
			page = p
		}
	}
	issues, err := a.client.ListIssues(owner, page, limit, q.State)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "github list issues", err)
	}
	items := make([]*ability.ForgeIssue, 0, len(issues))
	for i := range issues {
		items = append(items, toForgeIssue(issues[i]))
	}
	result := &ability.ListResult[ability.ForgeIssue]{
		Items: items,
		Page: &ability.PageInfo{
			Limit:   limit,
			HasMore: len(issues) >= limit,
		},
	}
	if result.Page.HasMore {
		nextCursor, err := ability.EncodeCursor(a.cursorSecret, ability.CursorPayload{
			Capability:     "github",
			Backend:        provider.ID,
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

// GetIssue returns a single issue by owner, repo name, and index.
// Lists issues for the owner and filters by repository name and index.
func (a *Adapter) GetIssue(ctx context.Context, owner, repo string, index int64) (*ability.ForgeIssue, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github get issue canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "repo is required")
	}
	issues, err := a.client.ListIssues(owner, 1, 100, "")
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "github get issue", err)
	}
	for _, iss := range issues {
		if iss.Number != nil && int64(*iss.Number) == index {
			repoName := ""
			if iss.Repository != nil && iss.Repository.Name != nil {
				repoName = *iss.Repository.Name
			}
			if repoName == repo {
				return toForgeIssue(iss), nil
			}
		}
	}
	return nil, types.Errorf(types.ErrNotFound, "issue #%d not found in %s/%s", index, owner, repo)
}

// GetCommitDiff returns the diff for a specific commit from the GitHub API.
func (a *Adapter) GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*ability.ForgeCommitDiff, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github get commit diff canceled", err)
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
		return nil, types.WrapError(types.ErrProvider, "github get commit diff", err)
	}
	if diff == nil {
		return nil, types.Errorf(types.ErrNotFound, "commit %s not found in %s/%s", commitID, owner, repo)
	}
	return &ability.ForgeCommitDiff{
		CommitID:      diff.CommitID,
		CommitMessage: diff.CommitMessage,
		Files:         diff.Files,
		DiffContent:   diff.DiffContent,
	}, nil
}

// GetFileContent returns file content at a specific commit with line range from the GitHub API.
func (a *Adapter) GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github get file content canceled", err)
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
		return nil, types.WrapError(types.ErrProvider, "github get file content", err)
	}
	return content, nil
}

// ListNotifications returns the authenticated user's notifications from the GitHub API.
func (a *Adapter) ListNotifications(ctx context.Context, q *githubsvc.PageQuery) (*ability.ListResult[ability.Notification], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github list notifications canceled", err)
	}
	if q == nil {
		q = &githubsvc.PageQuery{}
	}
	notifications, err := a.client.GetNotifications()
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "github list notifications", err)
	}
	items := make([]*ability.Notification, 0, len(notifications))
	for i := range notifications {
		items = append(items, toNotification(notifications[i]))
	}
	return &ability.ListResult[ability.Notification]{
		Items: items,
		Page:  &ability.PageInfo{Limit: normalizedLimit(q.Page.Limit), HasMore: false},
	}, nil
}

// ListReleases returns releases for a repository from the GitHub API.
func (a *Adapter) ListReleases(ctx context.Context, owner, repo string, q *githubsvc.PageQuery) (*ability.ListResult[ability.Release], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "github list releases canceled", err)
	}
	if owner == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "owner is required")
	}
	if repo == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "repo is required")
	}
	if q == nil {
		q = &githubsvc.PageQuery{}
	}
	limit := normalizedLimit(q.Page.Limit)
	page := 1
	if q.Page.Cursor != "" {
		payload, err := ability.DecodeCursor(a.cursorSecret, q.Page.Cursor, a.now())
		if err != nil {
			return nil, err
		}
		if p, err := strconv.Atoi(payload.ProviderCursor); err == nil {
			page = p
		}
	}
	releases, err := a.client.GetReleases(owner, repo, page, limit)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "github list releases", err)
	}
	items := make([]*ability.Release, 0, len(releases))
	for i := range releases {
		items = append(items, toRelease(releases[i]))
	}
	result := &ability.ListResult[ability.Release]{
		Items: items,
		Page: &ability.PageInfo{
			Limit:   limit,
			HasMore: len(releases) >= limit,
		},
	}
	if result.Page.HasMore {
		nextCursor, err := ability.EncodeCursor(a.cursorSecret, ability.CursorPayload{
			Capability:     "github",
			Backend:        provider.ID,
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

func normalizedLimit(limit int) int {
	const defaultLimit = 50
	const maxLimit = 100
	if limit <= 0 || limit > maxLimit {
		return defaultLimit
	}
	return limit
}

func toForgeUser(user *provider.User) *ability.ForgeUser {
	if user == nil {
		return nil
	}
	return &ability.ForgeUser{
		ID:        derefInt64(user.ID),
		UserName:  derefString(user.Login),
		Email:     derefString(user.Email),
		AvatarURL: derefString(user.AvatarURL),
	}
}

func toForgeRepo(repo *provider.Repository) *ability.ForgeRepo {
	if repo == nil {
		return nil
	}
	ownerName := ""
	if repo.Owner != nil {
		ownerName = derefString(repo.Owner.Login)
	}
	return &ability.ForgeRepo{
		ID:          derefInt64(repo.ID),
		Name:        derefString(repo.Name),
		FullName:    derefString(repo.FullName),
		Description: derefString(repo.Description),
		Private:     derefBool(repo.Private),
		HTMLURL:     derefString(repo.HTMLURL),
		CloneURL:    derefString(repo.CloneURL),
		Owner:       ownerName,
	}
}

func toForgeIssue(issue *provider.Issue) *ability.ForgeIssue {
	if issue == nil {
		return nil
	}
	author := ""
	if issue.User != nil {
		author = derefString(issue.User.Login)
	}
	return &ability.ForgeIssue{
		ID:      derefInt64(issue.ID),
		Index:   int64(derefInt(issue.Number)),
		Title:   derefString(issue.Title),
		Body:    derefString(issue.Body),
		State:   derefString(issue.State),
		HTMLURL: derefString(issue.HTMLURL),
		Author:  author,
	}
}

func toNotification(n *provider.Notification) *ability.Notification {
	if n == nil {
		return nil
	}
	repoName := ""
	if n.Repository != nil {
		repoName = derefString(n.Repository.FullName)
	}
	subject := ""
	if n.Subject != nil {
		subject = derefString(n.Subject.Title)
	}
	return &ability.Notification{
		ID:       derefString(n.ID),
		Reason:   derefString(n.Reason),
		Unread:   derefBool(n.Unread),
		Subject:  subject,
		RepoName: repoName,
	}
}

func toRelease(r *provider.RepositoryRelease) *ability.Release {
	if r == nil {
		return nil
	}
	return &ability.Release{
		ID:         derefInt64(r.ID),
		TagName:    derefString(r.TagName),
		Name:       derefString(r.Name),
		Body:       derefString(r.Body),
		Draft:      derefBool(r.Draft),
		Prerelease: derefBool(r.Prerelease),
		HTMLURL:    derefString(r.HTMLURL),
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// Compile-time interface check.
var _ githubsvc.Service = (*Adapter)(nil)
