//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resource Relations Page", func() {
	Describe("GET /service/web/relations", func() {
		It("loads the page with search input and empty state", func() {
			Skip("scaffold: requires full DI wiring for BDD")
		})
	})

	Describe("GET /service/web/relations/search", func() {
		It("finds matching nodes by entity ID", func() {
			Skip("scaffold: requires full DI wiring for BDD")
		})

		It("returns empty state for no match", func() {
			Skip("scaffold: requires full DI wiring for BDD")
		})
	})

	Describe("GET /service/web/relations/tree", func() {
		It("shows node with upstream and downstream", func() {
			Skip("scaffold: requires full DI wiring for BDD")
		})

		It("filters by pipeline name", func() {
			Skip("scaffold: requires full DI wiring for BDD")
		})
	})

	Describe("GET /service/web/relations/detail", func() {
		It("shows node metadata", func() {
			Skip("scaffold: requires full DI wiring for BDD")
		})

		It("shows edge metadata", func() {
			Skip("scaffold: requires full DI wiring for BDD")
		})
	})
})
