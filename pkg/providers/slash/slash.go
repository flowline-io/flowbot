package slash

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/providers"
	"resty.dev/v3"
)

const (
	ID          = "slash"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type Slash struct {
	c     *resty.Client
	token string
}

func GetClient() *Slash {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)

	return NewSlash(endpoint.String(), token.String())
}

func NewSlash(endpoint string, token string) *Slash {
	v := &Slash{token: token}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)

	return v
}

func (i *Slash) CreateShortcut(item Shortcut) error {
	resp, err := i.c.R().SetAuthToken(i.token).
		SetBody(item).
		Post("/api/v1/shortcuts")
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to create slash shortcut: %s", resp.String())
	}
	return nil
}

func (i *Slash) UpdateShortcut(item Shortcut) error {
	resp, err := i.c.R().SetAuthToken(i.token).
		SetBody(item).
		Put(fmt.Sprintf("/api/v1/shortcuts/%d", item.Id))
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to update slash shortcut: %s", resp.String())
	}
	return nil
}

func (i *Slash) DeleteShortcut(id int32) error {
	resp, err := i.c.R().SetAuthToken(i.token).
		Delete(fmt.Sprintf("/api/v1/shortcuts/%d", id))
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to delete slash shortcut: %s", resp.String())
	}
	return nil
}

func (i *Slash) GetShortcut(id int32) (*Shortcut, error) {
	resp, err := i.c.R().SetAuthToken(i.token).
		SetResult(&Shortcut{}).
		Get(fmt.Sprintf("/api/v1/shortcuts/%d", id))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get slash shortcut: %s", resp.String())
	}
	return resp.Result().(*Shortcut), nil
}

func (i *Slash) ListShortcuts() ([]*Shortcut, error) {
	resp, err := i.c.R().SetAuthToken(i.token).
		SetResult([]*Shortcut{}).
		Get("/api/v1/shortcuts")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to list slash shortcut: %s", resp.String())
	}
	return resp.Result().([]*Shortcut), nil
}
