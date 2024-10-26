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
