//go:build integration
// +build integration

package specs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

// viewPageAdapter satisfies store.Adapter for BDD view page tests,
// delegating GetDB to the shared EntClient and providing stub auth.
type viewPageAdapter struct {
	store.Adapter
	ent *gen.Client
	db  store.Adapter // original adapter to restore after tests
}

func (a *viewPageAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *viewPageAdapter) Close() error                     { return nil }
func (a *viewPageAdapter) IsOpen() bool                     { return true }
func (a *viewPageAdapter) GetName() string                  { return "bdd-view-page" }
func (a *viewPageAdapter) Stats() any                       { return nil }
func (a *viewPageAdapter) GetDB() any                       { return a.ent }

func (a *viewPageAdapter) ParameterGet(_ context.Context, _ string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:   1,
		Flag: "bdd-test",
		Params: map[string]any{
			"uid":    "testuser",
			"topic":  "test",
			"scopes": []string{"admin:*"},
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (a *viewPageAdapter) ParameterSet(_ context.Context, _ string, _ types.KV, _ time.Time) error {
	return nil
}

func (a *viewPageAdapter) ParameterDelete(_ context.Context, _ string) error {
	return nil
}

var _ = Describe("View Pages", Label("module", "web"), func() {
	var (
		origDB     store.Adapter
		pageStore  *store.PageDataStore
	)

	BeforeEach(func() {
		origDB = store.Database
		store.Database = &viewPageAdapter{ent: EntClient, db: origDB}
		pageStore = store.NewPageDataStore(EntClient)

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf) // ignore "already initialized" on subsequent calls
		webmod.MountForE2E(App)
	})

	AfterEach(func() {
		store.Database = origDB
	})

	Describe("POST /service/web/view", func() {
		Context("with a valid text payload", func() {
			It("returns 201 with token and URL", func() {
				body, _ := sonic.Marshal(types.KV{
					"type":  "text",
					"title": "Test Page",
					"data":  types.KV{"content": "hello world"},
				})
				req := JSONRequest(http.MethodPost, "/service/web/view", body)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				respBody := ReadBody(resp)
				Expect(string(respBody)).To(ContainSubstring(`"token"`))
				Expect(string(respBody)).To(ContainSubstring(`"url":"/service/web/view/`))
			})
		})

		Context("with missing type field", func() {
			It("returns 400 with error", func() {
				body, _ := sonic.Marshal(types.KV{
					"title": "NoType",
					"data":  types.KV{},
				})
				req := JSONRequest(http.MethodPost, "/service/web/view", body)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(string(ReadBody(resp))).To(ContainSubstring(`"error"`))
			})
		})

		Context("with invalid JSON", func() {
			It("returns 400 with error", func() {
				req := JSONRequest(http.MethodPost, "/service/web/view", []byte(`not-json`))
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(string(ReadBody(resp))).To(ContainSubstring(`"error"`))
			})
		})

		Context("with expires_at set", func() {
			It("returns 201 with token and URL", func() {
				body, _ := sonic.Marshal(types.KV{
					"type":       "text",
					"title":      "Timed Page",
					"data":       types.KV{"content": "timed"},
					"expires_at": "2099-01-01T00:00:00Z",
				})
				req := JSONRequest(http.MethodPost, "/service/web/view", body)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				Expect(string(ReadBody(resp))).To(ContainSubstring(`"url":"/service/web/view/`))
			})
		})
	})

	Describe("GET /service/web/view/:token", func() {
		Context("with a valid text page", func() {
			var token string

			BeforeEach(func() {
				token = "bdd-render-" + types.Id()
				err := pageStore.CreatePageData(context.Background(), token,
					"text", "Render Test", types.KV{"content": "Hello BDD"}, "testuser", nil)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_, _ = pageStore.DeletePageData(context.Background(), token)
			})

			It("renders the page with content", func() {
				req := MakeRequest(http.MethodGet, "/service/web/view/"+token, nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Hello BDD"))
			})
		})

		Context("with a non-existent token", func() {
			It("shows the expired page message", func() {
				req := MakeRequest(http.MethodGet, "/service/web/view/bdd-no-such-token", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(string(ReadBody(resp))).To(ContainSubstring("Page not found or expired"))
			})
		})

		Context("with an expired page", func() {
			var token string

			BeforeEach(func() {
				token = "bdd-expired-" + types.Id()
				oneHourAgo := time.Now().Add(-1 * time.Hour)
				err := pageStore.CreatePageData(context.Background(), token,
					"text", "Expired Page", types.KV{"content": "stale"}, "testuser", &oneHourAgo)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_, _ = pageStore.DeletePageData(context.Background(), token)
			})

			It("shows the expired page message", func() {
				req := MakeRequest(http.MethodGet, "/service/web/view/"+token, nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(string(ReadBody(resp))).To(ContainSubstring("Page not found or expired"))
			})
		})

		Context("without authentication", func() {
			It("redirects to login page", func() {
				req := MakeRequest(http.MethodGet, "/service/web/view/some-token", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
				Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
			})
		})
	})

	Describe("DELETE /service/web/view/:token", func() {
		Context("with an existing page", func() {
			var token string

			BeforeEach(func() {
				token = "bdd-delete-" + types.Id()
				err := pageStore.CreatePageData(context.Background(), token,
					"text", "Delete Me", types.KV{}, "testuser", nil)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_, _ = pageStore.DeletePageData(context.Background(), token)
			})

			It("returns 204 and page becomes unavailable", func() {
				req := MakeRequest(http.MethodDelete, "/service/web/view/"+token, nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

				// Verify page is gone
				stored, err := pageStore.GetPageDataByToken(context.Background(), token)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored).To(BeNil())
			})
		})

		Context("with a non-existent token", func() {
			It("returns 404", func() {
				req := MakeRequest(http.MethodDelete, "/service/web/view/bdd-no-such-token-for-delete", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-test-token"})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})
	})

	Describe("POST /service/web/login", func() {
		Context("with valid credentials", func() {
			It("authenticates and redirects to home", func() {
				formData := "username=admin&password=flowbot-dev-pass"
				req := MakeRequest(http.MethodPost, "/service/web/login", []byte(formData))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Or(
					Equal(http.StatusSeeOther),
					Equal(http.StatusOK),
				))
			})
		})

		Context("with invalid credentials", func() {
			It("returns login page with error", func() {
				formData := "username=wrong&password=wrong"
				req := MakeRequest(http.MethodPost, "/service/web/login", []byte(formData))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// login page renders HTML regardless
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(Or(
					ContainSubstring("login"),
					ContainSubstring("Login"),
				))
			})
		})
	})

	Describe("Direct PageData store operations", func() {
		It("creates and retrieves a page via store", func() {
			token := "bdd-store-" + types.Id()
			err := pageStore.CreatePageData(context.Background(), token,
				"text", "Store Test", types.KV{"content": "direct"}, "testuser", nil)
			Expect(err).NotTo(HaveOccurred())
			defer pageStore.DeletePageData(context.Background(), token)

			pageData, err := pageStore.GetPageDataByToken(context.Background(), token)
			Expect(err).NotTo(HaveOccurred())
			Expect(pageData).NotTo(BeNil())
			Expect(pageData.Title).To(Equal("Store Test"))
			Expect(pageData.Type).To(Equal("text"))
		})

		It("returns nil for non-existent token", func() {
			pageData, err := pageStore.GetPageDataByToken(context.Background(), "bdd-never-created")
			Expect(err).NotTo(HaveOccurred())
			Expect(pageData).To(BeNil())
		})

		It("deletes a page and verifies removal", func() {
			token := "bdd-store-del-" + types.Id()
			err := pageStore.CreatePageData(context.Background(), token,
				"text", "To Delete", types.KV{}, "testuser", nil)
			Expect(err).NotTo(HaveOccurred())

			affected, err := pageStore.DeletePageData(context.Background(), token)
			Expect(err).NotTo(HaveOccurred())
			Expect(affected).To(Equal(1))

			pageData, err := pageStore.GetPageDataByToken(context.Background(), token)
			Expect(err).NotTo(HaveOccurred())
			Expect(pageData).To(BeNil())
		})

		It("returns 0 affected for deleting non-existent token", func() {
			affected, err := pageStore.DeletePageData(context.Background(), "bdd-never-existed")
			Expect(err).NotTo(HaveOccurred())
			Expect(affected).To(Equal(0))
		})
	})
})
