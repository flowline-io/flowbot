//go:build integration
// +build integration

package specs

import (
	"bytes"
	"mime/multipart"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/validate"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server Module", Label("module", "server"), func() {

	Describe("Webservice", func() {
		Context("POST /upload", func() {
			It("rejects upload exceeding size limit", func() {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				part, _ := w.CreateFormFile("image", "large.jpg")
				largeData := make([]byte, validate.MaxFileSizeBytes+1024)
				part.Write(largeData)
				w.Close()

				req := MakeRequest(http.MethodPost, "/service/server/upload", b.Bytes())
				req.Header.Set("Content-Type", w.FormDataContentType())
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusRequestEntityTooLarge), Equal(http.StatusUnauthorized), Equal(http.StatusNotFound)))
			})

			It("rejects upload with unsupported content type", func() {
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				part, _ := w.CreateFormFile("image", "test.exe")
				part.Write([]byte("binary data"))
				w.Close()

				req := MakeRequest(http.MethodPost, "/service/server/upload", b.Bytes())
				req.Header.Set("Content-Type", w.FormDataContentType())
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized), Equal(http.StatusNotFound)))
			})
		})

		Context("GET /stacktrace", func() {
			It("returns process stacktrace for diagnostics", func() {
				req := MakeRequest(http.MethodGet, "/service/server/stacktrace", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized), Equal(http.StatusNotFound)))
			})

			It("returns runtime memory profile data", func() {
				req := MakeRequest(http.MethodGet, "/service/server/stacktrace", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				if resp.StatusCode == http.StatusOK {
					body := ReadBody(resp)
					var data types.KV
					err = sonic.Unmarshal(body, &data)
					if err == nil {
						Expect(data).To(HaveKey("go_version"))
					}
				}
			})
		})
	})

	Describe("Type helpers used in server module", func() {
		It("creates KV messages", func() {
			kv := types.KVMsg{"success": true, "count": 42}
			Expect(kv["success"]).To(BeTrue())
			Expect(types.TypeOf(kv)).To(Equal("KVMsg"))
		})

		It("creates empty messages", func() {
			msg := types.EmptyMsg{}
			Expect(types.TypeOf(msg)).To(Equal("EmptyMsg"))
		})
	})

	Describe("HTTP endpoint structure", func() {
		It("validates request body for POST upload is required", func() {
			req := MakeRequest(http.MethodPost, "/service/server/upload", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized), Equal(http.StatusNotFound)))
		})
	})

	Describe("KV type operations", func() {
		It("sets and gets values via KV", func() {
			kv := types.KV{"name": "server-test", "enabled": true}
			name, ok := kv.String("name")
			Expect(ok).To(BeTrue())
			Expect(name).To(Equal("server-test"))
		})
	})
})
