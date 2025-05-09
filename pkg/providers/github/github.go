package github

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
)

const (
	ID              = "github"
	ClientIdKey     = "id"
	ClientSecretKey = "secret"
)

type Github struct {
	c            *resty.Client
	clientId     string
	clientSecret string
	redirectURI  string
	accessToken  string
}

func NewGithub(clientId, clientSecret, redirectURI, accessToken string) *Github {
	v := &Github{clientId: clientId, clientSecret: clientSecret, redirectURI: redirectURI, accessToken: accessToken}

	v.c = resty.New()
	v.c.SetBaseURL("https://api.github.com")
	v.c.SetTimeout(time.Minute)

	return v
}

func (v *Github) GetAuthorizeURL() string {
	return fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=repo", v.clientId, v.redirectURI)
}

func (v *Github) completeAuth(code string) (interface{}, error) {
	resp, err := v.c.R().
		SetResult(&TokenResponse{}).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetBody(map[string]interface{}{
			"client_id":     v.clientId,
			"client_secret": v.clientSecret,
			"code":          code,
		}).
		Post("https://github.com/login/oauth/access_token")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*TokenResponse)
		v.accessToken = result.AccessToken
		return result, nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) Redirect(_ *http.Request) (string, error) {
	appRedirectURI := v.GetAuthorizeURL()
	return appRedirectURI, nil
}

func (v *Github) GetAccessToken(ctx *fiber.Ctx) (types.KV, error) {
	code := ctx.Query("code")
	tokenResp, err := v.completeAuth(code)
	if err != nil {
		return nil, err
	}

	extra, err := sonic.Marshal(&tokenResp)
	if err != nil {
		return nil, err
	}
	return types.KV{
		"name":  ID,
		"type":  ID,
		"token": v.accessToken,
		"extra": extra,
	}, nil
}

func (v *Github) GetAuthenticatedUser() (*User, error) {
	resp, err := v.c.R().
		SetResult(&User{}).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		Get("/user")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*User), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) GetUser(username string) (*User, error) {
	resp, err := v.c.R().
		SetResult(&User{}).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		Get(fmt.Sprintf("/users/%s", username))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*User), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) GetStarred(username string) (*[]Repository, error) {
	resp, err := v.c.R().
		SetResult(&[]Repository{}).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		Get(fmt.Sprintf("/users/%s/starred", username))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*[]Repository), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) GetFollowers() (*[]User, error) {
	resp, err := v.c.R().
		SetResult(&[]User{}).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		Get("/user/followers")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*[]User), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) CreateIssue(owner, repo string, issue Issue) (*Issue, error) {
	resp, err := v.c.R().
		SetResult(&Issue{}).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		SetBody(issue).
		Post(fmt.Sprintf("/repos/%s/%s/issues", owner, repo))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusCreated {
		return resp.Result().(*Issue), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) GetUserProjects(username string) (*[]Project, error) {
	resp, err := v.c.R().
		SetResult(&[]Project{}).
		SetHeader("Accept", "application/vnd.github.inertia-preview+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		Get(fmt.Sprintf("/users/%s/projects", username))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*[]Project), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) GetProjectColumns(projectID int64) (*[]ProjectColumn, error) {
	resp, err := v.c.R().
		SetResult(&[]ProjectColumn{}).
		SetHeader("Accept", "application/vnd.github.inertia-preview+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		Get(fmt.Sprintf("/projects/%d/columns", projectID))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*[]ProjectColumn), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) CreateCard(columnID int64, card ProjectCard) (*ProjectCard, error) {
	resp, err := v.c.R().
		SetResult(&ProjectCard{}).
		SetHeader("Accept", "application/vnd.github.inertia-preview+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		SetBody(card).
		Post(fmt.Sprintf("/projects/columns/%d/cards", columnID))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusCreated {
		return resp.Result().(*ProjectCard), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

func (v *Github) GetRepository(owner, repo string) (*Repository, error) {
	resp, err := v.c.R().
		SetResult(&Repository{}).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		Get(fmt.Sprintf("/repos/%s/%s", owner, repo))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*Repository), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}

// get user notifications
func (v *Github) GetNotifications() (*[]Notification, error) {
	resp, err := v.c.R().
		SetResult(&[]Notification{}).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetHeader("Authorization", fmt.Sprintf("token %s", v.accessToken)).
		Get("/notifications")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*[]Notification), nil
	} else {
		return nil, fmt.Errorf("%d, %s (%s)", resp.StatusCode(), resp.Header().Get("X-Error-Code"), resp.Header().Get("X-Error"))
	}
}
