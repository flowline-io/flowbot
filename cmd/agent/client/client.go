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

func (v *flowbot) fetcher(action types.Action, content types.KV) ([]byte, error) {
	resp, err := v.c.R().
		SetAuthToken(v.accessToken).
		SetResult(&protocol.Response{}).
		SetBody(types.AgentData{
			Action:  action,
			Version: types.ApiVersion,
			Content: content,
		}).
		Post("/agent")
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

func Pull() (*InstructResult, error) {
	v := newFlowbot()
	data, err := v.fetcher(types.Pull, nil)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
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

func Collect(content types.CollectData) error {
	v := newFlowbot()
	_, err := v.fetcher(types.Collect, types.KV{
		"id":      content.Id,
		"content": content.Content,
	})
	if err != nil {
		return err
	}
	return err
}

func Ack(no string) error {
	v := newFlowbot()
	_, err := v.fetcher(types.Ack, types.KV{
		"no": no,
	})
	if err != nil {
		return err
	}
	return err
}

func Online(hostid, hostname string) error {
	v := newFlowbot()
	_, err := v.fetcher(types.Online, types.KV{
		"hostid":   hostid,
		"hostname": hostname,
	})
	if err != nil {
		return err
	}
	return err
}

func Offline(hostid string) error {
	v := newFlowbot()
	_, err := v.fetcher(types.Offline, types.KV{
		"hostid": hostid,
	})
	if err != nil {
		return err
	}
	return err
}
