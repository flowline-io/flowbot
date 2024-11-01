package bot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/go-resty/resty/v2"
)

const (
	StatsAgentID  = "stats_agent"
	ReviewAgentID = "review_agent"
)

func AnkiStats() {
	html, err := getCollectionStatsHTML()
	if err != nil {
		flog.Error(err)
		return
	}
	_, err = client.Agent(types.FlowkitData{
		//Id:      StatsAgentID,
		Version: types.ApiVersion,
		Content: map[string]any{
			"html": html,
		},
	})
	if err != nil {
		flog.Error(err)
	}
}

func AnkiReview() {
	num, err := getNumCardsReviewedToday()
	if err != nil {
		flog.Error(err)
		return
	}
	_, err = client.Agent(types.FlowkitData{
		//Id:      ReviewAgentID,
		Version: types.ApiVersion,
		Content: map[string]any{
			"num": num,
		},
	})
	if err != nil {
		flog.Error(err)
	}
}

func getCollectionStatsHTML() (string, error) {
	c := resty.New()
	resp, err := c.R().
		SetContext(context.Background()).
		SetBody(Param{
			Action:  "getCollectionStatsHTML",
			Version: types.ApiVersion,
			Params: map[string]any{
				"wholeCollection": true,
			},
		}).
		SetResult(&Response{}).
		Post(ApiURI)
	if err != nil {
		return "", err
	}

	if resp.StatusCode() == http.StatusOK {
		respResult := resp.Result().(*Response)
		if respResult != nil {
			if respResult.Error != nil {
				return "", errors.New(*respResult.Error)
			}

			return string(respResult.Result), nil
		}
	}
	return "", errors.New("result error")
}

func getNumCardsReviewedToday() (int, error) {
	c := resty.New()
	resp, err := c.R().
		SetContext(context.Background()).
		SetBody(Param{
			Action:  "getNumCardsReviewedToday",
			Version: types.ApiVersion,
		}).
		SetResult(&Response{}).
		Post(ApiURI)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode() == http.StatusOK {
		respResult := resp.Result().(*Response)
		if respResult != nil {
			if respResult.Error != nil {
				return 0, errors.New(*respResult.Error)
			}

			n, _ := strconv.Atoi(string(respResult.Result))
			return n, nil
		}
	}
	return 0, errors.New("result error")
}

const ApiVersion = 6
const ApiURI = "http://localhost:8765"

type Param struct {
	Action  string `json:"action"`
	Version int    `json:"version"`
	Params  any    `json:"params,omitempty"`
}

type Response struct {
	Error  *string         `json:"error"`
	Result json.RawMessage `json:"result"`
}
