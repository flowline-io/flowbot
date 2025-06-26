package client

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"resty.dev/v3"
)

type flowbot struct {
	c           *resty.Client
	accessToken string
}

func newFlowbot() *flowbot {
	v := &flowbot{accessToken: config.App.Api.Token}

	v.c = resty.New()
	v.c.SetBaseURL(config.App.Api.Url)
	v.c.SetTimeout(time.Minute)
	v.c.SetDisableWarn(true)

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
		return nil, fmt.Errorf("request error: %w", err)
	}

	if resp.StatusCode() == http.StatusOK {
		r := resp.Result().(*protocol.Response)
		return sonic.Marshal(r.Data)
	} else {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode())
	}
}

func Pull() (*InstructResult, error) {
	v := newFlowbot()
	data, err := v.fetcher(types.PullAction, nil)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var r InstructResult
	err = sonic.Unmarshal(data, &r.Instruct)
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
	_, err := v.fetcher(types.CollectAction, types.KV{
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
	_, err := v.fetcher(types.AckAction, types.KV{
		"no": no,
	})
	if err != nil {
		return err
	}
	return err
}

func Online(hostid, hostname string) error {
	v := newFlowbot()
	_, err := v.fetcher(types.OnlineAction, types.KV{
		"hostid":   hostid,
		"hostname": hostname,
	})
	if err != nil {
		return err
	}
	return err
}

func Offline(hostid, hostname string) error {
	v := newFlowbot()
	_, err := v.fetcher(types.OfflineAction, types.KV{
		"hostid":   hostid,
		"hostname": hostname,
	})
	if err != nil {
		return err
	}
	return err
}

func Message(text string) error {
	v := newFlowbot()
	_, err := v.fetcher(types.MessageAction, types.KV{
		"message": text,
	})
	if err != nil {
		return err
	}
	return err
}
