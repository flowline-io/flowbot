package confluence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

func TestGetClient(t *testing.T) {
	providers.Configs = json.RawMessage(`{"confluence":{"site_url":"https://test.atlassian.net","email":"user@example.com","api_token":"tok"}}`)
	c := GetClient()
	require.NotNil(t, c)
	assert.Equal(t, "https://test.atlassian.net/wiki/rest/api", c.c.BaseURL())

	providers.Configs = json.RawMessage(`{"confluence":{"site_url":"https://test.atlassian.net"}}`)
	assert.Nil(t, GetClient())
}

func TestListSpaces(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/space", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":1,"key":"DEV","name":"Dev"}],"start":0,"limit":25,"size":1}`))
	}))
	defer srv.Close()

	client := NewConfluence(srv.URL, "user@example.com", "token")
	client.c.SetBaseURL(srv.URL)

	resp, err := client.ListSpaces(context.Background(), 0, 25)
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "DEV", resp.Results[0].Key)
}

func TestCreatePage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/content", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"42","title":"Hello","type":"page"}`))
	}))
	defer srv.Close()

	client := NewConfluence(srv.URL, "user@example.com", "token")
	client.c.SetBaseURL(srv.URL)

	page, err := client.CreatePage(context.Background(), "DEV", "Hello", "<p>hi</p>")
	require.NoError(t, err)
	assert.Equal(t, "42", page.ID)
}

func TestGetWebhookToken(t *testing.T) {
	providers.Configs = json.RawMessage(`{"confluence":{"webhook_token":"secret"}}`)
	assert.Equal(t, "secret", GetWebhookToken())
}
