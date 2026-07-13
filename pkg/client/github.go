package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// GithubClient provides access to the github API.
type GithubClient struct {
	c *Client
}

// GetUser returns the authenticated github user.
func (g *GithubClient) GetUser(ctx context.Context) (*capability.ForgeUser, error) {
	var result capability.ForgeUser
	err := g.c.Get(ctx, "/service/github/user", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetUserByLogin returns a github user by login name.
func (g *GithubClient) GetUserByLogin(ctx context.Context, login string) (*capability.ForgeUser, error) {
	if login == "" {
		return nil, fmt.Errorf("login is required")
	}
	path := fmt.Sprintf("/service/github/user/%s", login)
	var result capability.ForgeUser
	err := g.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRepo returns a repository by owner and repo name.
func (g *GithubClient) GetRepo(ctx context.Context, owner, repo string) (*capability.ForgeRepo, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}
	path := "/service/github/repo?" + url.Values{
		"owner": {owner},
		"repo":  {repo},
	}.Encode()
	var result capability.ForgeRepo
	err := g.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListIssues returns issues for an owner with optional filtering.
func (g *GithubClient) ListIssues(ctx context.Context, owner string, query *ListIssuesQuery) ([]*capability.ForgeIssue, error) {
	if owner == "" {
		return nil, fmt.Errorf("owner is required")
	}
	params := url.Values{"owner": {owner}}
	if query != nil {
		if query.State != "" {
			params.Set("state", query.State)
		}
		if query.Limit > 0 {
			params.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			params.Set("cursor", query.Cursor)
		}
	}
	path := "/service/github/issues?" + params.Encode()
	var result []*capability.ForgeIssue
	err := g.c.Get(ctx, path, &result)
	return result, err
}

// GetIssue returns a single issue by owner, repo, and issue number.
func (g *GithubClient) GetIssue(ctx context.Context, owner, repo string, number int64) (*capability.ForgeIssue, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}
	if number <= 0 {
		return nil, fmt.Errorf("number must be positive, got %d", number)
	}
	path := "/service/github/issue?" + url.Values{
		"owner":  {owner},
		"repo":   {repo},
		"number": {strconv.FormatInt(number, 10)},
	}.Encode()
	var result capability.ForgeIssue
	err := g.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCommitDiff returns the diff for a specific commit.
func (g *GithubClient) GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*capability.ForgeCommitDiff, error) {
	if owner == "" || repo == "" || commitID == "" {
		return nil, fmt.Errorf("owner, repo and commit_id are required")
	}
	path := "/service/github/commit-diff?" + url.Values{
		"owner":     {owner},
		"repo":      {repo},
		"commit_id": {commitID},
	}.Encode()
	var result capability.ForgeCommitDiff
	err := g.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetFileContent returns the content of a file at a specific commit.
func (g *GithubClient) GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, query *FileContentQuery) (string, error) {
	if owner == "" || repo == "" || commitID == "" || filePath == "" {
		return "", fmt.Errorf("owner, repo, commit_id and file_path are required")
	}
	params := url.Values{
		"owner":     {owner},
		"repo":      {repo},
		"commit_id": {commitID},
		"file_path": {filePath},
	}
	if query != nil {
		if query.LineStart > 0 {
			params.Set("line_start", strconv.Itoa(query.LineStart))
		}
		if query.LineCount > 0 {
			params.Set("line_count", strconv.Itoa(query.LineCount))
		}
	}
	path := "/service/github/file-content?" + params.Encode()
	var result string
	err := g.c.Get(ctx, path, &result)
	return result, err
}

// ListNotificationsQuery contains query parameters for listing notifications.
type ListNotificationsQuery struct {
	Limit  int
	Cursor string
}

// ListNotifications returns the authenticated user's notifications.
func (g *GithubClient) ListNotifications(ctx context.Context, query *ListNotificationsQuery) ([]*capability.Notification, error) {
	params := url.Values{}
	if query != nil {
		if query.Limit > 0 {
			params.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			params.Set("cursor", query.Cursor)
		}
	}
	path := "/service/github/notifications"
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}
	var result []*capability.Notification
	err := g.c.Get(ctx, path, &result)
	return result, err
}

// ListReleases returns releases for a repository.
func (g *GithubClient) ListReleases(ctx context.Context, owner, repo string, query *ListNotificationsQuery) ([]*capability.Release, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}
	params := url.Values{
		"owner": {owner},
		"repo":  {repo},
	}
	if query != nil {
		if query.Limit > 0 {
			params.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			params.Set("cursor", query.Cursor)
		}
	}
	path := "/service/github/releases?" + params.Encode()
	var result []*capability.Release
	err := g.c.Get(ctx, path, &result)
	return result, err
}
