package crates

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"net/http"
	"time"
)

const (
	ID = "crates"
)

type Crates struct {
	c *resty.Client
}

func NewCrates() *Crates {
	v := &Crates{}

	v.c = resty.New()
	v.c.SetBaseURL("https://crates.io/api/v1")
	v.c.SetTimeout(time.Minute)

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
