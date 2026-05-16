//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Ability Layer", Label("ability"), func() {

	Describe("ability.Invoke", func() {
		Context("with a registered capability", func() {
			It("invokes a capability operation successfully")
			It("returns the operation result")
			It("passes parameters to the capability handler")
		})

		Context("with an unregistered capability", func() {
			It("returns capability not found error")
		})

		Context("with a valid capability but invalid operation", func() {
			It("returns operation not supported error")
		})
	})

	Describe("Pagination", func() {
		It("returns paginated results with limit")
		It("returns a cursor for the next page")
		It("returns empty cursor on the last page")
		It("rejects negative limit values")
		It("caps limit at maximum page size")
	})

	Describe("Opaque Cursor", func() {
		It("encodes cursor data opaquely")
		It("decodes cursor back to original data")
		It("rejects tampered cursor data")
	})

	Describe("Parameter Validation", func() {
		It("validates required parameters are present")
		It("validates parameter types match schema")
		It("returns descriptive validation errors")
	})
})
