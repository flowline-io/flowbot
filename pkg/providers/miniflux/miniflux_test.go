package miniflux

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	rssClient "miniflux.app/v2/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

func TestNewMiniflux(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		apiKey   string
	}{
		{name: "creates client with endpoint", endpoint: "https://miniflux.example.com", apiKey: "key"},
		{name: "empty api key allowed", endpoint: "https://miniflux.example.com", apiKey: ""},
		{name: "local endpoint", endpoint: "http://127.0.0.1:8080", apiKey: "dev-key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewMiniflux(tt.endpoint, tt.apiKey)
			require.NotNil(t, c)
			assert.Equal(t, tt.apiKey, c.apiKey)
			assert.NotNil(t, c.c)
		})
	}
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantKey string
	}{
		{
			name:    "reads endpoint and api key from config",
			configs: json.RawMessage(`{"miniflux":{"endpoint":"https://mf.example.com","api_key":"secret"}}`),
			wantKey: "secret",
		},
		{
			name:    "empty config still returns client",
			configs: json.RawMessage(`{}`),
			wantKey: "",
		},
		{
			name:    "endpoint only",
			configs: json.RawMessage(`{"miniflux":{"endpoint":"https://mf.example.com"}}`),
			wantKey: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			c := GetClient()
			require.NotNil(t, c)
			assert.Equal(t, tt.wantKey, c.apiKey)
		})
	}
}

func TestGetWebhookSecret(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		want    string
	}{
		{name: "missing config returns empty", configs: json.RawMessage(`{}`), want: ""},
		{name: "reads secret", configs: json.RawMessage(`{"miniflux":{"webhook_secret":"hmac-secret"}}`), want: "hmac-secret"},
		{name: "empty secret value", configs: json.RawMessage(`{"miniflux":{"webhook_secret":""}}`), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			assert.Equal(t, tt.want, GetWebhookSecret())
		})
	}
}

func TestMiniflux_GetFeeds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantLen    int
	}{
		{
			name:       "returns feeds list",
			statusCode: http.StatusOK,
			body:       `[{"id":1,"title":"Blog","feed_url":"https://example.com/rss"}]`,
			wantLen:    1,
		},
		{
			name:       "empty feeds list",
			statusCode: http.StatusOK,
			body:       `[]`,
			wantLen:    0,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error":"boom"}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/v1/feeds", r.URL.Path)
				assert.Equal(t, "test-key", r.Header.Get("X-Auth-Token"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewMiniflux(srv.URL, "test-key")
			feeds, err := c.GetFeeds()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, feeds, tt.wantLen)
		})
	}
}

func TestMiniflux_GetFeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		feedID     int64
		statusCode int
		body       string
		wantErr    bool
		wantTitle  string
	}{
		{
			name:       "returns single feed",
			feedID:     42,
			statusCode: http.StatusOK,
			body:       `{"id":42,"title":"News","feed_url":"https://news.example/rss"}`,
			wantTitle:  "News",
		},
		{
			name:       "not found",
			feedID:     99,
			statusCode: http.StatusNotFound,
			body:       `{"error":"not found"}`,
			wantErr:    true,
		},
		{
			name:       "feed id one",
			feedID:     1,
			statusCode: http.StatusOK,
			body:       `{"id":1,"title":"Default","feed_url":"https://example.com/rss"}`,
			wantTitle:  "Default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/feeds/"+strconv.FormatInt(tt.feedID, 10), r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewMiniflux(srv.URL, "key")
			feed, err := c.GetFeed(tt.feedID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTitle, feed.Title)
		})
	}
}

func TestMiniflux_CreateFeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantID     int64
	}{
		{name: "creates feed", statusCode: http.StatusCreated, body: `{"feed_id":7}`, wantID: 7},
		{name: "creates feed id two", statusCode: http.StatusCreated, body: `{"feed_id":2}`, wantID: 2},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"error":"invalid"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v1/feeds", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewMiniflux(srv.URL, "key")
			id, err := c.CreateFeed(&rssClient.FeedCreationRequest{FeedURL: "https://example.com/rss"})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestMiniflux_MarkAllAsRead(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		feedsBody     string
		markStatus    int
		wantErr       bool
		wantMarkCalls int
	}{
		{
			name:          "marks each feed as read",
			feedsBody:     `[{"id":1,"title":"A"},{"id":2,"title":"B"}]`,
			markStatus:    http.StatusOK,
			wantMarkCalls: 2,
		},
		{
			name:          "continues when mark fails",
			feedsBody:     `[{"id":1,"title":"A"}]`,
			markStatus:    http.StatusInternalServerError,
			wantMarkCalls: 1,
		},
		{
			name:      "feed list error",
			feedsBody: ``,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			markCalls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/feeds":
					if tt.feedsBody == "" {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tt.feedsBody))
				default:
					markCalls++
					w.WriteHeader(tt.markStatus)
				}
			}))
			defer srv.Close()

			c := NewMiniflux(srv.URL, "key")
			err := c.MarkAllAsRead()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMarkCalls, markCalls)
		})
	}
}

func TestMiniflux_GetEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantTotal  int
	}{
		{
			name:       "returns entries",
			statusCode: http.StatusOK,
			body:       `{"total":1,"entries":[{"id":1,"title":"Entry"}]}`,
			wantTotal:  1,
		},
		{
			name:       "empty entries",
			statusCode: http.StatusOK,
			body:       `{"total":0,"entries":[]}`,
			wantTotal:  0,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/entries", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewMiniflux(srv.URL, "key")
			result, err := c.GetEntries(&rssClient.Filter{Status: "unread"})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, result.Total)
		})
	}
}
