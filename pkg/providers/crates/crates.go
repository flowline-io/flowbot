package crates

import (
	"fmt"
	"net/http"

	"github.com/flowline-io/flowbot/pkg/utils"
	"resty.dev/v3"
)

const (
	ID = "crates"
)

type Crates struct {
	c *resty.Client
}

func NewCrates() *Crates {
	v := &Crates{}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL("https://crates.io/api/v1")

	return v
}

func (v *Crates) Search(keyword string) (*SearchResponse, error) {
	resp, err := v.c.R().
		SetResult(&SearchResponse{}).
		SetQueryParams(map[string]string{
			"page":     "1",
			"per_page": "10",
			"q":        keyword,
		}).
		Get("/crates")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*SearchResponse), nil
	} else {
		return nil, fmt.Errorf("%d", resp.StatusCode())
	}
}

func (v *Crates) Info(crate string) (*InfoGenerated, error) {
	resp, err := v.c.R().
		SetResult(&InfoGenerated{}).
		Get(fmt.Sprintf("/crates/%s", crate))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*InfoGenerated), nil
	} else {
		return nil, fmt.Errorf("%d", resp.StatusCode())
	}
}
