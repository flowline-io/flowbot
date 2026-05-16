//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Server Module", Label("module", "server"), func() {

	Describe("Command", func() {
		Context("version", func() {
			It("returns the current server version")
		})

		Context("mem stats", func() {
			It("returns memory usage statistics")
			It("reports heap, stack, and GC metrics")
		})

		Context("golang stats", func() {
			It("returns Go runtime statistics")
			It("reports goroutine count, CPU count, and Go version")
		})

		Context("server stats", func() {
			It("returns server uptime and request count")
			It("reports active connections")
		})

		Context("online stats", func() {
			It("returns online user statistics")
		})

		Context("adguard status", func() {
			It("returns AdGuard Home service status")
			It("reports when AdGuard is unreachable")
		})

		Context("adguard stats", func() {
			It("returns DNS query statistics from AdGuard")
			It("returns blocked query count")
		})

		Context("queue stats", func() {
			It("returns Redis Stream queue statistics")
			It("reports pending and processed message counts")
		})

		Context("check", func() {
			It("runs a system health check")
			It("reports all subsystem statuses")
		})
	})

	Describe("Webservice", func() {
		Context("POST /upload", func() {
			It("accepts file upload and returns URL")
			It("rejects upload exceeding size limit")
			It("rejects upload with unsupported content type")
		})

		Context("GET /stacktrace", func() {
			It("returns process stacktrace for diagnostics")
			It("returns runtime memory profile data")
		})
	})

	Describe("Cron Jobs", func() {
		It("server_user_online_change — tracks user online status changes")
		It("docker_images_prune — prunes unused Docker images daily")
		It("docker_metrics — collects Docker container metrics every minute")
		It("monitor_metrics — collects UptimeKuma monitor metrics")
		It("online_agent_checker — checks agent online status every 2 minutes")
	})
})
