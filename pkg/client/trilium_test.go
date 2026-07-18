package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/validate"
)

func TestTriliumList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		query      *ListNotesQuery
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
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[{"id":"n1","title":"A"},{"id":"n2","title":"B"}],"page":{"limit":20,"has_more":false}}}`))
			},
			wantCount: 2,
		},
		{
			name:  "list with limit query and cursor",
			query: &ListNotesQuery{Limit: 10, Cursor: "abc", Query: "hello"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/trilium", r.URL.Path)
				assert.Equal(t, "10", r.URL.Query().Get("limit"))
				assert.Equal(t, "abc", r.URL.Query().Get("cursor"))
				assert.Equal(t, "hello", r.URL.Query().Get("query"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[{"id":"n1"}],"page":{"limit":10,"has_more":true,"next_cursor":"next"}}}`))
			},
			wantCount: 1,
		},
		{
			name:       "negative limit",
			query:      &ListNotesQuery{Limit: -1},
			wantErr:    true,
			errContain: "limit must be non-negative",
		},
		{
			name:       "limit exceeds max",
			query:      &ListNotesQuery{Limit: validate.MaxSearchLimit + 1},
			wantErr:    true,
			errContain: "limit exceeds maximum",
		},
		{
			name:  "api error",
			query: &ListNotesQuery{},
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
			result, err := c.Trilium.List(context.Background(), tt.query)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Items, tt.wantCount)
		})
	}
}

func TestTriliumGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         string
		handler    http.HandlerFunc
		wantID     string
		wantErr    bool
		errContain string
	}{
		{
			name: "note found",
			id:   "n1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/trilium/n1", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"id":"n1","title":"Hello"}}}`))
			},
			wantID: "n1",
		},
		{
			name:       "empty id",
			id:         "",
			wantErr:    true,
			errContain: "id is required",
		},
		{
			name: "not found",
			id:   "missing",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"not found"}`))
			},
			wantErr:    true,
			errContain: "not found",
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
			note, err := c.Trilium.Get(context.Background(), tt.id)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, note)
			assert.Equal(t, tt.wantID, note.ID)
		})
	}
}

func TestTriliumCreate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		req        *CreateNoteRequest
		handler    http.HandlerFunc
		wantID     string
		wantErr    bool
		errContain string
	}{
		{
			name: "create success",
			req:  &CreateNoteRequest{Title: "New", Content: "body", ParentNoteID: "root"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/service/trilium", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"id":"n9","title":"New"}}}`))
			},
			wantID: "n9",
		},
		{
			name:       "nil request",
			req:        nil,
			wantErr:    true,
			errContain: "request is required",
		},
		{
			name:       "empty title",
			req:        &CreateNoteRequest{Title: ""},
			wantErr:    true,
			errContain: "title is required",
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
			note, err := c.Trilium.Create(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, note)
			assert.Equal(t, tt.wantID, note.ID)
		})
	}
}

func TestTriliumUpdateDeleteSearchContentHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, c *Client, serverURL string)
	}{
		{
			name: "update success",
			run: func(t *testing.T, _ *Client, _ string) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPatch, r.Method)
					assert.Equal(t, "/service/trilium/n1", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"id":"n1","title":"Updated"}}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				note, err := c.Trilium.Update(context.Background(), "n1", &UpdateNoteRequest{Title: "Updated"})
				require.NoError(t, err)
				assert.Equal(t, "Updated", note.Title)
			},
		},
		{
			name: "update missing id",
			run: func(t *testing.T, _ *Client, _ string) {
				c := NewClient("http://example.com", "token")
				_, err := c.Trilium.Update(context.Background(), "", &UpdateNoteRequest{Title: "X"})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "id is required")
			},
		},
		{
			name: "delete success",
			run: func(t *testing.T, _ *Client, _ string) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodDelete, r.Method)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok"}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				require.NoError(t, c.Trilium.Delete(context.Background(), "n1"))
			},
		},
		{
			name: "search success",
			run: func(t *testing.T, _ *Client, _ string) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/service/trilium/search", r.URL.Path)
					assert.Equal(t, "todo", r.URL.Query().Get("q"))
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[{"id":"n2","title":"Todo"}]}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				result, err := c.Trilium.Search(context.Background(), &SearchNotesQuery{Q: "todo"})
				require.NoError(t, err)
				require.Len(t, result.Items, 1)
				assert.Equal(t, "n2", result.Items[0].ID)
			},
		},
		{
			name: "search empty query",
			run: func(t *testing.T, _ *Client, _ string) {
				c := NewClient("http://example.com", "token")
				_, err := c.Trilium.Search(context.Background(), &SearchNotesQuery{Q: ""})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "query is required")
			},
		},
		{
			name: "get and set content",
			run: func(t *testing.T, _ *Client, _ string) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case http.MethodGet:
						assert.Equal(t, "/service/trilium/n1/content", r.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"status":"ok","data":{"data":"hello body"}}`))
					case http.MethodPut:
						body, err := io.ReadAll(r.Body)
						assert.NoError(t, err)
						assert.Equal(t, "new body", string(body))
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"status":"ok"}`))
					default:
						t.Fatalf("unexpected method %s", r.Method)
					}
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				content, err := c.Trilium.GetContent(context.Background(), "n1")
				require.NoError(t, err)
				assert.Equal(t, "hello body", content)
				require.NoError(t, c.Trilium.SetContent(context.Background(), "n1", "new body"))
			},
		},
		{
			name: "health success",
			run: func(t *testing.T, _ *Client, _ string) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/service/trilium/health", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"id":"home","title":"Trilium Notes 0.63","type":"app_info"}}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				info, err := c.Trilium.Health(context.Background())
				require.NoError(t, err)
				assert.Contains(t, info.Title, "Trilium")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, nil, "")
		})
	}
}

func TestValidateListNotesQuery(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		query      *ListNotesQuery
		wantErr    bool
		errContain string
	}{
		{name: "valid", query: &ListNotesQuery{Limit: 10}},
		{name: "negative limit", query: &ListNotesQuery{Limit: -1}, wantErr: true, errContain: "non-negative"},
		{name: "limit too large", query: &ListNotesQuery{Limit: validate.MaxSearchLimit + 1}, wantErr: true, errContain: "maximum"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateListNotesQuery(tt.query)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				return
			}
			require.NoError(t, err)
		})
	}
}
