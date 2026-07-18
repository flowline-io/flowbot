package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderListFeeds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "list feeds with results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"title":"feed1","feed_url":"https://a.test/feed"},{"id":2,"title":"feed2","feed_url":"https://b.test/feed"}]}`))
			},
			wantCount: 2,
		},
		{
			name: "empty feed list",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"reader unavailable"}`))
			},
			wantErr:    true,
			errContain: "reader unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Reader.ListFeeds(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestReaderGetFeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int64
		handler    http.HandlerFunc
		wantTitle  string
		wantErr    bool
		errContain string
	}{
		{
			name: "feed found via list",
			id:   1,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"title":"My Feed","feed_url":"https://example.com/feed"},{"id":2,"title":"Other","feed_url":"https://other.test/feed"}]}`))
			},
			wantTitle: "My Feed",
		},
		{
			name: "feed not found in list",
			id:   99,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"title":"My Feed","feed_url":"https://example.com/feed"}]}`))
			},
			wantErr:    true,
			errContain: "feed not found",
		},
		{
			name:       "invalid id",
			id:         0,
			wantErr:    true,
			errContain: "id must be positive",
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
			result, err := c.Reader.GetFeed(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantTitle, result.Title)
		})
	}
}

func TestReaderCreateFeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		req        *CreateFeedRequest
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "create feed success",
			req:  &CreateFeedRequest{FeedURL: "https://example.com/feed", CategoryID: 1},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":42,"title":"Example","feed_url":"https://example.com/feed"}}`))
			},
		},
		{
			name:       "empty url",
			req:        &CreateFeedRequest{FeedURL: "", CategoryID: 1},
			wantErr:    true,
			errContain: "feed_url is required",
		},
		{
			name:       "invalid url format",
			req:        &CreateFeedRequest{FeedURL: "not-a-url", CategoryID: 1},
			wantErr:    true,
			errContain: "invalid feed_url",
		},
		{
			name: "api error",
			req:  &CreateFeedRequest{FeedURL: "https://example.com/feed", CategoryID: 1},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"duplicate feed"}`))
			},
			wantErr:    true,
			errContain: "duplicate feed",
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
			result, err := c.Reader.CreateFeed(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, int64(42), result.ID)
		})
	}
}

func TestReaderListEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		query      *ListEntriesQuery
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:  "list entries success",
			query: &ListEntriesQuery{Status: "unread", Limit: 10},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, "status=unread")
				assert.Contains(t, r.URL.RawQuery, "limit=10")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"title":"entry1","status":"unread"},{"id":2,"title":"entry2","status":"unread"}]}`))
			},
			wantCount: 2,
		},
		{
			name:  "list with feed filter",
			query: &ListEntriesQuery{FeedID: 5, Limit: 5},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, "feed_id=5")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":3,"title":"feed entry","feed_title":"Blog"}]}`))
			},
			wantCount: 1,
		},
		{
			name:  "empty entries",
			query: &ListEntriesQuery{},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Reader.ListEntries(context.Background(), tt.query)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestReaderUpdateEntriesStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		req        *UpdateEntriesRequest
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "update success",
			req:  &UpdateEntriesRequest{EntryIDs: []int64{1, 2}, Status: "read"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "empty entry ids",
			req:        &UpdateEntriesRequest{EntryIDs: nil, Status: "read"},
			wantErr:    true,
			errContain: "entry_ids is required",
		},
		{
			name:       "empty status",
			req:        &UpdateEntriesRequest{EntryIDs: []int64{1}, Status: ""},
			wantErr:    true,
			errContain: "status is required",
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
			result, err := c.Reader.UpdateEntriesStatus(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, result.Success)
		})
	}
}

func TestReaderGetFeedEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		feedID     int64
		query      *GetFeedEntriesQuery
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:   "entries for feed",
			feedID: 1,
			query:  &GetFeedEntriesQuery{Limit: 10},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/miniflux/entries", r.URL.Path)
				assert.Contains(t, r.URL.RawQuery, "feed_id=1")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"title":"entry1"},{"id":2,"title":"entry2"}]}`))
			},
			wantCount: 2,
		},
		{
			name:       "invalid feed id",
			feedID:     0,
			wantErr:    true,
			errContain: "feed_id must be positive",
		},
		{
			name:   "empty feed entries",
			feedID: 3,
			query:  &GetFeedEntriesQuery{Status: "read"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
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
			result, err := c.Reader.GetFeedEntries(context.Background(), tt.feedID, tt.query)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}
