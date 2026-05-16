//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Reader Module", Label("module", "reader"), func() {

	Describe("Webservice — Feeds", func() {
		Context("GET /", func() {
			It("lists all subscribed feeds")
			It("shows feed title, URL, and entry count")
			It("returns empty list when no feeds exist")
		})

		Context("POST /", func() {
			It("creates a new feed subscription")
			It("rejects invalid feed URL")
			It("rejects creation with duplicate URL")
		})
	})

	Describe("Webservice — Entries", func() {
		Context("GET /entries", func() {
			It("lists entries with pagination")
			It("filters entries by feed")
			It("filters entries by read/unread status")
			It("supports cursor-based pagination")
		})

		Context("PATCH /entries", func() {
			It("marks entries as read")
			It("marks entries as unread")
			It("batch-marks multiple entries at once")
		})
	})

	Describe("Webhook — Miniflux", func() {
		Context("NewEntriesEventType", func() {
			It("creates bookmarks for new feed entries")
			It("handles batches of new entries")
		})

		Context("SaveEntryEventType", func() {
			It("fires bookmark create bot event for saved entries")
		})
	})

	Describe("Command", func() {
		It("reader — shows reader provider ID and status")
	})

	Describe("Cron Jobs", func() {
		It("reader_metrics — collects entry statistics every minute")
		It("reader_daily_summary — generates AI news summary via LLM daily")
	})
})
