//go:build integration
// +build integration

package specs

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/capability"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

type pipelinePageAdapter struct {
	store.Adapter
	ent    *gen.Client
	uid    string
	scopes []string
}

func (a *pipelinePageAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *pipelinePageAdapter) Close() error                     { return nil }
func (a *pipelinePageAdapter) IsOpen() bool                     { return true }
func (a *pipelinePageAdapter) GetName() string                  { return "bdd-pipeline-page" }
func (a *pipelinePageAdapter) Stats() any                       { return nil }
func (a *pipelinePageAdapter) GetDB() any                       { return a.ent }
func (a *pipelinePageAdapter) GetClient() *gen.Client           { return a.ent }

func (a *pipelinePageAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:        1,
		Flag:      flag,
		Params:    bddWebAuthParams(a.uid, a.scopes),
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (a *pipelinePageAdapter) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	return bddNoopParameterSet(ctx, flag, params, expiredAt)
}

var _ = Describe("Pipeline run pages", Label("module", "web"), func() {
	var (
		origDB  store.Adapter
		adapter *pipelinePageAdapter
		ps      *store.PipelineStore
	)

	BeforeEach(func() {
		origDB = store.Database
		adapter = &pipelinePageAdapter{
			ent:    EntClient,
			uid:    "bdd-pipeline-uid-" + types.Id(),
			scopes: bddWebScopesAdmin(),
		}
		store.Database = adapter
		ps = store.NewPipelineStore(EntClient)

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)
		bddSeedAccessToken(adapter.uid, adapter.uid, adapter.scopes)
	})

	AfterEach(func() {
		pipeline.SetActiveEngine(nil)
		store.Database = origDB
	})

	seedFailedRun := func(parent, eventID string) int64 {
		GinkgoHelper()
		ctx := context.Background()
		_, err := EntClient.DataEvent.Create().
			SetEventID(eventID).
			SetEventType("bookmark.created").
			SetSource("karakeep").
			SetData(map[string]any{"url": "https://example.com"}).
			Save(ctx)
		Expect(err).NotTo(HaveOccurred())
		run, err := ps.CreateRun(ctx, parent+"__trigger_event_0", eventID, "bookmark.created", "event")
		Expect(err).NotTo(HaveOccurred())
		step, err := ps.CreateStepRun(ctx, run.ID, "task", string(hub.CapExample), "bdd-retry", map[string]any{"title": "x"}, 1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ps.UpdateStepRun(ctx, step.ID, int(schema.PipelineFailed), nil, "boom", 1)).To(Succeed())
		Expect(ps.UpdateRunStatus(ctx, run.ID, int(schema.PipelineFailed), "boom")).To(Succeed())
		return run.ID
	}

	Describe("GET /pipelines/:name/runs/:runID/steps", func() {
		It("includes retry confirm attributes on a failed run", func() {
			parent := "bdd-retry-btn-" + types.Id()
			runID := seedFailedRun(parent, "evt-bdd-retry-btn-"+types.Id())
			path := "/service/web/pipelines/" + parent + "/runs/" + strconv.FormatInt(runID, 10) + "/steps"
			req := MakeRequest(http.MethodGet, path, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring(`data-testid="retry-failed-run"`))
			Expect(body).To(ContainSubstring(`data-confirm=`))
			Expect(body).To(ContainSubstring("/service/web/pipelines/" + parent + "/runs/" + strconv.FormatInt(runID, 10) + "/retry"))
		})
	})

	Describe("POST /pipelines/:name/runs/:runID/retry", func() {
		It("toasts when the run is not failed", func() {
			ctx := context.Background()
			parent := "bdd-retry-done-" + types.Id()
			run, err := ps.CreateRun(ctx, parent+"__trigger_event_0", "evt-bdd-done-"+types.Id(), "bookmark.created", "event")
			Expect(err).NotTo(HaveOccurred())
			Expect(ps.UpdateRunStatus(ctx, run.ID, int(schema.PipelineDone), "")).To(Succeed())

			eng := pipeline.NewEngine(nil, store.NewPipelineRunStoreAdapter(ps), nil, nil, nil)
			DeferCleanup(eng.Stop)
			pipeline.SetActiveEngine(eng)

			req := MakeRequest(http.MethodPost, "/service/web/pipelines/"+parent+"/runs/"+strconv.FormatInt(run.ID, 10)+"/retry", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
			Expect(resp.Header.Get("HX-Trigger")).To(ContainSubstring("Only failed runs can be retried"))
		})

		It("redirects to the live page after claiming a failed run", func() {
			parent := "bdd-retry-ok-" + types.Id()
			eventID := "evt-bdd-retry-ok-" + types.Id()
			runID := seedFailedRun(parent, eventID)

			op := "bdd-retry-op-" + types.Id()
			Expect(capability.RegisterInvoker(hub.CapExample, op, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
				return &capability.InvokeResult{Data: map[string]any{"ok": true}}, nil
			})).To(Succeed())
			DeferCleanup(func() { capability.UnregisterInvoker(hub.CapExample, op) })

			def := pipeline.Definition{
				Name:    parent + "__trigger_event_0",
				Enabled: true,
				Trigger: pipeline.Trigger{Event: "bookmark.created"},
				Steps: []pipeline.Step{
					{Name: "task", Capability: hub.CapExample, Operation: op},
				},
			}
			eng := pipeline.NewEngine([]pipeline.Definition{def}, store.NewPipelineRunStoreAdapter(ps), nil, nil, nil)
			DeferCleanup(eng.Stop)
			pipeline.SetActiveEngine(eng)

			req := MakeRequest(http.MethodPost, "/service/web/pipelines/"+parent+"/runs/"+strconv.FormatInt(runID, 10)+"/retry", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("HX-Redirect")).To(Equal("/service/web/pipelines/" + parent + "/runs/" + strconv.FormatInt(runID, 10) + "/live"))
		})
	})
})
