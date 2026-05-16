//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Pipeline Engine", Label("pipeline"), func() {

	Describe("Pipeline Execution", func() {
		Context("with a valid definition", func() {
			It("executes all steps in order")
			It("passes step output as input to the next step")
			It("returns final step result on success")
			It("supports conditional step execution")
		})

		Context("with a disabled pipeline", func() {
			It("skips execution entirely")
			It("does not trigger on matching events")
		})

		Context("with a failing step", func() {
			It("stops execution at the failed step")
			It("records the failure in step results")
			It("retries if retry policy is configured")
			It("respects maximum retry count")
		})

		Context("with retry configuration", func() {
			It("retries with exponential backoff")
			It("gives up after maximum retries exceeded")
			It("identifies retryable vs non-retryable errors")
		})
	})

	Describe("Event Matching", func() {
		It("matches pipeline by exact event name")
		It("matches multiple pipelines to the same event")
		It("does not match unrelated events")
		It("supports event pattern matching")
	})

	Describe("Template Rendering", func() {
		It("renders step parameters from Go templates")
		It("injects trigger event data into template context")
		It("injects previous step results into template context")
		It("returns error for malformed template syntax")
	})

	Describe("Pipeline Loading", func() {
		It("loads pipeline definitions from configuration")
		It("validates DAG structure on load")
		It("rejects pipelines with circular dependencies")
		It("supports pipeline namespacing")
	})
})
