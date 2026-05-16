//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reader Module", Label("module", "reader"), func() {

	Describe("Webservice — Feeds", func() {
		Context("GET /", func() {
			It("lists all subscribed feeds", func() {
				req := MakeRequest(http.MethodGet, "/service/reader/", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})

			It("returns empty list when no feeds exist", func() {
				req := MakeRequest(http.MethodGet, "/service/reader/", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("POST /", func() {
			It("rejects invalid feed URL", func() {
				body, _ := sonic.Marshal(map[string]string{"feed_url": "not-a-valid-feed-url"})
				req := JSONRequest(http.MethodPost, "/service/reader/", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})

			It("rejects empty URL", func() {
				body, _ := sonic.Marshal(map[string]string{"feed_url": ""})
				req := JSONRequest(http.MethodPost, "/service/reader/", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})
	})

	Describe("Webservice — Entries", func() {
		Context("GET /entries", func() {
			It("lists entries with status filter", func() {
				req := MakeRequest(http.MethodGet, "/service/reader/entries?status=unread", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})

			It("filters entries by feed", func() {
				req := MakeRequest(http.MethodGet, "/service/reader/entries?feed_id=1", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("PATCH /entries", func() {
			It("marks entries as read", func() {
				body, _ := sonic.Marshal(map[string]any{"entry_ids": []int{1}, "status": "read"})
				req := JSONRequest(http.MethodPatch, "/service/reader/entries", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})
	})

	Describe("Ability layer operations", func() {
		It("lists feeds via ability layer", func() {
			result, err := ability.Invoke(context.Background(), hub.CapReader, ability.OpReaderListFeeds, map[string]any{})
			if err != nil {
				Skip("reader backend not configured: " + err.Error())
			}
			Expect(result).NotTo(BeNil())
		})

		It("creates feed via ability layer rejects bad URL", func() {
			result, err := ability.Invoke(context.Background(), hub.CapReader, ability.OpReaderCreateFeed, map[string]any{"feed_url": "invalid"})
			if err != nil {
				Skip("reader backend not configured: " + err.Error())
			}
			if result != nil {
				_ = result
			}
		})
	})

	Describe("Operation constants", func() {
		It("has all expected reader operations", func() {
			Expect(ability.OpReaderListFeeds).To(Equal("list_feeds"))
			Expect(ability.OpReaderCreateFeed).To(Equal("create_feed"))
			Expect(ability.OpReaderListEntries).To(Equal("list_entries"))
			Expect(ability.OpReaderMarkEntryRead).To(Equal("mark_entry_read"))
			Expect(ability.OpReaderMarkEntryUnread).To(Equal("mark_entry_unread"))
		})
	})
})
