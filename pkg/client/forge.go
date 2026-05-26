package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// ForgeClient provides access to the forge API.
type ForgeClient struct {
	c *Client
}

// GetUser returns the authenticated forge user.
func (f *ForgeClient) GetUser(ctx context.Context) (*ability.ForgeUser, error) {
	var result ability.ForgeUser
	err := f.c.Get(ctx, "/service/forge/user", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRepo returns a repository by owner and repo name.
func (f *ForgeClient) GetRepo(ctx context.Context, owner, repo string) (*ability.ForgeRepo, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}
	path := "/service/forge/repo?" + url.Values{
		"owner": {owner},
		"repo":  {repo},
	}.Encode()
	var result ability.ForgeRepo
	err := f.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListIssuesQuery contains query parameters for listing forge issues.
type ListIssuesQuery struct {
	State  string
	Limit  int
	Cursor string
}

// ListIssues returns issues for an owner with optional filtering.
func (f *ForgeClient) ListIssues(ctx context.Context, owner string, query *ListIssuesQuery) ([]*ability.ForgeIssue, error) {
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
	path := "/service/forge/issues?" + params.Encode()
	var result []*ability.ForgeIssue
	err := f.c.Get(ctx, path, &result)
	return result, err
}

// GetIssue returns a single issue by owner, repo, and issue index.
func (f *ForgeClient) GetIssue(ctx context.Context, owner, repo string, index int64) (*ability.ForgeIssue, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}
	if index <= 0 {
		return nil, fmt.Errorf("index must be positive, got %d", index)
	}
	path := "/service/forge/issue?" + url.Values{
		"owner": {owner},
		"repo":  {repo},
		"index": {strconv.FormatInt(index, 10)},
	}.Encode()
	var result ability.ForgeIssue
	err := f.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCommitDiff returns the diff for a specific commit.
func (f *ForgeClient) GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*ability.ForgeCommitDiff, error) {
	if owner == "" || repo == "" || commitID == "" {
		return nil, fmt.Errorf("owner, repo and commit_id are required")
	}
	path := "/service/forge/commit-diff?" + url.Values{
		"owner":     {owner},
		"repo":      {repo},
		"commit_id": {commitID},
	}.Encode()
	var result ability.ForgeCommitDiff
	err := f.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FileContentQuery contains optional query parameters for retrieving file content.
type FileContentQuery struct {
	LineStart int
	LineCount int
}

// GetFileContent returns the content of a file at a specific commit.
func (f *ForgeClient) GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, query *FileContentQuery) (string, error) {
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
	path := "/service/forge/file-content?" + params.Encode()
	var result string
	err := f.c.Get(ctx, path, &result)
	return result, err
}
