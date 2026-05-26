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
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"title":"feed1"},{"id":2,"title":"feed2"}]}`))
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
		handler    http.HandlerFunc
		wantTitle  string
		wantErr    bool
		errContain string
	}{
		{
			name: "feed found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"title":"My Feed","feed_url":"https://example.com/feed"}}`))
			},
			wantTitle: "My Feed",
		},
		{
			name: "feed not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"feed not found"}`))
			},
			wantErr:    true,
			errContain: "feed not found",
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
			result, err := c.Reader.GetFeed(context.Background(), 1)

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
		wantID     int64
		wantErr    bool
		errContain string
	}{
		{
			name: "create feed success",
			req:  &CreateFeedRequest{FeedURL: "https://example.com/feed", CategoryID: 1},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":42}}`))
			},
			wantID: 42,
		},
		{
			name:       "empty feed url",
			req:        &CreateFeedRequest{FeedURL: "", CategoryID: 1},
			wantErr:    true,
			errContain: "feed_url is required",
		},
		{
			name:       "invalid feed url",
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
				_, _ = w.Write([]byte(`{"status":"failed","message":"feed already exists"}`))
			},
			wantErr:    true,
			errContain: "feed already exists",
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
			assert.Equal(t, tt.wantID, result.ID)
		})
	}
}

func TestReaderUpdateFeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "update feed success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"title":"Updated Feed"}}`))
			},
		},
		{
			name: "feed not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"feed not found"}`))
			},
			wantErr:    true,
			errContain: "feed not found",
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
			result, err := c.Reader.UpdateFeed(context.Background(), 1, &UpdateFeedRequest{Title: "Updated Feed"})

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

func TestReaderRefreshFeed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "refresh success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name: "refresh failed",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"refresh failed"}`))
			},
			wantErr:    true,
			errContain: "refresh failed",
		},
		{
			name: "feed not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"feed not found"}`))
			},
			wantErr:    true,
			errContain: "feed not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Reader.RefreshFeed(context.Background(), 1)

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

func TestReaderListEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		query      *ListEntriesQuery
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name:  "list entries with no filters",
			query: nil,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"total":5,"entries":[{"id":1,"title":"entry1"}]}}`))
			},
		},
		{
			name: "list entries with status filter",
			query: &ListEntriesQuery{
				Status: "unread",
				Limit:  10,
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"total":2,"entries":[{"id":1,"title":"unread1"},{"id":2,"title":"unread2"}]}}`))
			},
		},
		{
			name:  "server error",
			query: nil,
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
			result, err := c.Reader.ListEntries(context.Background(), tt.query)

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
			name: "update status success",
			req:  &UpdateEntriesRequest{EntryIDs: []int64{1, 2}, Status: "read"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"success":true}}`))
			},
		},
		{
			name:       "empty entry ids",
			req:        &UpdateEntriesRequest{EntryIDs: []int64{}, Status: "read"},
			wantErr:    true,
			errContain: "entry_ids is required",
		},
		{
			name:       "empty status",
			req:        &UpdateEntriesRequest{EntryIDs: []int64{1}, Status: ""},
			wantErr:    true,
			errContain: "status is required",
		},
		{
			name: "api error",
			req:  &UpdateEntriesRequest{EntryIDs: []int64{1}, Status: "read"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":"failed","message":"invalid status"}`))
			},
			wantErr:    true,
			errContain: "invalid status",
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
		query      *GetFeedEntriesQuery
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name:  "get feed entries no filters",
			query: nil,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"total":3,"entries":[{"id":1,"title":"entry1"}]}}`))
			},
		},
		{
			name: "get feed entries with filters",
			query: &GetFeedEntriesQuery{
				Status: "unread",
				Limit:  20,
				Order:  "published_at",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"total":1,"entries":[{"id":2,"title":"unread entry"}]}}`))
			},
		},
		{
			name:  "feed not found",
			query: nil,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","message":"feed not found"}`))
			},
			wantErr:    true,
			errContain: "feed not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := NewClient(server.URL, "token")
			result, err := c.Reader.GetFeedEntries(context.Background(), 1, tt.query)

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
