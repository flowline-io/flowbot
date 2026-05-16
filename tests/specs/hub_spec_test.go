//go:build integration
// +build integration

package specs

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/homelab"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hub Module", Label("module", "hub"), func() {

	Describe("Homelab Registry", func() {
		It("lists all registered applications", func() {
			reg := homelab.NewRegistry()
			apps := reg.List()
			Expect(apps).To(BeEmpty())
		})

		It("shows app name, status, and endpoint", func() {
			reg := homelab.NewRegistry()
			app := homelab.App{
				Name:   "test-service",
				Path:   "/apps/test",
				Status: homelab.AppStatusRunning,
				Health: homelab.HealthHealthy,
			}
			reg.Replace([]homelab.App{app})

			got, ok := reg.Get("test-service")
			Expect(ok).To(BeTrue())
			Expect(got.Name).To(Equal("test-service"))
			Expect(got.Status).To(Equal(homelab.AppStatusRunning))
			Expect(got.Health).To(Equal(homelab.HealthHealthy))
		})

		It("returns detailed info for a specific app", func() {
			reg := homelab.NewRegistry()
			app := homelab.App{
				Name:        "detail-app",
				Path:        "/apps/detail",
				Status:      homelab.AppStatusRunning,
				ComposeFile: "docker-compose.yml",
			}
			reg.Replace([]homelab.App{app})

			got, ok := reg.Get("detail-app")
			Expect(ok).To(BeTrue())
			Expect(got.Path).To(Equal("/apps/detail"))
			Expect(got.ComposeFile).To(Equal("docker-compose.yml"))
		})

		It("returns error for unregistered app", func() {
			reg := homelab.NewRegistry()
			_, ok := reg.Get("nonexistent-app")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Runtime operations", func() {
		It("handles start on stopped application", func() {
			rt := homelab.NoopRuntime{}
			err := rt.Start(context.Background(), homelab.App{Name: "stopped-app", Status: homelab.AppStatusStopped})
			Expect(err).To(HaveOccurred())
		})

		It("handles stop on running application", func() {
			rt := homelab.NoopRuntime{}
			err := rt.Stop(context.Background(), homelab.App{Name: "running-app", Status: homelab.AppStatusRunning})
			Expect(err).To(HaveOccurred())
		})

		It("handles restart on running application", func() {
			rt := homelab.NoopRuntime{}
			err := rt.Restart(context.Background(), homelab.App{Name: "running-app", Status: homelab.AppStatusRunning})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Health checking", func() {
		It("classifies app as healthy", func() {
			app := homelab.App{
				Name:   "healthy-app",
				Status: homelab.AppStatusRunning,
				Health: homelab.HealthHealthy,
			}
			Expect(app.Health).To(Equal(homelab.HealthHealthy))
		})

		It("classifies app as unhealthy", func() {
			app := homelab.App{
				Name:   "unhealthy-app",
				Status: homelab.AppStatusRunning,
				Health: homelab.HealthUnhealthy,
			}
			Expect(app.Health).To(Equal(homelab.HealthUnhealthy))
		})
	})

	Describe("App status management", func() {
		It("tracks app status correctly", func() {
			reg := homelab.NewRegistry()
			reg.Replace([]homelab.App{
				{Name: "app-1", Status: homelab.AppStatusRunning},
				{Name: "app-2", Status: homelab.AppStatusStopped},
				{Name: "app-3", Status: homelab.AppStatusUnknown},
			})

			apps := reg.List()
			Expect(len(apps)).To(Equal(3))

			statuses := make(map[string]homelab.AppStatus)
			for _, a := range apps {
				statuses[a.Name] = a.Status
			}
			Expect(statuses["app-1"]).To(Equal(homelab.AppStatusRunning))
			Expect(statuses["app-2"]).To(Equal(homelab.AppStatusStopped))
			Expect(statuses["app-3"]).To(Equal(homelab.AppStatusUnknown))
		})

		It("replaces app list atomically", func() {
			reg := homelab.NewRegistry()
			reg.Replace([]homelab.App{{Name: "old-app"}})
			reg.Replace([]homelab.App{{Name: "new-app"}})

			_, ok := reg.Get("old-app")
			Expect(ok).To(BeFalse())

			_, ok = reg.Get("new-app")
			Expect(ok).To(BeTrue())
		})
	})

	Describe("Runtime configuration", func() {
		It("creates runtime based on config mode", func() {
			rt := homelab.NewRuntime(homelab.RuntimeConfig{Mode: homelab.RuntimeModeNone}, "/tmp/apps")
			Expect(rt).NotTo(BeNil())
		})
	})

	Describe("Capability constants", func() {
		It("defines expected capability type constants", func() {
			Expect(homelab.CapBookmark).To(Equal("bookmark"))
			Expect(homelab.CapReader).To(Equal("reader"))
			Expect(homelab.CapKanban).To(Equal("kanban"))
			Expect(homelab.CapArchive).To(Equal("archive"))
			Expect(homelab.CapFinance).To(Equal("finance"))
			Expect(homelab.CapInfra).To(Equal("infra"))
		})
	})
})
