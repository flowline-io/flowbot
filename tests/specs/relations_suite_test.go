//go:build integration
// +build integration

package specs

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type relationsWebAdapter struct {
	store.Adapter
	ent    *gen.Client
	uid    string
	scopes []string
}

func (a *relationsWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *relationsWebAdapter) Close() error                     { return nil }
func (a *relationsWebAdapter) IsOpen() bool                     { return true }
func (a *relationsWebAdapter) GetName() string                  { return "bdd-relations-page" }
func (a *relationsWebAdapter) Stats() any                       { return nil }
func (a *relationsWebAdapter) GetDB() any                       { return a.ent }

func (a *relationsWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:    1,
		Flag:  flag,
		Params: map[string]any{
			"uid":    a.uid,
			"topic":  "test",
			"scopes": a.scopes,
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

var _ = Describe("Resource Relations Page", Label("module", "web"), func() {
	var (
		origDB    store.Adapter
		adapter   *relationsWebAdapter
		seedLinks []*gen.ResourceLink
		seedEvts  []*gen.DataEvent
		link1     *gen.ResourceLink
		link2     *gen.ResourceLink
	)

	BeforeEach(func() {
		origDB = store.Database
		adapter = &relationsWebAdapter{
			ent:    EntClient,
			uid:    "bdd-relations-uid-" + types.Id(),
			scopes: []string{"read", "write"},
		}
		store.Database = adapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)

		ctx := context.Background()

		// Seed two data events for resource links (unique constraint on source+target event_id)
		evt1 := EntClient.DataEvent.Create().
			SetEventID("bdd-relations-event-src-" + types.Id()).
			SetEventType("bookmark.created").
			SetSource("test-agent").
			SetCreatedAt(time.Now().Add(-1 * time.Hour)).
			SaveX(ctx)
		evt2 := EntClient.DataEvent.Create().
			SetEventID("bdd-relations-event-tgt-" + types.Id()).
			SetEventType("webhook.push").
			SetSource("github").
			SetCreatedAt(time.Now().Add(-45 * time.Minute)).
			SaveX(ctx)
		seedEvts = []*gen.DataEvent{evt1, evt2}

		// Link 1: gitea|repo|e1 → flowbot|bookmark|e2
		link1 = EntClient.ResourceLink.Create().
			SetSourceEventID(evt1.EventID).
			SetTargetEventID(evt2.EventID).
			SetSourceApp("gitea").
			SetSourceCapability("repo").
			SetSourceEntityID("repo-entity-1").
			SetTargetApp("flowbot").
			SetTargetCapability("bookmark").
			SetTargetEntityID("bookmark-entity-2").
			SetPipelineName("sync-pipeline").
			SetCreatedAt(time.Now().Add(-30 * time.Minute)).
			SaveX(ctx)

		// Link 2: flowbot|bookmark|e2 → slack|chat|e3 (different pipeline)
		link2 = EntClient.ResourceLink.Create().
			SetSourceEventID(evt2.EventID).
			SetTargetEventID(evt1.EventID).
			SetSourceApp("flowbot").
			SetSourceCapability("bookmark").
			SetSourceEntityID("bookmark-entity-2").
			SetTargetApp("slack").
			SetTargetCapability("chat").
			SetTargetEntityID("chat-entity-3").
			SetPipelineName("notify-pipeline").
			SetCreatedAt(time.Now().Add(-10 * time.Minute)).
			SaveX(ctx)

		seedLinks = []*gen.ResourceLink{link1, link2}
	})

	AfterEach(func() {
		for _, l := range seedLinks {
			EntClient.ResourceLink.Delete().Where(
				resourcelink.ID(l.ID),
			).ExecX(context.Background())
		}
		for _, e := range seedEvts {
			EntClient.DataEvent.Delete().Where(
				dataevent.ID(e.ID),
			).ExecX(context.Background())
		}
		store.Database = origDB
	})

	Describe("GET /relations", func() {
		Context("with valid auth token", func() {
			It("returns the relations page with search input", func() {
				req := MakeRequest(http.MethodGet, "/service/web/relations", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(Or(
					ContainSubstring("search"),
					ContainSubstring("resource"),
					ContainSubstring("relation"),
				))
			})
		})

		Context("without authentication", func() {
			It("redirects to login page", func() {
				req := MakeRequest(http.MethodGet, "/service/web/relations", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
				Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
			})
		})
	})

	Describe("GET /relations/tree", func() {
		Context("with missing node parameter", func() {
			It("returns a placeholder message", func() {
				req := MakeRequest(http.MethodGet, "/service/web/relations/tree", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(string(ReadBody(resp))).To(ContainSubstring("Search for a resource entity ID"))
			})
		})

		Context("with invalid node format", func() {
			It("returns 400 with error message", func() {
				req := MakeRequest(http.MethodGet, "/service/web/relations/tree?node=bad-format", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(string(ReadBody(resp))).To(ContainSubstring("Invalid node format"))
			})
		})

		Context("with a valid node that has relations", func() {
			It("returns upstream and downstream edges", func() {
				nodeParam := "gitea|repo|repo-entity-1"
				req := MakeRequest(http.MethodGet, "/service/web/relations/tree?node="+nodeParam, nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(Or(
					ContainSubstring("flowbot"),
					ContainSubstring("bookmark"),
				))
			})
		})
	})

	Describe("GET /relations/search", func() {
		Context("with empty query", func() {
			It("returns empty body", func() {
				req := MakeRequest(http.MethodGet, "/service/web/relations/search", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(len(body)).To(BeZero())
			})
		})

		Context("with matching query", func() {
			It("finds matching resource nodes", func() {
				req := MakeRequest(http.MethodGet, "/service/web/relations/search?q=gitea", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(Or(
					ContainSubstring("gitea"),
					ContainSubstring("repo"),
				))
			})
		})
	})

	Describe("GET /relations/detail", func() {
		Context("with edge type", func() {
			It("renders edge metadata", func() {
				q := "type=edge" +
					"&source_app=gitea" +
					"&source_capability=repo" +
					"&source_entity=repo-entity-1" +
					"&target_app=flowbot" +
					"&target_capability=bookmark" +
					"&target_entity=bookmark-entity-2" +
					"&pipeline=sync-pipeline" +
					"&created_at=" + link1.CreatedAt.Format(time.RFC3339)

				req := MakeRequest(http.MethodGet, "/service/web/relations/detail?"+q, nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(Or(
					ContainSubstring("gitea"),
					ContainSubstring("flowbot"),
				))
			})
		})
	})
})
