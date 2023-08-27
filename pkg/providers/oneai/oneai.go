package oneai

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"net/http"
	"time"
)

const (
	ID     = "oneai"
	ApiKey = "api_key"
)

type Response struct {
	InputText interface{} `json:"input_text"`
	Input     []struct {
		Utterance string `json:"utterance"`
	} `json:"input"`
	Status string      `json:"status"`
	Error  interface{} `json:"error"`
	Output []struct {
		TextGeneratedByStepName string      `json:"text_generated_by_step_name"`
		TextGeneratedByStepID   int         `json:"text_generated_by_step_id"`
		Text                    interface{} `json:"text"`
		Contents                []struct {
			Utterance string `json:"utterance"`
		} `json:"contents"`
	} `json:"output"`
	Warnings interface{} `json:"warnings"`
	Stats    struct {
		ConcurrencyWaitTime interface{} `json:"concurrency_wait_time"`
		TotalRunningJobs    int         `json:"total_running_jobs"`
		TotalWaitingJobs    int         `json:"total_waiting_jobs"`
	} `json:"stats"`
}

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
