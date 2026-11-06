//go:build integration

package specs

import (
	"context"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	analyticsmod "github.com/flowline-io/flowbot/internal/modules/analytics"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
)

var _ = Describe("Lineage / flows / events API", Label("module", "analytics", "lineage"), func() {
	var (
		token string
		uid   string
	)

	BeforeEach(func() {
		Expect(analyticsmod.InitForE2E(nil)).To(Succeed())
		analyticsmod.MountForE2E(App)
		token = "bdd-lineage-" + types.Id()
		uid = "bdd-lineage-user-" + types.Id()
		bddSeedAccessToken(token, uid, []string{auth.ScopeLineageRead})
	})

	It("rejects unauthenticated lineage trace", func() {
		req := MakeRequest(http.MethodGet, "/service/lineage/trace?event_id=ev-1", nil)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		_ = resp.Body.Close()
	})

	It("returns flow.graph edges for the seeded window", func() {
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)
		_, err := EntClient.ResourceLink.Create().
			SetSourceEventID("bdd-lineage-src-" + types.Id()).
			SetTargetEventID("bdd-lineage-tgt-" + types.Id()).
			SetSourceApp("miniflux").
			SetTargetApp("karakeep").
			SetPipelineName("bdd-flow").
			SetRunKind(resourcelink.RunKindPipeline).
			SetStatus(resourcelink.StatusOk).
			SetCreatedAt(now.Add(-time.Hour)).
			Save(ctx)
		Expect(err).NotTo(HaveOccurred())

		req := MakeRequest(http.MethodGet, "/service/flows/graph", nil)
		req.Header.Set("X-AccessToken", token)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body := ReadBody(resp)
		var envelope map[string]any
		Expect(sonic.Unmarshal(body, &envelope)).To(Succeed())
		Expect(envelope["status"]).To(Equal("ok"))
		data, _ := sonic.Marshal(envelope["data"])
		var graph types.FlowGraph
		Expect(sonic.Unmarshal(data, &graph)).To(Succeed())
		Expect(graph.Edges).NotTo(BeEmpty())
		found := false
		for _, e := range graph.Edges {
			if e.Source == "miniflux" && e.Target == "karakeep" && e.Via.Name == "bdd-flow" {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())
	})

	It("rejects lineage.trace without exactly one seed", func() {
		req := MakeRequest(http.MethodGet, "/service/lineage/trace", nil)
		req.Header.Set("X-AccessToken", token)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		_ = resp.Body.Close()
	})

	It("omits event payloads by default and marks payload_omitted", func() {
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)
		eventID := "bdd-timeline-" + types.Id()
		_, err := EntClient.DataEvent.Create().
			SetEventID(eventID).
			SetEventType("karakeep.bookmark.created").
			SetApp("karakeep").
			SetCapability("karakeep").
			SetUID(uid).
			SetData(map[string]any{"secret": "x"}).
			SetCreatedAt(now.Add(-time.Hour)).
			Save(ctx)
		Expect(err).NotTo(HaveOccurred())

		req := MakeRequest(http.MethodGet, "/service/events/timeline", nil)
		req.Header.Set("X-AccessToken", token)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body := ReadBody(resp)
		Expect(string(body)).To(ContainSubstring(`"payload_omitted":true`))
		Expect(string(body)).NotTo(ContainSubstring(`"secret"`))
	})
})
