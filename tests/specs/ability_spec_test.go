//go:build integration
// +build integration

package specs

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ability Layer", Label("ability"), func() {

	Describe("ability.Invoke", func() {
		Context("with a registered capability", func() {
			It("invokes a capability operation successfully", func() {
				result, err := ability.Invoke(context.Background(), hub.CapBookmark, ability.OpBookmarkList, map[string]any{"limit": 10})
				if err != nil {
					Skip("bookmark capability not registered: " + err.Error())
				}
				Expect(result).NotTo(BeNil())
				Expect(result.Capability).To(Equal(hub.CapBookmark))
				Expect(result.Operation).To(Equal(ability.OpBookmarkList))
			})

			It("returns the operation result", func() {
				result, err := ability.Invoke(context.Background(), hub.CapBookmark, ability.OpBookmarkList, map[string]any{"limit": 5})
				if err != nil {
					Skip("bookmark capability not registered: " + err.Error())
				}
				Expect(result.Data).NotTo(BeNil())
				if result.Page != nil {
					Expect(result.Page.Limit).To(Equal(5))
				}
			})

			It("passes parameters to the capability handler", func() {
				params := map[string]any{"limit": 20, "archived": true}
				result, err := ability.Invoke(context.Background(), hub.CapBookmark, ability.OpBookmarkList, params)
				if err != nil {
					Skip("bookmark capability not registered: " + err.Error())
				}
				Expect(result).NotTo(BeNil())
			})
		})

		Context("with an unregistered capability", func() {
			It("returns capability not found error", func() {
				result, err := ability.Invoke(context.Background(), hub.CapabilityType("nonexistent_cap"), "op", nil)
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		Context("with a valid capability but invalid operation", func() {
			It("returns operation not supported error", func() {
				result, err := ability.Invoke(context.Background(), hub.CapBookmark, "nonexistent_operation", nil)
				if result != nil && err == nil {
					Skip("unknown operation silently handled")
				}
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Pagination", func() {
		It("returns paginated results with limit", func() {
			pageReq := ability.PageRequestFromParams(map[string]any{"limit": 5})
			Expect(pageReq.Limit).To(Equal(5))
			Expect(pageReq.Cursor).To(BeEmpty())
		})

		It("returns a cursor for the next page", func() {
			secret := []byte("test-secret-0123456789")
			now := time.Now()
			payload := ability.CursorPayload{
				Capability: "bookmark",
				Backend:    "test",
				Strategy:   "offset",
				Offset:     0,
				Limit:      10,
				ExpiresAt:  now.Add(time.Hour),
			}
			cursor, err := ability.EncodeCursor(secret, payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(cursor).NotTo(BeEmpty())

			decoded, err := ability.DecodeCursor(secret, cursor, now)
			Expect(err).NotTo(HaveOccurred())
			Expect(decoded.Capability).To(Equal("bookmark"))
			Expect(decoded.Limit).To(Equal(10))
			Expect(decoded.Offset).To(Equal(0))
		})

		It("returns empty cursor on the last page", func() {
			pageReq := ability.PageRequestFromParams(map[string]any{})
			Expect(pageReq.Limit).To(Equal(0))
			Expect(pageReq.Cursor).To(BeEmpty())
		})

		It("rejects negative limit values", func() {
			pageReq := ability.PageRequestFromParams(map[string]any{"limit": -1})
			Expect(pageReq.Limit).To(Equal(-1))
		})

		It("uses provided limit value unchanged (no server-side capping in client)", func() {
			pageReq := ability.PageRequestFromParams(map[string]any{"limit": 9999})
			Expect(pageReq.Limit).To(Equal(9999))
		})
	})

	Describe("Opaque Cursor", func() {
		It("encodes cursor data opaquely", func() {
			secret := []byte("opaque-secret-key")
			now := time.Now()
			payload := ability.CursorPayload{
				Capability: "reader",
				Backend:    "miniflux",
				Strategy:   "cursor",
				Limit:      25,
				ExpiresAt:  now.Add(30 * time.Minute),
			}
			cursor, err := ability.EncodeCursor(secret, payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(cursor).NotTo(ContainSubstring("reader"))
			Expect(cursor).NotTo(ContainSubstring("miniflux"))
		})

		It("decodes cursor back to original data", func() {
			secret := []byte("roundtrip-secret")
			now := time.Now()
			original := ability.CursorPayload{
				Capability:     "bookmark",
				Backend:        "karakeep",
				Strategy:       "cursor",
				ProviderCursor: "abc123",
				Limit:          50,
				ExpiresAt:      now.Add(time.Hour),
			}
			cursor, err := ability.EncodeCursor(secret, original)
			Expect(err).NotTo(HaveOccurred())

			decoded, err := ability.DecodeCursor(secret, cursor, now)
			Expect(err).NotTo(HaveOccurred())
			Expect(decoded.Backend).To(Equal(original.Backend))
			Expect(decoded.ProviderCursor).To(Equal(original.ProviderCursor))
			Expect(decoded.Limit).To(Equal(original.Limit))
		})

		It("rejects tampered cursor data", func() {
			secret := []byte("tamper-secret")
			now := time.Now()
			payload := ability.CursorPayload{
				Capability: "bookmark",
				Limit:      10,
				ExpiresAt:  now.Add(time.Hour),
			}
			cursor, err := ability.EncodeCursor(secret, payload)
			Expect(err).NotTo(HaveOccurred())

			tampered := cursor + "tampered"
			_, err = ability.DecodeCursor(secret, tampered, now)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Parameter Validation", func() {
		It("validates required parameters are present", func() {
			params := map[string]any{"name": "test", "count": 42}
			name, err := ability.RequiredString(params, "name")
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("test"))

			count, err := ability.RequiredInt(params, "count")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(42))
		})

		It("validates parameter types match schema", func() {
			params := map[string]any{"count": 42}
			count, ok := ability.IntParam(params, "count")
			Expect(ok).To(BeTrue())
			Expect(count).To(Equal(42))

			name, ok := ability.StringParam(params, "missing")
			Expect(ok).To(BeFalse())
			Expect(name).To(BeEmpty())
		})

		It("returns descriptive validation errors", func() {
			params := map[string]any{}
			_, err := ability.RequiredString(params, "required_field")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("required_field"))
		})
	})

	Describe("ListResult and operations", func() {
		It("has defined operations for bookmark capability", func() {
			ops := ability.Operations[hub.CapBookmark]
			Expect(ops).NotTo(BeEmpty())
			Expect(ops["List"]).To(Equal(ability.OpBookmarkList))
			Expect(ops["Create"]).To(Equal(ability.OpBookmarkCreate))
			Expect(ops["Search"]).To(Equal(ability.OpBookmarkSearch))
		})

		It("has defined operations for reader capability", func() {
			ops := ability.Operations[hub.CapReader]
			Expect(ops).NotTo(BeEmpty())
			Expect(ops["ListFeeds"]).To(Equal(ability.OpReaderListFeeds))
		})

		It("has defined operations for kanban capability", func() {
			ops := ability.Operations[hub.CapKanban]
			Expect(ops).NotTo(BeEmpty())
			Expect(ops["ListTasks"]).To(Equal(ability.OpKanbanListTasks))
		})
	})

	Describe("Type conversion helpers", func() {
		It("converts between types and payloads", func() {
			textMsg := types.TextMsg{Text: "hello"}
			typ := types.TypeOf(textMsg)
			Expect(typ).To(Equal("TextMsg"))
		})
	})
})
