package lobehub

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/go-resty/resty/v2"
)

const (
	ID = "slash"
)

type Lobehub struct {
	c *resty.Client
}

func NewLobehub() *Lobehub {
	v := &Lobehub{}
	v.c = resty.New()
	v.c.SetTimeout(time.Minute)

	return v
}

func (i *Lobehub) WebCrawler(url string) (*WebCrawlerResponse, error) {
	resp, err := i.c.R().
		SetResult(&WebCrawlerResponse{}).
		SetBody(map[string]any{
			"url": url,
		}).
		Post("https://web-crawler.chat-plugin.lobehub.com/api/v1")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*WebCrawlerResponse)
		return result, nil
	} else {
		return nil, fmt.Errorf("%d, %s", resp.StatusCode(), utils.BytesToString(resp.Body()))
	}
}
