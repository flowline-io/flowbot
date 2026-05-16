//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Notify Module", Label("module", "notify"), func() {

	Describe("Command", func() {
		Context("notify list", func() {
			It("lists all notification templates")
			It("returns empty list when no templates exist")
		})

		Context("notify delete [string]", func() {
			It("deletes a notification template by name")
			It("returns error for non-existent template")
		})

		Context("notify config", func() {
			It("shows current notification configuration")
			It("displays configured channels and their status")
		})
	})

	Describe("Form — create_notify", func() {
		It("creates a new notification template with name and template body")
		It("rejects creation with empty name")
		It("rejects creation with empty template")
		It("persists template to config store")
	})

	Describe("Multi-Channel Delivery", func() {
		Context("Slack", func() {
			It("sends notification to Slack channel")
			It("handles Slack API errors gracefully")
		})

		Context("Discord", func() {
			It("sends notification to Discord channel")
			It("handles Discord API rate limits")
		})

		Context("ntfy", func() {
			It("sends notification to ntfy topic")
			It("handles ntfy server unreachable")
		})

		Context("Email", func() {
			It("sends email notification")
			It("handles SMTP errors")
		})

		Context("Fallback", func() {
			It("falls back to next channel when primary fails")
			It("reports delivery failure when all channels fail")
		})
	})

	Describe("Template Rendering", func() {
		It("renders notification body from Go template")
		It("injects event data into template context")
		It("returns error for malformed template syntax")
	})
})
