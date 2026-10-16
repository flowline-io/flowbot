package command

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
)

func TestAllLeavesInheritOutput(t *testing.T) {
	t.Parallel()
	root := NewRoot("test")
	var n int
	walkLeaves(t, root, root.Name(), &n)
	require.Greater(t, n, 10)
}

func walkLeaves(t *testing.T, cmd *cobra.Command, path string, n *int) {
	t.Helper()
	if len(cmd.Commands()) == 0 {
		if cmd.Name() == "" || cmd.Name() == "help" {
			return
		}
		*n++
		require.NotNilf(t, lookupFlag(cmd, "output"), "%s missing --output", path)
		require.NotNilf(t, lookupFlag(cmd, "yes"), "%s missing --yes", path)
		require.NotNilf(t, lookupFlag(cmd, "no-input"), "%s missing --no-input", path)
		return
	}
	for _, sub := range cmd.Commands() {
		walkLeaves(t, sub, path+" "+sub.Name(), n)
	}
}

func TestVersionJSONReportsServerError(t *testing.T) {
	cmd := VersionCommand("9.9.9")
	wireClientFlags(cmd)
	cmd.SetArgs([]string{"-o", "json", "--server-url", "http://127.0.0.1:1"})
	out, err := captureStdout(t, cmd.Execute)
	require.NoError(t, err)
	var env map[string]any
	require.NoError(t, sonic.Unmarshal([]byte(out), &env))
	data, ok := env["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "9.9.9", data["client"])
	assert.NotEmpty(t, data["server_error"])
	assert.NotContains(t, data, "server")
}

func TestBookmarkDeleteNoInputExitCode(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	cmd := bookmarkDeleteCommand()
	wireClientFlags(cmd)
	cmd.SetArgs([]string{"--no-input", "bm-1"})
	_, err := captureStdout(t, cmd.Execute)
	require.Error(t, err)
	require.ErrorIs(t, err, utils.ErrConfirmationRequired)
	assert.Equal(t, 7, utils.ExitCode(err))
	obj := utils.ErrorObject(err)
	assert.Equal(t, "confirmation_required", obj["code"])
}

func TestMemoListJSONKeepsEmptyPage(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okJSON(`{"data":[],"page":{"limit":20,"has_more":false}}`)))
	})
	cmd := memoListCommand()
	wireClientFlags(cmd)
	cmd.SetArgs([]string{"-o", "json"})
	out, err := captureStdout(t, cmd.Execute)
	require.NoError(t, err)
	var env map[string]any
	require.NoError(t, sonic.Unmarshal([]byte(out), &env))
	data, ok := env["data"].([]any)
	require.True(t, ok)
	assert.Empty(t, data)
	page, ok := env["page"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, page["has_more"])
	assert.NotContains(t, page, "next_cursor")
}

func TestMemoGetJSONObject(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okJSON(`{"data":{"name":"memos/3","content":"Body"}}`)))
	})
	cmd := memoGetCommand()
	wireClientFlags(cmd)
	cmd.SetArgs([]string{"-o", "json", "memos/3"})
	out, err := captureStdout(t, cmd.Execute)
	require.NoError(t, err)
	var env map[string]any
	require.NoError(t, sonic.Unmarshal([]byte(out), &env))
	data, ok := env["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "memos/3", data["name"])
	assert.NotContains(t, env, "page")
}

func TestPipelineExportJSONYAML(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okJSON(`{"yaml":"name: demo\n"}`)))
	})
	cmd := pipelineExportCommand()
	wireClientFlags(cmd)
	cmd.SetArgs([]string{"-o", "json", "demo"})
	out, err := captureStdout(t, cmd.Execute)
	require.NoError(t, err)
	var env map[string]any
	require.NoError(t, sonic.Unmarshal([]byte(out), &env))
	data, ok := env["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "name: demo\n", data["yaml"])
}

func TestGithubFileJSONBody(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okJSON(`"hello\n"`)))
	})
	cmd := githubFileCommand()
	wireClientFlags(cmd)
	cmd.SetArgs([]string{"-o", "json", "acme", "app", "abc", "README.md"})
	out, err := captureStdout(t, cmd.Execute)
	require.NoError(t, err)
	var env map[string]any
	require.NoError(t, sonic.Unmarshal([]byte(out), &env))
	assert.Equal(t, "hello\n", env["data"])
}

func TestBookmarkGetJSONErrorLeavesStdoutEmpty(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"not found"}`))
	})
	cmd := bookmarkGetCommand()
	wireClientFlags(cmd)
	cmd.SetArgs([]string{"-o", "json", "missing"})
	out, err := captureStdout(t, cmd.Execute)
	require.Error(t, err)
	assert.Empty(t, out)
	var buf bytes.Buffer
	require.NoError(t, utils.PrintError(&buf, true, err))
	assert.Contains(t, buf.String(), `"code": "not_found"`)
}

func TestLoginJSONStdoutIsEnvelope(t *testing.T) {
	setupMockCLI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	cmd := LoginCommand()
	wireClientFlags(cmd)
	cmd.SetArgs([]string{"-o", "json", "--token", "tok", "--server-url", "http://example.test"})
	out, err := captureStdout(t, cmd.Execute)
	require.NoError(t, err)
	assert.NotContains(t, out, "Server URL saved")
	var env map[string]any
	require.NoError(t, sonic.Unmarshal([]byte(out), &env))
	data, ok := env["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "http://example.test", data["server_url"])
}
