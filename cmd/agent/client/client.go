package client

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"
)

type flowbot struct {
	c           *resty.Client
	accessToken string
}

func newFlowbot() *flowbot {
	v := &flowbot{accessToken: config.App.ApiToken}

	v.c = resty.New()
	v.c.SetBaseURL(config.App.ApiUrl)
	v.c.SetTimeout(time.Minute)

	return v
}

func (v *flowbot) fetcher(action types.Action, content any) ([]byte, error) {
	resp, err := v.c.R().
		SetAuthToken(v.accessToken).
		SetResult(&protocol.Response{}).
		SetBody(map[string]any{
			"action":  action,
			"version": types.ApiVersion,
			"content": content,
		}).
		Post("/flowkit")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		r := resp.Result().(*protocol.Response)
		return jsoniter.Marshal(r.Data)
	} else {
		return nil, fmt.Errorf("%d", resp.StatusCode())
	}
}

func Bots() (*BotsResult, error) {
	v := newFlowbot()
	data, err := v.fetcher(types.Bots, nil)
	if err != nil {
		return nil, err
	}
	var r BotsResult
	err = jsoniter.Unmarshal(data, &r.Bots)
	if err != nil {
		return nil, err
	}
	return &r, err
}

type BotsResult struct {
	Bots []struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"bots"`
}

func Help() (*HelpResult, error) {
	v := newFlowbot()
	data, err := v.fetcher(types.Help, nil)
	if err != nil {
		return nil, err
	}
	var r HelpResult
	err = jsoniter.Unmarshal(data, &r.Bots)
	if err != nil {
		return nil, err
	}
	return &r, err
}

type HelpResult struct {
	Bots []struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"bots"`
}

func Pull() (*InstructResult, error) {
	v := newFlowbot()
	data, err := v.fetcher(types.Pull, nil)
	if err != nil {
		return nil, err
	}
	var r InstructResult
	err = jsoniter.Unmarshal(data, &r.Instruct)
	if err != nil {
		return nil, err
	}
	return &r, err
}

type InstructResult struct {
	Instruct []Instruct `json:"instruct"`
}

type Instruct struct {
	No       string `json:"no"`
	Bot      string `json:"bot"`
	Flag     string `json:"flag"`
	Content  any    `json:"content"`
	ExpireAt string `json:"expire_at"`
}

func Collect(content types.FlowkitData) (string, error) {
	v := newFlowbot()
	data, err := v.fetcher(types.Collect, content)
	if err != nil {
		return "", err
	}
	return string(data), err
}
