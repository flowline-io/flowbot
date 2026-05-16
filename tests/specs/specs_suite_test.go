//go:build integration
// +build integration

// Package specs provides Ginkgo BDD acceptance tests for Flowbot.
// These tests require Docker to be running.
package specs

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSpecs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flowbot Acceptance Suite")
}
