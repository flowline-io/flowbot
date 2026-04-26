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

func (v *Miniflux) GetFeed(feedID int64) (*rssClient.Feed, error) {
	feed, err := v.c.Feed(feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed, %w", err)
	}
	return feed, nil
}

func (v *Miniflux) CreateFeed(req *rssClient.FeedCreationRequest) (int64, error) {
	feedID, err := v.c.CreateFeed(req)
	if err != nil {
		return 0, fmt.Errorf("failed to create feed, %w", err)
	}
	return feedID, nil
}

func (v *Miniflux) UpdateFeed(feedID int64, req *rssClient.FeedModificationRequest) (*rssClient.Feed, error) {
	feed, err := v.c.UpdateFeed(feedID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update feed, %w", err)
	}
	return feed, nil
}

func (v *Miniflux) RefreshFeed(feedID int64) error {
	err := v.c.RefreshFeed(feedID)
	if err != nil {
		return fmt.Errorf("failed to refresh feed, %w", err)
	}
	return nil
}

func (v *Miniflux) UpdateEntries(entryIDs []int64, status string) error {
	err := v.c.UpdateEntries(entryIDs, status)
	if err != nil {
		return fmt.Errorf("failed to update entries, %w", err)
	}
	return nil
}

func (v *Miniflux) GetFeedEntries(feedID int64, filter *rssClient.Filter) (*rssClient.EntryResultSet, error) {
	entries, err := v.c.FeedEntries(feedID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed entries, %w", err)
	}
	return entries, nil
}
