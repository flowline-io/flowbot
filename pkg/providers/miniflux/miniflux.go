package miniflux

import (
	"fmt"
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
			return fmt.Errorf("failed to mark feed as read, %w", err)
		}
	}

	return nil
}
