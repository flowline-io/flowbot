//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Bookmark Module", Label("module", "bookmark"), func() {

	Describe("Webservice — bookmark CRUD", func() {
		Context("GET /", func() {
			It("returns paginated bookmark list")
			It("returns empty list when no bookmarks exist")
			It("filters bookmarks by tag")
			It("filters bookmarks by search query")
			It("supports cursor-based pagination")
		})

		Context("GET /:id", func() {
			It("returns a single bookmark by ID")
			It("returns 404 for non-existent bookmark")
		})

		Context("POST /", func() {
			It("creates a bookmark with a valid URL")
			It("rejects bookmark with empty URL")
			It("rejects bookmark with invalid URL format")
			It("deduplicates — rejects already bookmarked URL")
		})

		Context("PATCH /:id", func() {
			It("archives a bookmark to ArchiveBox")
			It("returns error when archiving non-existent bookmark")
			It("returns error when ArchiveBox is unreachable")
		})

		Context("POST /:id/tags", func() {
			It("attaches one or more tags to a bookmark")
			It("rejects duplicate tags")
			It("returns error for non-existent bookmark")
		})

		Context("DELETE /:id/tags", func() {
			It("detaches a tag from a bookmark")
			It("is idempotent — removing non-attached tag succeeds")
		})

		Context("GET /check-url", func() {
			It("returns true when URL is already bookmarked")
			It("returns false when URL is not bookmarked")
		})

		Context("GET /search", func() {
			It("searches bookmarks by query string")
			It("returns empty results for unmatched query")
		})
	})

	Describe("Command", func() {
		It("bookmark list — returns the newest 10 bookmarks")
		It("bookmark list — formats output for chat display")
	})

	Describe("Cron Jobs", func() {
		It("bookmarks_tag — auto-tags untagged bookmarks via LLM")
		It("bookmarks_tag_merge — merges similar tags via LLM")
		It("bookmarks_metrics — updates bookmark counter stats")
		It("bookmarks_search — indexes bookmarks to search engine")
		It("bookmarks_task — creates kanban tasks for new bookmarks")
	})

	Describe("Event Handlers", func() {
		It("BookmarkArchiveBotEventID — archives bookmark and sends notification")
		It("BookmarkCreateBotEventID — creates bookmark and sends notification")
		It("ArchiveBoxAddBotEventID — archives URL to ArchiveBox and notifies")
	})
})
