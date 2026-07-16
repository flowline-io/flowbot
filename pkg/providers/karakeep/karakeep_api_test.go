package karakeep

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

func TestGetWebhookToken(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		want    string
	}{
		{name: "missing config returns empty", configs: json.RawMessage(`{}`), want: ""},
		{name: "reads token", configs: json.RawMessage(`{"karakeep":{"webhook_token":"wh-token"}}`), want: "wh-token"},
		{name: "empty token", configs: json.RawMessage(`{"karakeep":{"webhook_token":""}}`), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			assert.Equal(t, tt.want, GetWebhookToken())
		})
	}
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantURL string
	}{
		{name: "empty config", configs: json.RawMessage(`{}`), wantURL: ""},
		{name: "configured endpoint", configs: json.RawMessage(`{"karakeep":{"endpoint":"https://keep.example.com","api_key":"k"}}`), wantURL: "https://keep.example.com"},
		{name: "endpoint only", configs: json.RawMessage(`{"karakeep":{"endpoint":"https://keep.example.com"}}`), wantURL: "https://keep.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			c := GetClient()
			require.NotNil(t, c)
			assert.Equal(t, tt.wantURL, c.c.BaseURL())
		})
	}
}

func TestKarakeep_GetAllBookmarks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		query   *BookmarksQuery
		body    string
		wantLen int
		wantErr bool
	}{
		{
			name:    "returns bookmarks with query",
			query:   &BookmarksQuery{Limit: 10, Archived: true, Cursor: "c1"},
			body:    `{"bookmarks":[{"id":"b1"}],"nextCursor":"c2"}`,
			wantLen: 1,
		},
		{
			name:    "nil query uses defaults",
			query:   nil,
			body:    `{"bookmarks":[],"nextCursor":""}`,
			wantLen: 0,
		},
		{
			name:    "connection failure",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				c := NewKarakeep("http://127.0.0.2:1", "key")
				_, err := c.GetAllBookmarks(tt.query)
				assert.Error(t, err)
				return
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/bookmarks", r.URL.Path)
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				if tt.query != nil && tt.query.Limit > 0 {
					assert.Equal(t, "10", r.URL.Query().Get("limit"))
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewKarakeep(srv.URL, "test-key")
			resp, err := c.GetAllBookmarks(tt.query)
			require.NoError(t, err)
			assert.Len(t, resp.Bookmarks, tt.wantLen)
		})
	}
}

func TestKarakeep_GetAllTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		wantLen int
		wantErr bool
	}{
		{name: "returns tags", body: `{"tags":[{"id":"t1","name":"go"}]}`, wantLen: 1},
		{name: "empty tags", body: `{"tags":[]}`, wantLen: 0},
		{name: "server unreachable", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				c := NewKarakeep("http://127.0.0.2:1", "key")
				_, err := c.GetAllTags()
				assert.Error(t, err)
				return
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/tags", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewKarakeep(srv.URL, "key")
			tags, err := c.GetAllTags()
			require.NoError(t, err)
			assert.Len(t, tags, tt.wantLen)
		})
	}
}

func TestKarakeep_CreateBookmark(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		url     string
		body    string
		wantID  string
		wantErr bool
	}{
		{name: "creates bookmark", url: "https://example.com", body: `{"id":"bm1","createdAt":"2025-01-01T00:00:00Z"}`, wantID: "bm1"},
		{name: "another url", url: "https://go.dev", body: `{"id":"bm2","createdAt":"2025-01-01T00:00:00Z"}`, wantID: "bm2"},
		{name: "server error", url: "https://bad.example", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				c := NewKarakeep("http://127.0.0.2:1", "key")
				_, err := c.CreateBookmark(tt.url)
				assert.Error(t, err)
				return
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/bookmarks", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewKarakeep(srv.URL, "key")
			bm, err := c.CreateBookmark(tt.url)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, bm.Id)
		})
	}
}

func TestKarakeep_GetBookmark(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		id      string
		body    string
		wantErr bool
	}{
		{name: "returns bookmark", id: "bm1", body: `{"id":"bm1","createdAt":"2025-01-01T00:00:00Z"}`},
		{name: "another id", id: "bm2", body: `{"id":"bm2","createdAt":"2025-01-02T00:00:00Z"}`},
		{name: "not reachable", id: "x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				c := NewKarakeep("http://127.0.0.2:1", "key")
				_, err := c.GetBookmark(tt.id)
				assert.Error(t, err)
				return
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/bookmarks/"+tt.id, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewKarakeep(srv.URL, "key")
			bm, err := c.GetBookmark(tt.id)
			require.NoError(t, err)
			assert.Equal(t, tt.id, bm.Id)
		})
	}
}

func TestKarakeep_CheckUrlExists(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		wantNil bool
		wantID  string
		wantErr bool
	}{
		{name: "url exists", body: `{"bookmarkId":"bm1"}`, wantID: "bm1"},
		{name: "url not found", body: `{"bookmarkId":null}`, wantNil: true},
		{name: "server error", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				c := NewKarakeep("http://127.0.0.2:1", "key")
				_, err := c.CheckUrlExists("https://example.com")
				assert.Error(t, err)
				return
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/bookmarks/check-url", r.URL.Path)
				assert.Equal(t, "https://example.com", r.URL.Query().Get("url"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewKarakeep(srv.URL, "key")
			id, err := c.CheckUrlExists("https://example.com")
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, id)
				return
			}
			require.NotNil(t, id)
			assert.Equal(t, tt.wantID, *id)
		})
	}
}

func TestKarakeep_ArchiveBookmark(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		archived bool
		wantErr  bool
	}{
		{name: "archives bookmark", archived: true},
		{name: "archive false response", archived: false},
		{name: "server error", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				c := NewKarakeep("http://127.0.0.2:1", "key")
				_, err := c.ArchiveBookmark("bm1")
				assert.Error(t, err)
				return
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/bookmarks/bm1", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"archived":` + boolStr(tt.archived) + `}`))
			}))
			defer srv.Close()

			c := NewKarakeep(srv.URL, "key")
			ok, err := c.ArchiveBookmark("bm1")
			require.NoError(t, err)
			assert.Equal(t, tt.archived, ok)
		})
	}
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func TestKarakeep_AttachAndDetachTags(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/bookmarks/bm1/tags":
			_, _ = w.Write([]byte(`{"attached":["go"]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/bookmarks/bm1/tags":
			_, _ = w.Write([]byte(`{"detached":["go"]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewKarakeep(srv.URL, "key")
	attached, err := c.AttachTagsToBookmark("bm1", []string{"go"})
	require.NoError(t, err)
	assert.Equal(t, []string{"go"}, attached)

	detached, err := c.DetachTagsToBookmark("bm1", []string{"go"})
	require.NoError(t, err)
	assert.Equal(t, []string{"go"}, detached)
}

func TestKarakeep_SearchBookmarks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		query   *SearchBookmarksQuery
		wantLen int
	}{
		{name: "search with query", query: &SearchBookmarksQuery{Q: "go", Limit: 5, IncludeContent: true}, wantLen: 1},
		{name: "nil query", query: nil, wantLen: 0},
		{name: "cursor query", query: &SearchBookmarksQuery{Cursor: "c1", SortOrder: "relevance"}, wantLen: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/bookmarks/search", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if tt.wantLen == 0 {
					_, _ = w.Write([]byte(`{"bookmarks":[],"nextCursor":""}`))
					return
				}
				_, _ = w.Write([]byte(`{"bookmarks":[{"id":"b1"}],"nextCursor":""}`))
			}))
			defer srv.Close()

			c := NewKarakeep(srv.URL, "key")
			resp, err := c.SearchBookmarks(tt.query)
			require.NoError(t, err)
			assert.Len(t, resp.Bookmarks, tt.wantLen)
		})
	}
}
