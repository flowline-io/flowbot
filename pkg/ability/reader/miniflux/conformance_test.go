package miniflux

import (
	"testing"
	"time"

	rdr "github.com/flowline-io/flowbot/pkg/ability/reader"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	rssClient "miniflux.app/v2/client"
)

func TestMinifluxConformance(t *testing.T) {
	conformance.RunReaderConformance(t, func(t *testing.T, cfg conformance.ReaderConfig) rdr.Service {
		c := &fakeClient{
			feeds:      cfgToFeeds(cfg),
			feedsErr:   cfg.FeedsErr,
			createFeedID: cfg.CreateFeedID,
			createFeedErr: cfg.CreateFeedErr,
			entries:       cfgToEntryResultSet(cfg),
			entriesErr:    cfg.EntriesErr,
			markReadErr:   cfg.MarkReadErr,
			markUnreadErr: cfg.MarkUnreadErr,
		}
		return NewWithClient(c)
	})
}

func cfgToFeeds(cfg conformance.ReaderConfig) rssClient.Feeds {
	feeds := make(rssClient.Feeds, 0, len(cfg.Feeds))
	for _, f := range cfg.Feeds {
		feeds = append(feeds, &rssClient.Feed{
			ID:      f.ID,
			Title:   f.Title,
			FeedURL: f.FeedURL,
			SiteURL: f.SiteURL,
		})
	}
	return feeds
}

func cfgToEntryResultSet(cfg conformance.ReaderConfig) *rssClient.EntryResultSet {
	if cfg.EntriesErr != nil || len(cfg.Entries) == 0 && cfg.EntriesTotal == 0 {
		return &rssClient.EntryResultSet{Total: int(cfg.EntriesTotal), Entries: rssClient.Entries{}}
	}
	entries := make(rssClient.Entries, 0, len(cfg.Entries))
	for _, e := range cfg.Entries {
		entries = append(entries, &rssClient.Entry{
			ID:      e.ID,
			Title:   e.Title,
			URL:     e.URL,
			Content: e.Content,
			Status:  e.Status,
			Starred: e.Starred,
			Date:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	}
	return &rssClient.EntryResultSet{Total: int(cfg.EntriesTotal), Entries: entries}
}

// Ensure conformance config helpers compile.
var _ = cfgToFeeds
var _ = cfgToEntryResultSet
