package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/validate"
)

func TestBookmarkList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		query      *ListBookmarksQuery
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:  "list with default query",
			query: nil,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"bookmarks":[{"id":"b1"},{"id":"b2"}]}}`))
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "list with limit and cursor",
			query: &ListBookmarksQuery{Limit: 10, Cursor: "abc"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"bookmarks":[{"id":"b1"}]}}`))
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:  "list with archived filter",
			query: &ListBookmarksQuery{Limit: 10, Archived: true},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"bookmarks":[]}}`))
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:       "limit exceeds max",
			query:      &ListBookmarksQuery{Limit: validate.MaxSearchLimit + 1},
			wantErr:    true,
			errContain: "limit exceeds maximum",
		},
		{
			name:       "negative limit",
			query:      &ListBookmarksQuery{Limit: -1},
			wantErr:    true,
			errContain: "limit must be non-negative",
		},
		{
			name:  "api error",
			query: &ListBookmarksQuery{},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"server error"}`))
			},
			wantErr:    true,
			errContain: "server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Bookmark.List(context.Background(), tt.query)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Bookmarks)
			assert.Len(t, result.Bookmarks, tt.wantCount)
		})
	}
}

func TestBookmarkGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantID     string
		wantErr    bool
		errContain string
	}{
		{
			name: "bookmark found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":"b1","title":"test"}}`))
			},
			wantID: "b1",
		},
		{
			name: "bookmark not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"not found"}`))
			},
			wantErr:    true,
			errContain: "not found",
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"error"}`))
			},
			wantErr:    true,
			errContain: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Bookmark.Get(context.Background(), "b1")

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantID, result.Id)
		})
	}
}

func TestBookmarkCreate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		url        string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "create bookmark success",
			url:  "https://example.com",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":"new-b1","title":"Example"}}`))
			},
			wantErr: false,
		},
		{
			name:       "empty url",
			url:        "",
			wantErr:    true,
			errContain: "invalid url",
		},
		{
			name:       "invalid url format",
			url:        "not-a-valid-url",
			wantErr:    true,
			errContain: "invalid url",
		},
		{
			name: "api error",
			url:  "https://example.com",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"url already exists"}`))
			},
			wantErr:    true,
			errContain: "url already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Bookmark.Create(context.Background(), tt.url)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestBookmarkArchive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "archive success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"archived":true}}`))
			},
		},
		{
			name: "unarchive success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"archived":false}}`))
			},
		},
		{
			name: "not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"bookmark not found"}`))
			},
			wantErr:    true,
			errContain: "bookmark not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Bookmark.Archive(context.Background(), "b1")

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestBookmarkAttachTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		tags       []string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "attach tags success",
			tags: []string{"tag1", "tag2"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"attached":["tag1","tag2"]}}`))
			},
		},
		{
			name:       "zero length tag",
			tags:       []string{""},
			handler:    nil,
			wantErr:    true,
			errContain: "tag cannot be empty",
		},
		{
			name:       "empty tag in list",
			tags:       []string{"valid", ""},
			handler:    nil,
			wantErr:    true,
			errContain: "tag cannot be empty",
		},
		{
			name: "api error",
			tags: []string{"tag1"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"error"}`))
			},
			wantErr:    true,
			errContain: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Bookmark.AttachTags(context.Background(), "b1", tt.tags)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestBookmarkDetachTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		tags       []string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "detach tags success",
			tags: []string{"tag1"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"detached":["tag1"]}}`))
			},
		},
		{
			name:       "empty tags validation fails",
			tags:       []string{""},
			handler:    nil,
			wantErr:    true,
			errContain: "tag cannot be empty",
		},
		{
			name: "api error",
			tags: []string{"tag1"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"bookmark not found"}`))
			},
			wantErr:    true,
			errContain: "bookmark not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Bookmark.DetachTags(context.Background(), "b1", tt.tags)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestBookmarkCheckUrl(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		url        string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "url exists",
			url:  "https://example.com",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"bookmarkId":"b-existing"}}`))
			},
		},
		{
			name: "url does not exist",
			url:  "https://new.example.com",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"bookmarkId":null}}`))
			},
		},
		{
			name:       "empty url",
			url:        "",
			wantErr:    true,
			errContain: "invalid url",
		},
		{
			name: "api error",
			url:  "https://example.com",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"error"}`))
			},
			wantErr:    true,
			errContain: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Bookmark.CheckUrl(context.Background(), tt.url)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestBookmarkSearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		query      *SearchBookmarksQuery
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name:  "search success",
			query: &SearchBookmarksQuery{Q: "test", Limit: 10},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"bookmarks":[{"id":"b1","title":"test bookmark"}]}}`))
			},
		},
		{
			name:       "empty query",
			query:      &SearchBookmarksQuery{Q: ""},
			wantErr:    true,
			errContain: "search query is required",
		},
		{
			name:  "nil query defaults",
			query: nil,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"bookmarks":[]}}`))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Bookmark.Search(context.Background(), tt.query)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestValidateListBookmarksQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      *ListBookmarksQuery
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid query",
			query:   &ListBookmarksQuery{Limit: 10, Cursor: "abc"},
			wantErr: false,
		},
		{
			name:       "negative limit",
			query:      &ListBookmarksQuery{Limit: -1},
			wantErr:    true,
			errContain: "limit must be non-negative",
		},
		{
			name:       "limit exceeds max",
			query:      &ListBookmarksQuery{Limit: validate.MaxSearchLimit + 1},
			wantErr:    true,
			errContain: "limit exceeds maximum",
		},
		{
			name:    "zero limit is valid",
			query:   &ListBookmarksQuery{Limit: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateListBookmarksQuery(tt.query)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tags       []string
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid tags",
			tags:    []string{"tag1", "tag2"},
			wantErr: false,
		},
		{
			name:       "empty tag",
			tags:       []string{""},
			wantErr:    true,
			errContain: "tag cannot be empty",
		},
		{
			name:    "empty tag list is valid",
			tags:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateTags(tt.tags)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateSearchBookmarksQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      *SearchBookmarksQuery
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid query",
			query:   &SearchBookmarksQuery{Q: "test", Limit: 10},
			wantErr: false,
		},
		{
			name:       "empty query",
			query:      &SearchBookmarksQuery{Q: ""},
			wantErr:    true,
			errContain: "search query is required",
		},
		{
			name:       "negative limit",
			query:      &SearchBookmarksQuery{Q: "test", Limit: -1},
			wantErr:    true,
			errContain: "limit must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSearchBookmarksQuery(tt.query)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
