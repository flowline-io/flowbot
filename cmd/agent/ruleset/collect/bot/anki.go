package bot

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"net/http"
	"resty.dev/v3"
	"strconv"
)

const (
	StatsCollectID  = "stats_collect"
	ReviewCollectID = "review_collect"
)

func AnkiStats() {
	if !checkAnkiApiAvailable() {
		flog.Debug("anki api not available")
		return
	}

	html, err := getCollectionStatsHTML()
	if err != nil {
		flog.Error(err)
		return
	}
	err = client.Collect(types.CollectData{
		Id: StatsCollectID,
		Content: map[string]any{
			"html": html,
		},
	})
	if err != nil {
		flog.Error(err)
	}
}

func AnkiReview() {
	if !checkAnkiApiAvailable() {
		flog.Debug("anki api not available")
		return
	}

	num, err := getNumCardsReviewedToday()
	if err != nil {
		flog.Error(err)
		return
	}
	err = client.Collect(types.CollectData{
		Id: ReviewCollectID,
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

func checkAnkiApiAvailable() bool {
	return utils.PortAvailable("8765")
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
