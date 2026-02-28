package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 14)
}

func TestCommandRules_AllDefines(t *testing.T) {
	expected := []string{
		"dev setting", "id", "form test", "queue test",
		"instruct test", "page test", "docker test", "torrent test",
		"slash test", "llm test", "notify test", "fs test",
		"event test", "test",
	}
	defines := make(map[string]bool)
	for _, r := range commandRules {
		defines[r.Define] = true
	}
	for _, e := range expected {
		assert.True(t, defines[e], "expected define %q to exist", e)
	}
}

func TestCommandRules_AllHandlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}

func TestCommandRules_AllHaveHelp(t *testing.T) {
	for _, r := range commandRules {
		assert.NotEmpty(t, r.Help, "help for %q should not be empty", r.Define)
	}
}
