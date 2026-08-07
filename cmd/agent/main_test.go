package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCLI(t *testing.T) {
	got, err := parseCLI(cliInput{
		Print:        true,
		OutputFormat: "text",
		PromptArgs:   []string{"hello", "world"},
		Workspace:    t.TempDir(),
	})
	require.NoError(t, err)
	assert.Equal(t, "hello world", got.Prompt)
	assert.NotEmpty(t, got.Workspace)

	_, err = parseCLI(cliInput{Print: false, PromptArgs: []string{"x"}})
	require.Error(t, err)

	_, err = parseCLI(cliInput{Print: true, OutputFormat: "json", PromptArgs: []string{"x"}})
	require.Error(t, err)

	_, err = parseCLI(cliInput{Print: true, OutputFormat: "text"})
	require.Error(t, err)

	got, err = parseCLI(cliInput{
		Print:        true,
		Force:        true,
		OutputFormat: "text",
		PromptArgs:   []string{"do it"},
		Workspace:    t.TempDir(),
	})
	require.NoError(t, err)
	assert.True(t, got.Force)
}
