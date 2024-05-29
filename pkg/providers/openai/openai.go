package openai

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	ID = "openai"

	SecretKey = "secret_key"
)

type OpenAI struct {
	c         *resty.Client
	secretKey string // ConsumerKey
}

func NewOpenAI(secretKey string) *OpenAI {
	v := &OpenAI{secretKey: secretKey}

	v.c = resty.New()
	v.c.SetBaseURL("https://api.openai.com")
	v.c.SetTimeout(time.Minute)
	v.c.SetAuthToken(secretKey)

	return v
}

func (v *OpenAI) Chat(text string) (*Response, error) {
	resp, err := v.c.R().
		SetResult(&Response{}).
		SetBody(map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": text,
				},
			},
			"model": "gpt-3.5-turbo",
		}).
		Post("/v1/chat/completions")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*Response), nil
	} else {
		return nil, fmt.Errorf("%d", resp.StatusCode())
	}
}
