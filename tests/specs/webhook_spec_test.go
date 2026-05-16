//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Webhook Module", Label("module", "webhook"), func() {

	Describe("Command", func() {
		Context("webhook list", func() {
			It("lists all configured webhooks")
			It("shows webhook secret and active status")
		})

		Context("webhook create [flag]", func() {
			It("creates a new webhook with generated secret")
			It("rejects creation with duplicate flag")
		})

		Context("webhook del [secret]", func() {
			It("deletes a webhook by secret")
			It("returns error for non-existent secret")
		})

		Context("webhook activate [secret]", func() {
			It("activates a disabled webhook")
			It("returns error when webhook is already active")
		})

		Context("webhook inactive [secret]", func() {
			It("deactivates a webhook")
			It("returns error when webhook is already inactive")
		})
	})
})
