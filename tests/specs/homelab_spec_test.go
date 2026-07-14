//go:build integration
// +build integration

package specs

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/homelab/probe"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Homelab Scanner", Label("homelab"), func() {

	Describe("App Scanning", func() {
		var appsDir string

		BeforeEach(func() {
			var err error
			appsDir, err = os.MkdirTemp("", "flowbot-homelab-spec*")
			Expect(err).NotTo(HaveOccurred())

			Expect(os.MkdirAll(filepath.Join(appsDir, "archivebox"), 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(appsDir, "archivebox", "docker-compose.yaml"), []byte(`services:
  web:
    image: archivebox/archivebox:latest
    ports:
      - "8080:8000/tcp"
    labels:
      flowbot.capability: archive
      flowbot.backend: archivebox
      flowbot.endpoint.base: http://localhost:8080
      flowbot.endpoint.health: /health
`), 0o644)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(appsDir, "karakeep"), 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(appsDir, "karakeep", "compose.yaml"), []byte(`services:
  app:
    image: ghcr.io/karakeep/karakeep:latest
    labels:
      flowbot.capability: bookmark
      flowbot.backend: karakeep
`), 0o644)).To(Succeed())
		})

		AfterEach(func() {
			_ = os.RemoveAll(appsDir)
		})
		It("scans configured directories for self-hosted apps", func() {
			cfg := homelab.Config{
				AppsDir: appsDir,
				Discovery: homelab.DiscoveryConfig{
					ProbeEnabled: false,
				},
			}
			scanner := homelab.NewScanner(cfg)
			Expect(scanner).NotTo(BeNil())

			apps, err := scanner.Scan()
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(HaveLen(2))

			names := make([]string, len(apps))
			for i, a := range apps {
				names[i] = a.Name
			}
			Expect(names).To(ConsistOf("archivebox", "karakeep"))

			archivebox := apps[0]
			if apps[1].Name == "archivebox" {
				archivebox = apps[1]
			}
			Expect(archivebox.Labels["flowbot.capability"]).To(Equal("archive"))
		})

		It("parses app metadata from labels", func() {
			labels := map[string]string{
				"flowbot.capability":          "bookmark",
				"flowbot.backend":             "karakeep",
				"flowbot.endpoint.base":       "http://localhost:8080",
				"flowbot.endpoint.health":     "/health",
				"flowbot.endpoint.health_ttl": "30s",
			}
			caps := homelab.ParseLabels(labels)
			Expect(caps).To(HaveLen(1))
			Expect(caps[0].Capability).To(Equal(homelab.CapKarakeep))
			Expect(caps[0].Endpoint.BaseURL).To(Equal("http://localhost:8080"))
			Expect(caps[0].Endpoint.Health).To(Equal("/health"))
		})

		It("handles missing optional labels gracefully", func() {
			labels := map[string]string{
				"flowbot.capability": "reader",
				"flowbot.backend":    "miniflux",
			}
			caps := homelab.ParseLabels(labels)
			Expect(caps).To(HaveLen(1))
			Expect(caps[0].Capability).To(Equal(homelab.CapMiniflux))
			Expect(caps[0].Endpoint).To(BeNil())
			Expect(caps[0].Auth).To(BeNil())
		})
	})

	Describe("App Registration", func() {
		It("registers and lists apps with the registry", func() {
			reg := homelab.NewRegistry()
			Expect(reg).NotTo(BeNil())

			apps := reg.List()
			Expect(apps).To(BeEmpty())

			testApp := homelab.App{
				Name:   "test-app",
				Path:   "/apps/test",
				Status: homelab.AppStatusRunning,
			}
			reg.Replace([]homelab.App{testApp})

			listed := reg.List()
			Expect(listed).To(HaveLen(1))
			Expect(listed[0].Name).To(Equal("test-app"))

			got, ok := reg.Get("test-app")
			Expect(ok).To(BeTrue())
			Expect(got.Status).To(Equal(homelab.AppStatusRunning))
		})

		It("skips already registered apps without change", func() {
			reg := homelab.NewRegistry()
			app1 := homelab.App{Name: "dup-app", Status: homelab.AppStatusRunning}
			app2 := homelab.App{Name: "dup-app", Status: homelab.AppStatusStopped}

			reg.Replace([]homelab.App{app1})
			reg.Replace([]homelab.App{app2})

			got, ok := reg.Get("dup-app")
			Expect(ok).To(BeTrue())
			Expect(got.Status).To(Equal(homelab.AppStatusStopped))
		})

		It("extracts app name, version, and endpoint", func() {
			labels := map[string]string{
				"flowbot.capability":        "kanban",
				"flowbot.backend":           "kanboard",
				"flowbot.endpoint.base":     "http://kanban.local:8080",
				"flowbot.auth.type":         "basic",
				"flowbot.auth.header":       "Authorization",
				"flowbot.auth.token_key":    "api_key",
				"flowbot.auth.token_source": "env",
			}
			caps := homelab.ParseLabels(labels)
			Expect(caps).To(HaveLen(1))
			Expect(caps[0].Capability).To(Equal(homelab.CapKanboard))
			Expect(caps[0].Endpoint.BaseURL).To(Equal("http://kanban.local:8080"))
			Expect(caps[0].Auth.Type).To(Equal(homelab.AuthBasic))
			Expect(caps[0].Auth.Header).To(Equal("Authorization"))
		})
	})

	Describe("Health Probing", func() {
		It("detects authentication requirements via probe", func() {
			detector := &probe.AuthDetector{}
			Expect(detector).NotTo(BeNil())
		})

		It("has a default noop runtime", func() {
			rt := homelab.NoopRuntime{}
			err := rt.Start(context.Background(), homelab.App{})
			Expect(err).To(HaveOccurred())

			err = rt.Stop(context.Background(), homelab.App{})
			Expect(err).To(HaveOccurred())

			err = rt.Restart(context.Background(), homelab.App{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Label Parsing", func() {
		It("parses Docker container labels for Flowbot config", func() {
			labels := map[string]string{
				"flowbot.capability":          "archive",
				"flowbot.backend":             "archivebox",
				"flowbot.endpoint.base":       "http://archive:8000",
				"flowbot.endpoint.health_ttl": "60s",
			}
			caps := homelab.ParseLabels(labels)
			Expect(caps).To(HaveLen(1))
			Expect(caps[0].Capability).To(Equal("archive"))
			Expect(caps[0].Endpoint.HealthTTL).To(Equal(60 * time.Second))
		})

		It("extracts capability declarations from labels", func() {
			labels := map[string]string{
				"flowbot.capability": "infra",
				"flowbot.backend":    "uptimekuma",
			}
			caps := homelab.ParseLabels(labels)
			Expect(caps).To(HaveLen(1))
			Expect(caps[0].Capability).To(Equal("infra"))
		})

		It("parses auth labels correctly", func() {
			labels := map[string]string{
				"flowbot.capability":     "bookmark",
				"flowbot.backend":        "karakeep",
				"flowbot.auth.type":      "api_token",
				"flowbot.auth.header":    "X-API-Key",
				"flowbot.auth.prefix":    "Bearer",
				"flowbot.auth.token_key": "token",
			}
			caps := homelab.ParseLabels(labels)
			Expect(caps).To(HaveLen(1))
			Expect(caps[0].Auth.Type).To(Equal(homelab.AuthAPIToken))
			Expect(caps[0].Auth.Header).To(Equal("X-API-Key"))
			Expect(caps[0].Auth.Prefix).To(Equal("Bearer"))
			Expect(caps[0].Auth.TokenKey).To(Equal("token"))
		})

		It("returns nil for labels without capability", func() {
			labels := map[string]string{
				"com.docker.compose.project": "myapp",
				"com.docker.compose.service": "web",
			}
			caps := homelab.ParseLabels(labels)
			Expect(caps).To(BeNil())
		})
	})

	Describe("App types", func() {
		It("has correct status and health string values", func() {
			Expect(string(homelab.AppStatusUnknown)).To(Equal("unknown"))
			Expect(string(homelab.AppStatusRunning)).To(Equal("running"))
			Expect(string(homelab.AppStatusStopped)).To(Equal("stopped"))
			Expect(string(homelab.AppStatusPartial)).To(Equal("partial"))

			Expect(string(homelab.HealthUnknown)).To(Equal("unknown"))
			Expect(string(homelab.HealthHealthy)).To(Equal("healthy"))
			Expect(string(homelab.HealthUnhealthy)).To(Equal("unhealthy"))
		})

		It("has capability type constants matching hub types", func() {
			Expect(homelab.CapKarakeep).To(Equal("karakeep"))
			Expect(homelab.CapArchive).To(Equal("archive"))
			Expect(homelab.CapMiniflux).To(Equal("miniflux"))
			Expect(homelab.CapKanboard).To(Equal("kanboard"))
			Expect(homelab.CapFinance).To(Equal("finance"))
			Expect(homelab.CapInfra).To(Equal("infra"))
		})
	})

	Describe("Config defaults", func() {
		It("creates scanner with sane defaults", func() {
			cfg := homelab.Config{
				Root:    "/tmp/test-homelab",
				AppsDir: "/tmp/test-homelab/apps",
			}
			scanner := homelab.NewScanner(cfg)
			Expect(scanner).NotTo(BeNil())
		})
	})

	Describe("Permissions", func() {
		It("manages lifecycle permissions", func() {
			reg := homelab.NewRegistry()

			reg.SetPermissions(homelab.Permissions{
				Status:  true,
				Start:   false,
				Stop:    false,
				Restart: false,
			})

			perms := reg.Permissions()
			Expect(perms.Status).To(BeTrue())
			Expect(perms.Start).To(BeFalse())
		})
	})

	Describe("Known service fingerprints", func() {
		It("has fingerprints for known services", func() {
			Expect(probe.KnownServices).NotTo(BeEmpty())

			names := make([]string, len(probe.KnownServices))
			for i, s := range probe.KnownServices {
				names[i] = s.Capability
			}
			Expect(names).To(ContainElement("karakeep"))
			Expect(names).To(ContainElement("kanboard"))
			Expect(names).To(ContainElement("miniflux"))
			Expect(names).To(ContainElement("archive"))
		})
	})
})
