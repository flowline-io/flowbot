package openai

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"net/http"
	"time"
)

const (
	ID = "openai"

	SecretKey = "secret_key"
)

type Response struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	} `json:"choices"`
}

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
