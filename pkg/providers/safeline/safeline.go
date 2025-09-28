package safeline

import (
	"context"
	"net/http"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
	"resty.dev/v3"
)

const (
	ID          = "safeline"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type SafeLine struct {
	c     *resty.Client
	token string
}

func GetClient() *SafeLine {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)

	return NewSafeLine(endpoint.String(), token.String())
}

func NewSafeLine(endpoint string, token string) *SafeLine {
	v := &SafeLine{token: token}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)

	return v
}

func (v *SafeLine) QPS(ctx context.Context) (*Response, error) {
	resp, err := v.c.R().
		SetContext(ctx).
		SetHeader("X-SLCE-API-TOKEN", v.token).
		SetResult(&Response{}).
		Get("/stat/qps")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*Response)
		return result, nil
	}
	return nil, nil
}
