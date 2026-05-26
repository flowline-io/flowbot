package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/validate"
)

func TestSearchSearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		query      string
		source     string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:   "search returns results",
			query:  "test",
			source: "",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":{
						"hits":[
							{"id":"1","title":"result1","source":"docs","url":"http://a"},
							{"id":"2","title":"result2","source":"docs","url":"http://b"}
						]
					}
				}`))
			},
			wantCount: 2,
		},
		{
			name:   "search with source filter",
			query:  "test",
			source: "docs",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"hits":[{"id":"1","title":"result1","source":"docs","url":"http://a"}]}}`))
			},
			wantCount: 1,
		},
		{
			name:   "search no results",
			query:  "no-match",
			source: "",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"hits":[]}}`))
			},
			wantCount: 0,
		},
		{
			name:       "empty query",
			query:      "",
			wantErr:    true,
			errContain: "query is required",
		},
		{
			name:  "api error",
			query: "test",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"search unavailable"}`))
			},
			wantErr:    true,
			errContain: "search unavailable",
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
			result, err := c.Search.Search(context.Background(), tt.query, tt.source)

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

func TestSearchAutocomplete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		query      string
		handler    http.HandlerFunc
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name:  "autocomplete returns results",
			query: "te",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"status":"ok",
					"data":{
						"hits":[
							{"id":"1","title":"test","source":"docs"},
							{"id":"2","title":"testing","source":"docs"},
							{"id":"3","title":"temp","source":"docs"}
						]
					}
				}`))
			},
			wantCount: 3,
		},
		{
			name:  "autocomplete empty results",
			query: "zzz",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"hits":[]}}`))
			},
			wantCount: 0,
		},
		{
			name:       "empty query",
			query:      "",
			wantErr:    true,
			errContain: "query is required",
		},
		{
			name:  "api error",
			query: "te",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"failed","message":"autocomplete unavailable"}`))
			},
			wantErr:    true,
			errContain: "autocomplete unavailable",
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
			result, err := c.Search.Autocomplete(context.Background(), tt.query, "")

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

func TestValidateSearchQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid query",
			query:   "test",
			wantErr: false,
		},
		{
			name:       "empty query",
			query:      "",
			wantErr:    true,
			errContain: "query is required",
		},
		{
			name:    "query at max length boundary",
			query:   string(make([]byte, validate.QueryMaxLen)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSearchQuery(tt.query)
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

func TestExtractSearchResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      types.KV
		wantCount int
		wantID    string
	}{
		{
			name: "extract multiple results",
			data: types.KV{
				"hits": []any{
					map[string]any{"id": "1", "title": "t1", "source": "s1", "url": "u1", "content": "c1"},
					map[string]any{"id": "2", "title": "t2", "source": "s2", "url": "u2"},
				},
			},
			wantCount: 2,
			wantID:    "1",
		},
		{
			name:      "no hits key returns empty",
			data:      types.KV{},
			wantCount: 0,
		},
		{
			name: "hits not an array returns empty",
			data: types.KV{
				"hits": "not-an-array",
			},
			wantCount: 0,
		},
		{
			name: "array with non-map elements skipped",
			data: types.KV{
				"hits": []any{
					"not-a-map",
					map[string]any{"id": "1", "title": "valid", "source": "s1"},
				},
			},
			wantCount: 1,
			wantID:    "1",
		},
		{
			name: "result with content field",
			data: types.KV{
				"hits": []any{
					map[string]any{"id": "1", "title": "t", "source": "s", "content": "full content here"},
				},
			},
			wantCount: 1,
			wantID:    "1",
		},
		{
			name:      "nil data",
			data:      nil,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := extractSearchResults(tt.data)
			assert.Len(t, results, tt.wantCount)
			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantID, results[0].ID)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want string
	}{
		{
			name: "key exists with string value",
			m:    map[string]any{"name": "hello"},
			key:  "name",
			want: "hello",
		},
		{
			name: "key exists with non-string value",
			m:    map[string]any{"count": 42},
			key:  "count",
			want: "",
		},
		{
			name: "key missing",
			m:    map[string]any{"other": "value"},
			key:  "missing",
			want: "",
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getString(tt.m, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}
