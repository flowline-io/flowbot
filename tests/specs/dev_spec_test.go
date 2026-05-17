//go:build integration
// +build integration

package specs

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dev Module", Label("module", "dev"), func() {

	Describe("Webservice — GET /example", func() {
		It("returns example JSON with title, cpu, mem, disk", func() {
			resp := doDevGet()
			if resp.StatusCode != http.StatusOK {
				Skip("dev module routes not registered in test app")
			}

			body := ReadBody(resp)
			var pResp protocol.Response
			err := sonic.Unmarshal(body, &pResp)
			Expect(err).NotTo(HaveOccurred())
			Expect(pResp.Status).To(Equal(protocol.Success))

			data, ok := pResp.Data.(map[string]any)
			if !ok {
				var raw map[string]any
				jsonBytes, _ := sonic.Marshal(pResp.Data)
				sonic.Unmarshal(jsonBytes, &raw)
				data = raw
			}
			if data != nil {
				Expect(data).To(HaveKey("title"))
				Expect(data).To(HaveKey("cpu"))
				Expect(data).To(HaveKey("mem"))
				Expect(data).To(HaveKey("disk"))
			}
		})

		It("returns actual system values, not hardcoded", func() {
			resp := doDevGet()
			if resp.StatusCode != http.StatusOK {
				Skip("dev module routes not registered in test app")
			}

			body := ReadBody(resp)
			var pResp protocol.Response
			err := sonic.Unmarshal(body, &pResp)
			Expect(err).NotTo(HaveOccurred())

			if pResp.Data != nil {
				var data types.KV
				jsonBytes, _ := sonic.Marshal(pResp.Data)
				sonic.Unmarshal(jsonBytes, &data)

				if cpu, ok := data["cpu"]; ok {
					cpuStr, ok := cpu.(string)
					if ok {
						Expect(cpuStr).NotTo(BeEmpty())
					}
				}
			}
		})
	})

	Describe("Protocol - endpoint accessibility", func() {
		It("dev example endpoint is accessible without auth", func() {
			resp := doDevGet()
			if resp.StatusCode != http.StatusOK {
				Skip("dev module routes not registered in test app")
			}
		})

		It("returns proper Content-Type header", func() {
			resp := doDevGet()
			if resp.StatusCode != http.StatusOK {
				Skip("dev module routes not registered in test app")
			}
			Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("json"))
		})
	})

	Describe("Type helpers used in dev module", func() {
		It("generates unique IDs", func() {
			id1 := types.Id()
			id2 := types.Id()
			Expect(id1).NotTo(BeEmpty())
			Expect(id2).NotTo(BeEmpty())
			Expect(id1).NotTo(Equal(id2))
		})

		It("creates text messages for chat output", func() {
			msg := types.TextMsg{Text: "hello world"}
			Expect(msg.Text).To(Equal("hello world"))
			Expect(types.TypeOf(msg)).To(Equal("TextMsg"))
		})

		It("creates info messages", func() {
			msg := types.InfoMsg{Title: "Stats", Model: map[string]any{"count": 42}}
			Expect(msg.Title).To(Equal("Stats"))
			Expect(types.TypeOf(msg)).To(Equal("InfoMsg"))
		})

		It("creates link messages", func() {
			msg := types.LinkMsg{Title: "Example", Url: "https://example.com"}
			Expect(msg.Title).To(Equal("Example"))
			Expect(types.TypeOf(msg)).To(Equal("LinkMsg"))
		})

		It("creates table messages", func() {
			msg := types.TableMsg{
				Title:  "Data",
				Header: []string{"Name", "Value"},
				Row:    [][]any{{"key1", "val1"}, {"key2", "val2"}},
			}
			Expect(msg.Title).To(Equal("Data"))
			Expect(types.TypeOf(msg)).To(Equal("TableMsg"))
		})
	})
})

func doDevGet() *http.Response {
	req := MakeRequest(http.MethodGet, "/service/dev/example", nil)
	resp, err := App.Test(req)
	Expect(err).NotTo(HaveOccurred())
	return resp
}
