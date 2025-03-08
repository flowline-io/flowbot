package gitea

import (
	"bytes"
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
)

const (
	ID          = "gitea"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type Gitea struct {
	token string
	c     *gitea.Client
}

func GetClient() (*Gitea, error) {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)

	return NewGitea(endpoint.String(), token.String())
}

func NewGitea(endpoint, token string) (*Gitea, error) {
	var err error
	v := &Gitea{token: token}
	if config.App.Log.Level == flog.DebugLevel {
		v.c, err = gitea.NewClient(endpoint, gitea.SetToken(token), gitea.SetDebugMode())
		if err != nil {
			return nil, err
		}
	} else {
		v.c, err = gitea.NewClient(endpoint, gitea.SetToken(token))
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}

func (v *Gitea) GetRepositories(owner, reponame string) (*gitea.Repository, error) {
	repo, resp, err := v.c.GetRepo(owner, reponame)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s, %w", reponame, err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get repository %s, %s", reponame, resp.Status)
	}

	return repo, nil
}

func (v *Gitea) GetMyUserInfo() (*gitea.User, error) {
	user, resp, err := v.c.GetMyUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info, %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get user, %s", resp.Status)
	}

	return user, nil
}

func (v *Gitea) ListIssues(owner string, page, pageSize int) ([]*gitea.Issue, error) {
	list, resp, err := v.c.ListIssues(gitea.ListIssueOption{
		ListOptions: gitea.ListOptions{
			Page:     page,
			PageSize: pageSize,
		},
		State: gitea.StateOpen,
		Owner: owner,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list issues, %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list issues, %s", resp.Status)
	}

	return list, nil
}

func (v *Gitea) GetCommitDiff(owner, repo, commitID string) ([]byte, error) {
	diff, resp, err := v.c.GetCommitDiff(owner, repo, commitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit diff, %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get commit diff, %s", resp.Status)
	}

	return diff, nil
}

func (v *Gitea) GetDiff(owner, repo, commitID string) (*CommitDiff, error) {
	commit, resp, err := v.c.GetSingleCommit(owner, repo, commitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit, %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get commit, %s", resp.Status)
	}

	diff, err := v.GetCommitDiff(owner, repo, commitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit diff, %w", err)
	}

	files := make([]string, 0)
	for _, file := range commit.Files {
		files = append(files, file.Filename)
	}

	commitDiff := &CommitDiff{
		CommitID:      commitID,
		CommitMessage: commit.RepoCommit.Message,
		Files:         files,
		DiffContent:   string(diff),
	}

	return commitDiff, nil
}

func (v *Gitea) GetFileContent(owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error) {
	fileContent, resp, err := v.c.GetFile(owner, repo, commitID, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content, %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get file content, %s", resp.Status)
	}

	lines := bytes.Split(fileContent, []byte("\n"))

	start := max(0, lineStart-lineCount)
	end := min(len(lines), lineStart+lineCount)

	content := bytes.Join(lines[start:end], []byte("\n"))

	flog.Info("get %d lines of content for %s (size: %d bytes)", len(bytes.Split(content, []byte("\n"))), filePath, len(content))

	return content, nil
}
