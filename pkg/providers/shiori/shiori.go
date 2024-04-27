package shiori

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"time"
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

type LoginResponse struct {
	Session string `json:"session"`
	Account struct {
		Id       int    `json:"id"`
		Username string `json:"username"`
		Owner    bool   `json:"owner"`
	} `json:"account"`
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

type BookmarksResponse struct {
	Bookmarks []struct {
		Id         int    `json:"id"`
		Url        string `json:"url"`
		Title      string `json:"title"`
		Excerpt    string `json:"excerpt"`
		Author     string `json:"author"`
		Public     int    `json:"public"`
		Modified   string `json:"modified"`
		ImageURL   string `json:"imageURL"`
		HasContent bool   `json:"hasContent"`
		HasArchive bool   `json:"hasArchive"`
		Tags       []struct {
			Id   int    `json:"id"`
			Name string `json:"name"`
		} `json:"tags"`
		CreateArchive bool `json:"createArchive"`
	} `json:"bookmarks"`
	MaxPage int `json:"maxPage"`
	Page    int `json:"page"`
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

type BookmarkResponse struct {
	Id         int    `json:"id"`
	Url        string `json:"url"`
	Title      string `json:"title"`
	Excerpt    string `json:"excerpt"`
	Author     string `json:"author"`
	Public     int    `json:"public"`
	Modified   string `json:"modified"`
	Html       string `json:"html"`
	ImageURL   string `json:"imageURL"`
	HasContent bool   `json:"hasContent"`
	HasArchive bool   `json:"hasArchive"`
	Tags       []struct {
		Name string `json:"name"`
	} `json:"tags"`
	CreateArchive bool `json:"createArchive"`
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
