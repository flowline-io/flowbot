package miniflux

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	rssClient "miniflux.app/v2/client"
)

const (
	ID          = "miniflux"
	EndpointKey = "endpoint"
	ApikeyKey   = "api_key"
)

type Miniflux struct {
	apiKey string
	c      *rssClient.Client
}

func GetClient() *Miniflux {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	apiKey, _ := providers.GetConfig(ID, ApikeyKey)

	return NewMiniflux(endpoint.String(), apiKey.String())
}

func NewMiniflux(endpoint, apiKey string) *Miniflux {
	v := &Miniflux{apiKey: apiKey}
	v.c = rssClient.NewClient(endpoint, apiKey)

	return v
}

func (v *Miniflux) GetFeeds() (rssClient.Feeds, error) {
	list, err := v.c.Feeds()
	if err != nil {
		return nil, fmt.Errorf("failed to get feeds, %w", err)
	}

	return list, nil
}

func (v *Miniflux) GetEntries(filter *rssClient.Filter) (*rssClient.EntryResultSet, error) {
	list, err := v.c.Entries(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get entries, %w", err)
	}

	return list, nil
}

func (v *Miniflux) MarkAllAsRead() error {
	feeds, err := v.c.Feeds()
	if err != nil {
		return fmt.Errorf("failed to get feeds, %w", err)
	}

	for _, feed := range feeds {
		err = v.c.MarkFeedAsRead(feed.ID)
		if err != nil {
			flog.Warn("failed to mark feed as read, %v", err)
		}
	}

	return nil
}
