//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("GitHub Module", Label("module", "github"), func() {

	Describe("Command", func() {
		Context("github setting", func() {
			It("configures GitHub OAuth settings")
			It("persists settings to config store")
		})

		Context("github oauth", func() {
			It("initiates OAuth authorization flow")
			It("returns authorization URL")
		})

		Context("github user", func() {
			It("returns authenticated user profile")
			It("returns error when not authenticated")
		})

		Context("github card [string]", func() {
			It("returns a GitHub repository card view")
			It("returns error for non-existent repo")
		})

		Context("github repo [string]", func() {
			It("returns detailed repository information")
			It("supports owner/repo format")
		})

		Context("github user [string]", func() {
			It("returns GitHub user profile by username")
			It("returns error for non-existent user")
		})

		Context("deploy", func() {
			It("triggers deployment for a package")
			It("returns error when deployment fails")
		})
	})

	Describe("Webhook — package", func() {
		Context("ping event", func() {
			It("responds with pong")
		})

		Context("package:published event", func() {
			It("triggers deployment for published package")
			It("handles different package types")
		})
	})

	Describe("Cron Jobs", func() {
		It("github_starred — syncs starred repositories every 30 minutes")
		It("github_notifications — syncs GitHub notifications every minute")
	})
})
