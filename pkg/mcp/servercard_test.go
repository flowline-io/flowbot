package mcp

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerCardURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "mcp path", in: "https://flowbot.lan/mcp", want: "https://flowbot.lan/mcp/server-card"},
		{name: "trailing slash", in: "http://127.0.0.1:6060/mcp/", want: "http://127.0.0.1:6060/mcp/server-card"},
		{name: "strips query user and fragment", in: "https://user:secret@flowbot.lan/mcp?x=1#card", want: "https://flowbot.lan/mcp/server-card"},
		{name: "rejects stdio", in: "stdio://local", wantErr: true},
		{name: "rejects empty", in: "  ", wantErr: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ServerCardURL(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, ServerCardURLOrEmpty(tt.in))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want, ServerCardURLOrEmpty(tt.in))
		})
	}
}

func TestNewServerCard(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		endpoint string
		version  string
		wantURL  string
		wantErr  bool
	}{
		{
			name:     "uses the endpoint without query or userinfo",
			endpoint: "https://flowbot.lan/mcp?token=secret",
			version:  "v0.99.13",
			wantURL:  "https://flowbot.lan/mcp",
		},
		{
			name:     "rejects a blank version",
			endpoint: "https://flowbot.lan/mcp",
			version:  " ",
			wantErr:  true,
		},
		{
			name:     "rejects a non-http endpoint",
			endpoint: "ftp://flowbot.lan/mcp",
			version:  "v0.99.13",
			wantErr:  true,
		},
	}
	namePattern := regexp.MustCompile(`^[a-zA-Z0-9.-]+/[a-zA-Z0-9._-]+$`)
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			card, err := NewServerCard(tt.endpoint, tt.version)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, ServerCardSchema, card.Schema)
			assert.Equal(t, ServerCardName, card.Name)
			assert.True(t, namePattern.MatchString(card.Name))
			assert.Equal(t, "v0.99.13", card.Version)
			assert.NotEmpty(t, card.Description)
			assert.LessOrEqual(t, len(card.Description), 100)
			assert.NotContains(t, card.Description, "pipeline")
			require.Len(t, card.Remotes, 1)
			assert.Equal(t, "streamable-http", card.Remotes[0].Type)
			assert.Equal(t, tt.wantURL, card.Remotes[0].URL)
			assert.Equal(t, []string{SupportedProtocolVersion}, card.Remotes[0].SupportedProtocolVersions)
			assert.NotContains(t, card.Remotes[0].URL, "secret")
		})
	}
}

func TestRenderSnippetIncludesServerCard(t *testing.T) {
	t.Parallel()
	got, err := RenderSnippet(ClientGeneric, SnippetOptions{URL: "https://flowbot.lan/mcp"})
	require.NoError(t, err)
	assert.Equal(t, "https://flowbot.lan/mcp/server-card", got.ServerCard)
	assert.NotContains(t, got.Snippet, "server-card")

	nonHTTP, err := RenderSnippet(ClientGeneric, SnippetOptions{URL: "ftp://flowbot.lan/mcp"})
	require.NoError(t, err)
	assert.Empty(t, nonHTTP.ServerCard)
}
