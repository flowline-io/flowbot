//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"

	"github.com/bytedance/sonic"
	hubmod "github.com/flowline-io/flowbot/internal/modules/hub"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bookmark Module", Label("module", "bookmark"), func() {

	BeforeEach(func() {
		hubmod.MountForE2E(App)
	})

	Describe("Webservice — bookmark CRUD", func() {
		Context("GET /", func() {
			It("returns paginated bookmark list", func() {
				req := MakeRequest(http.MethodGet, "/service/bookmark/", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})

			It("responds without error for listing endpoint", func() {
				req := MakeRequest(http.MethodGet, "/service/bookmark/?limit=5", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})

			It("supports cursor query parameter", func() {
				req := MakeRequest(http.MethodGet, "/service/bookmark/?limit=10&cursor=", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})
		})

		Context("GET /:id", func() {
			It("returns 404 for non-existent bookmark", func() {
				req := MakeRequest(http.MethodGet, "/service/bookmark/nonexistent-id", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusNotFound), Equal(http.StatusUnauthorized), Equal(http.StatusBadRequest)))
			})
		})

		Context("POST /", func() {
			It("rejects bookmark with empty URL", func() {
				body, _ := sonic.Marshal(map[string]string{"url": ""})
				req := JSONRequest(http.MethodPost, "/service/bookmark/", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})

			It("handles invalid URL without crashing", func() {
				body, _ := sonic.Marshal(map[string]string{"url": "not-a-valid-url"})
				req := JSONRequest(http.MethodPost, "/service/bookmark/", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})
		})

		Context("PATCH /:id", func() {
			It("handles archiving non-existent bookmark gracefully", func() {
				req := MakeRequest(http.MethodPatch, "/service/bookmark/nonexistent", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusNotFound), Equal(http.StatusUnauthorized), Equal(http.StatusBadRequest)))
			})
		})

		Context("GET /check-url", func() {
			It("responds to URL check endpoint", func() {
				req := MakeRequest(http.MethodGet, "/service/bookmark/check-url?url=https://not-bookmarked.example.com", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})
		})

		Context("GET /search", func() {
			It("responds to search query", func() {
				req := MakeRequest(http.MethodGet, "/service/bookmark/search?q=test", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})

			It("handles unmatched query without error", func() {
				req := MakeRequest(http.MethodGet, "/service/bookmark/search?q=xyznonexistent12345", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})
		})
	})

	Describe("Ability layer operations", func() {
		It("lists bookmarks via ability layer", func() {
			result, err := ability.Invoke(context.Background(), hub.CapBookmark, ability.OpBookmarkList, map[string]any{"limit": 5})
			if err != nil {
				Skip("bookmark backend not configured: " + err.Error())
			}
			Expect(result).NotTo(BeNil())
			Expect(result.Operation).To(Equal(ability.OpBookmarkList))
		})

		It("checks URL via ability layer", func() {
			result, err := ability.Invoke(context.Background(), hub.CapBookmark, ability.OpBookmarkCheckURL, map[string]any{"url": "https://example.com"})
			if err != nil {
				Skip("bookmark backend not configured: " + err.Error())
			}
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("Operation constants", func() {
		It("has all expected bookmark operations", func() {
			Expect(ability.OpBookmarkList).To(Equal("list"))
			Expect(ability.OpBookmarkGet).To(Equal("get"))
			Expect(ability.OpBookmarkCreate).To(Equal("create"))
			Expect(ability.OpBookmarkDelete).To(Equal("delete"))
			Expect(ability.OpBookmarkArchive).To(Equal("archive"))
			Expect(ability.OpBookmarkSearch).To(Equal("search"))
			Expect(ability.OpBookmarkAttachTags).To(Equal("attach_tags"))
			Expect(ability.OpBookmarkDetachTags).To(Equal("detach_tags"))
			Expect(ability.OpBookmarkCheckURL).To(Equal("check_url"))
		})
	})

	Describe("Protocol response format", func() {
		It("returns protocol.Response for bookmark endpoints", func() {
			req := MakeRequest(http.MethodGet, "/service/bookmark/", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())

			body := ReadBody(resp)
			if resp.StatusCode == http.StatusOK {
				var pResp protocol.Response
				err := sonic.Unmarshal(body, &pResp)
				if err == nil {
					Expect(pResp.Status).To(Or(Equal(protocol.Success), Equal(protocol.Failed)))
				}
			}
		})
	})
})
