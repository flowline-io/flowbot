package oneai

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	ID     = "oneai"
	ApiKey = "api_key"
)

type OneAI struct {
	c      *resty.Client
	apiKey string // ConsumerKey
}

func NewOneAI(apiKey string) *OneAI {
	v := &OneAI{apiKey: apiKey}

	v.c = resty.New()
	v.c.SetBaseURL("https://api.oneai.com")
	v.c.SetTimeout(time.Minute)
	v.c.SetHeader("accept", "application/json")
	v.c.SetHeader("Content-Type", "application/json")

	return v
}

func (v *OneAI) Summarize(url string) (*Response, error) {
	resp, err := v.c.R().
		SetResult(&Response{}).
		SetHeader("api-key", v.apiKey).
		SetBody(map[string]interface{}{
			"input":       url,
			"input_type":  "article",
			"output_type": "json",
			"multilingual": map[string]interface{}{
				"enabled": true,
			},
			"steps": []map[string]interface{}{
				{
					"skill": "html-extract-article",
				},
				{
					"skill": "summarize",
					"params": map[string]interface{}{
						"auto_length":  true,
						"find_origins": true,
					},
				},
			},
		}).
		Post("/api/v0/pipeline")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*Response), nil
	} else {
		return nil, fmt.Errorf("%d", resp.StatusCode())
	}
}
