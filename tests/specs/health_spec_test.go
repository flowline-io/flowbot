//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health", Label("health", "smoke"), func() {

	Describe("HTTP health endpoints", func() {
		DescribeTable("returns 200",
			func(endpoint string) {
				req := MakeRequest(http.MethodGet, endpoint, nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			},
			Entry("liveness endpoint", "/livez"),
			Entry("readiness endpoint", "/readyz"),
			Entry("startup endpoint", "/startupz"),
		)
	})

	Describe("infrastructure connectivity", func() {
		It("database is accessible", func() {
			Expect(DB.Ping()).To(Succeed())
		})

		It("Redis is accessible", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := Redis.Set(ctx, "test:specs:key", "value", time.Minute).Err()
			Expect(err).NotTo(HaveOccurred())

			val, err := Redis.Get(ctx, "test:specs:key").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("value"))

			Expect(Redis.Del(ctx, "test:specs:key").Err()).NotTo(HaveOccurred())
		})

		It("containers are running", func() {
			if GinkgoParallelProcess() != 1 {
				Skip("container references only available in process 1")
			}
			Expect(pgC).NotTo(BeNil(), "PostgreSQL container should be running")
			Expect(redisC).NotTo(BeNil(), "Redis container should be running")

			Eventually(func(g Gomega) {
				state, err := pgC.State(suiteCtx)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(state.Running).To(BeTrue(), "PostgreSQL container should be in running state")
			}, "30s", "500ms").Should(Succeed())

			Eventually(func(g Gomega) {
				state, err := redisC.State(suiteCtx)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(state.Running).To(BeTrue(), "Redis container should be in running state")
			}, "30s", "500ms").Should(Succeed())
		})
	})

	Describe("schema migrations", func() {
		It("users table exists", func() {
			var exists bool
			err := DB.QueryRow(
				"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')",
			).Scan(&exists)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "users table should exist after migrations")
		})
	})
})
