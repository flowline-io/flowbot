package gitea

import (
	"code.gitea.io/sdk/gitea"
	"fmt"
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

func NewGitea(endpoint, token string) (*Gitea, error) {
	var err error
	v := &Gitea{token: token}
	v.c, err = gitea.NewClient(endpoint, gitea.SetToken(token), gitea.SetDebugMode())
	if err != nil {
		return nil, err
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
