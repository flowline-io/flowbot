//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Homelab Scanner", Label("homelab"), func() {

	Describe("App Scanning", func() {
		It("scans configured directories for self-hosted apps")
		It("detects Docker Compose files")
		It("detects static configuration files")
		It("parses app metadata from labels")
	})

	Describe("App Registration", func() {
		It("registers a discovered app with the hub")
		It("extracts app name, version, and endpoint")
		It("assigns capabilities based on detected labels")
		It("skips already registered apps without change")
	})

	Describe("Health Probing", func() {
		It("probes app health endpoint on registration")
		It("detects authentication requirements via probe")
		It("classifies app as healthy when endpoint responds 200")
		It("classifies app as degraded when endpoint responds non-200")
	})

	Describe("Label Parsing", func() {
		It("parses Docker container labels for Flowbot config")
		It("extracts capability declarations from labels")
		It("extracts health check URLs from labels")
		It("handles missing optional labels gracefully")
	})
})
