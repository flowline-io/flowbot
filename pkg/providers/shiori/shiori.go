package shiori

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	ID          = "shiori"
	EndpointKey = "endpoint"
	UsernameKey = "username"
	PasswordKey = "password"
)

type Shiori struct {
	c         *resty.Client
	sessionId string
}

func NewShiori(endpoint string) *Shiori {
	v := &Shiori{}

	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)

	return v
}

func (v *Shiori) Login(username string, password string) (*LoginResponse, error) {
	resp, err := v.c.R().
		SetResult(&LoginResponse{}).
		SetBody(map[string]interface{}{
			"username": username,
			"password": password,
			"remember": false,
			"owner":    false,
		}).
		Post("/api/login")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), string(resp.Body()))
	}

	r := resp.Result().(*LoginResponse)
	v.sessionId = r.Session
	v.c.SetHeader("X-Session-Id", r.Session)

	return r, nil
}

func (v *Shiori) Logout(sessionId string) error {
	resp, err := v.c.R().
		SetHeader("X-Session-Id", sessionId).
		Post("/api/logout")
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), string(resp.Body()))
	}
	return nil
}

func (v *Shiori) GetBookmarks() (*BookmarksResponse, error) {
	resp, err := v.c.R().
		SetResult(&BookmarksResponse{}).
		Get("/api/bookmarks")
	if err != nil {
		return nil, err
	}
	return resp.Result().(*BookmarksResponse), nil
}

func (v *Shiori) AddBookmark(url, title string) (*BookmarkResponse, error) {
	resp, err := v.c.R().
		SetResult(&BookmarkResponse{}).
		SetBody(map[string]interface{}{
			"url":           url,
			"createArchive": false,
			"public":        0,
			"tags":          nil,
			"title":         title,
			"excerpt":       "",
		}).
		Post("/api/bookmarks")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Result().(*BookmarkResponse), nil
}

func (v *Shiori) DeleteBookmark(bookmarkIds []int) error {
	resp, err := v.c.R().
		SetBody(bookmarkIds).
		Delete("/api/bookmarks")
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}
	return nil
}
