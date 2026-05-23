package provider_event_source_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProviderEventSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider Event Source Suite")
}
