//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var gatewayRoutesOnce sync.Once

func mountGatewayRoutes(app *fiber.App) {
	gatewayRoutesOnce.Do(func() {
		server.RegisterGatewayRoutes(app)
	})
}

func gatewayWorkerToken(ctx context.Context) string {
	token, err := auth.NewToken()
	Expect(err).NotTo(HaveOccurred())
	bddSeedAccessToken(token, "gateway-worker-bdd", []string{auth.ScopeGatewayWorker})
	return token
}

func gatewayRequest(method, path, token string, body []byte) *http.Request {
	req := JSONRequest(method, path, body)
	if token != "" {
		req.Header.Set("X-AccessToken", token)
	}
	return req
}

var _ = Describe("Local CLI Gateway API", Label("gateway", "api"), func() {
	BeforeEach(func() {
		mountGatewayRoutes(App)
	})

	It("rejects claim without gateway:worker scope", func() {
		body, _ := sonic.Marshal(types.GatewayClaimRequest{WorkerID: "w1"})
		req := gatewayRequest(http.MethodPost, "/gateway/v1/claim", "", body)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("heartbeats, claims, completes, and returns truncated metadata", func() {
		ctx := context.Background()
		token := gatewayWorkerToken(ctx)
		gs := store.GatewayStoreFromDB()

		hbBody, _ := sonic.Marshal(types.GatewayHeartbeatRequest{WorkerID: "bdd-worker-1"})
		hbReq := gatewayRequest(http.MethodPost, "/gateway/v1/heartbeat", token, hbBody)
		hbResp, err := App.Test(hbReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(hbResp.StatusCode).To(Equal(http.StatusOK))

		emptyBody, _ := sonic.Marshal(types.GatewayClaimRequest{WorkerID: "bdd-worker-1"})
		emptyReq := gatewayRequest(http.MethodPost, "/gateway/v1/claim", token, emptyBody)
		emptyResp, err := App.Test(emptyReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(emptyResp.StatusCode).To(Equal(http.StatusOK))
		var emptyOut protocol.Response
		Expect(sonic.Unmarshal(ReadBody(emptyResp), &emptyOut)).To(Succeed())
		Expect(emptyOut.Status).To(Equal(protocol.Success))

		job, err := gs.Create(ctx, types.GatewayCreateJob{
			UID: "bdd", CLI: types.GatewayCLICursor, Prompt: "say hi",
		})
		Expect(err).NotTo(HaveOccurred())

		claimBody, _ := sonic.Marshal(types.GatewayClaimRequest{WorkerID: "bdd-worker-1"})
		claimReq := gatewayRequest(http.MethodPost, "/gateway/v1/claim", token, claimBody)
		claimResp, err := App.Test(claimReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(claimResp.StatusCode).To(Equal(http.StatusOK))
		var claimOut protocol.Response
		Expect(sonic.Unmarshal(ReadBody(claimResp), &claimOut)).To(Succeed())
		raw, err := sonic.Marshal(claimOut.Data)
		Expect(err).NotTo(HaveOccurred())
		var claimed types.GatewayClaimResponse
		Expect(sonic.Unmarshal(raw, &claimed)).To(Succeed())
		Expect(claimed.Job).NotTo(BeNil())
		Expect(claimed.Job.JobID).To(Equal(job.JobID))
		Expect(claimed.Job.Status).To(Equal(types.GatewayJobRunning))

		code := 0
		resultBody, _ := sonic.Marshal(types.GatewayCompleteRequest{
			WorkerID: "bdd-worker-1",
			Status:   types.GatewayJobSucceeded,
			Output:   "abcdefghij",
			ExitCode: &code,
		})
		prevMax := config.App.Gateway.MaxOutputBytes
		prevTool := config.App.ChatAgent.MaxToolOutput
		config.App.Gateway.MaxOutputBytes = 5
		config.App.ChatAgent.MaxToolOutput = 5
		DeferCleanup(func() {
			config.App.Gateway.MaxOutputBytes = prevMax
			config.App.ChatAgent.MaxToolOutput = prevTool
		})

		resultReq := gatewayRequest(http.MethodPost, "/gateway/v1/jobs/"+job.JobID+"/result", token, resultBody)
		resultResp, err := App.Test(resultReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(resultResp.StatusCode).To(Equal(http.StatusOK))
		var resultOut protocol.Response
		Expect(sonic.Unmarshal(ReadBody(resultResp), &resultOut)).To(Succeed())
		jobRaw, err := sonic.Marshal(resultOut.Data)
		Expect(err).NotTo(HaveOccurred())
		var done types.GatewayJob
		Expect(sonic.Unmarshal(jobRaw, &done)).To(Succeed())
		Expect(done.Status).To(Equal(types.GatewayJobSucceeded))
		Expect(done.Truncated).To(BeTrue())
		Expect(done.Output).To(Equal("ab..."))

		getReq := gatewayRequest(http.MethodGet, "/gateway/v1/jobs/"+job.JobID, token, nil)
		getResp, err := App.Test(getReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(getResp.StatusCode).To(Equal(http.StatusOK))
	})

	It("reclaims expired leases back to pending for another claim", func() {
		ctx := context.Background()
		token := gatewayWorkerToken(ctx)
		gs := store.GatewayStoreFromDB()

		job, err := gs.Create(ctx, types.GatewayCreateJob{
			CLI: types.GatewayCLICursor, Prompt: "lease reclaim",
		})
		Expect(err).NotTo(HaveOccurred())

		claimed, err := gs.Claim(ctx, "lease-w1", time.Millisecond)
		Expect(err).NotTo(HaveOccurred())
		Expect(claimed).NotTo(BeNil())
		Expect(claimed.JobID).To(Equal(job.JobID))

		time.Sleep(5 * time.Millisecond)
		Expect(gs.ReclaimExpired(ctx)).To(Succeed())

		got, err := gs.Get(ctx, job.JobID)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Status).To(Equal(types.GatewayJobPending))

		claimBody, _ := sonic.Marshal(types.GatewayClaimRequest{WorkerID: "lease-w2"})
		claimReq := gatewayRequest(http.MethodPost, "/gateway/v1/claim", token, claimBody)
		claimResp, err := App.Test(claimReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(claimResp.StatusCode).To(Equal(http.StatusOK))
		var claimOut protocol.Response
		Expect(sonic.Unmarshal(ReadBody(claimResp), &claimOut)).To(Succeed())
		raw, err := sonic.Marshal(claimOut.Data)
		Expect(err).NotTo(HaveOccurred())
		var again types.GatewayClaimResponse
		Expect(sonic.Unmarshal(raw, &again)).To(Succeed())
		Expect(again.Job).NotTo(BeNil())
		Expect(again.Job.JobID).To(Equal(job.JobID))
		Expect(again.Job.WorkerID).To(Equal("lease-w2"))
	})
})
