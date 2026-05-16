//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Gitea Module", Label("module", "gitea"), func() {

	Describe("Command", func() {
		Context("gitea", func() {
			It("fetches demo repository information")
			It("returns repository name, description, and clone URL")
		})
	})

	Describe("Webhook — issue", func() {
		Context("issue created / opened", func() {
			It("processes new issue creation event")
			It("creates a notification for the issue")
		})

		Context("issue closed", func() {
			It("processes issue closure event")
			It("updates related task status")
		})
	})

	Describe("Webhook — repo", func() {
		Context("push event", func() {
			It("processes repository push event")
			It("triggers related pipeline on push to main branch")
			It("ignores push to non-tracked branches")
		})
	})

	Describe("Cron", func() {
		It("gitea_metrics — collects issue count statistics every minute")
	})
})
