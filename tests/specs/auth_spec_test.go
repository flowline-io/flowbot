//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Authentication", Label("auth"), func() {

	Describe("AuthContext", func() {
		Context("REST context", func() {
			It("extracts user from bearer token")
			It("returns unauthorized for missing token")
			It("returns unauthorized for expired token")
			It("returns unauthorized for invalid token")
		})

		Context("CLI context", func() {
			It("extracts user from stored credentials")
			It("returns unauthorized for expired session")
		})

		Context("Chat context", func() {
			It("extracts user from chat platform ID")
			It("auto-registers new chat users")
		})

		Context("Webhook context", func() {
			It("validates webhook secret")
			It("associates webhook with owning user")
		})

		Context("Cron context", func() {
			It("runs as system user")
			It("scopes permissions to cron job level")
		})

		Context("Pipeline context", func() {
			It("inherits trigger event's auth context")
			It("propagates user identity through pipeline steps")
		})

		Context("Workflow context", func() {
			It("runs as workflow owner by default")
			It("supports run-as override for shared workflows")
		})
	})

	Describe("Permission Checks", func() {
		It("grants access to owned resources")
		It("denies access to other user's resources")
		It("grants access to admin users for all resources")
		It("denies access to unauthenticated requests")
	})
})
