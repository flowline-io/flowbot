package memos

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
	tests := []struct {
		name    string
		configs json.RawMessage
		wantNil bool
		wantURL string
	}{
		{
			name:    "empty config returns nil",
			configs: json.RawMessage(`{}`),
			wantNil: true,
		},
		{
			name:    "configured endpoint returns client",
			configs: json.RawMessage(`{"memos":{"endpoint":"https://memos.example.com"}}`),
			wantNil: false,
			wantURL: "https://memos.example.com",
		},
		{
			name:    "endpoint with token returns client",
			configs: json.RawMessage(`{"memos":{"endpoint":"https://memos.example.com","token":"test-token-123"}}`),
			wantNil: false,
			wantURL: "https://memos.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			c := GetClient()
			if tt.wantNil {
				assert.Nil(t, c)
				return
			}
			require.NotNil(t, c)
			assert.Equal(t, tt.wantURL, c.c.BaseURL())
		})
	}
}

func TestNewMemos(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		token    string
		wantNil  bool
		wantURL  string
	}{
		{
			name:     "explicit endpoint creates client",
			endpoint: "https://memos.example.com",
			token:    "",
			wantNil:  false,
			wantURL:  "https://memos.example.com",
		},
		{
			name:     "empty endpoint returns nil",
			endpoint: "",
			token:    "",
			wantNil:  true,
		},
		{
			name:     "with auth token creates client",
			endpoint: "https://memos.example.com",
			token:    "test-token",
			wantNil:  false,
			wantURL:  "https://memos.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewMemos(tt.endpoint, tt.token)
			if tt.wantNil {
				assert.Nil(t, c)
				return
			}
			require.NotNil(t, c)
			assert.Equal(t, tt.wantURL, c.c.BaseURL())
		})
	}
}

func TestCreateMemo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		content    string
		visibility string
		statusCode int
		respBody   string
		wantErr    bool
	}{
		{
			name:       "creates memo with content and visibility",
			content:    "Hello world",
			visibility: "PUBLIC",
			statusCode: http.StatusOK,
			respBody:   `{"name":"memos/1","content":"Hello world","visibility":"PUBLIC"}`,
		},
		{
			name:       "empty visibility defaults to PRIVATE",
			content:    "private note",
			visibility: "",
			statusCode: http.StatusOK,
			respBody:   `{"name":"memos/2","content":"private note","visibility":"PRIVATE"}`,
		},
		{
			name:       "server error returns error",
			content:    "test",
			visibility: "PRIVATE",
			statusCode: http.StatusInternalServerError,
			respBody:   `{"error":"internal"}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/memos", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := NewMemos(srv.URL, "")
			resp, err := c.CreateMemo(context.Background(), tt.content, tt.visibility)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestGetMemo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		memoName   string
		statusCode int
		respBody   string
		wantErr    bool
		wantName   string
	}{
		{
			name:       "gets memo by resource name",
			memoName:   "memos/42",
			statusCode: http.StatusOK,
			respBody:   `{"name":"memos/42","content":"found memo","visibility":"PRIVATE"}`,
			wantName:   "memos/42",
		},
		{
			name:       "not found returns error",
			memoName:   "memos/999",
			statusCode: http.StatusNotFound,
			respBody:   `{"error":"not found"}`,
			wantErr:    true,
		},
		{
			name:       "server error returns error",
			memoName:   "memos/1",
			statusCode: http.StatusInternalServerError,
			respBody:   `{}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/"+tt.memoName, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := NewMemos(srv.URL, "")
			resp, err := c.GetMemo(context.Background(), tt.memoName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, resp.Name)
		})
	}
}

func TestListMemos(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		params     ListMemosParams
		statusCode int
		respBody   string
		wantErr    bool
		wantCount  int
		wantToken  string
	}{
		{
			name: "lists memos with pagination",
			params: ListMemosParams{
				PageSize:  10,
				PageToken: "token-abc",
			},
			statusCode: http.StatusOK,
			respBody:   `{"memos":[{"name":"memos/1","content":"first"},{"name":"memos/2","content":"second"}],"nextPageToken":"token-next"}`,
			wantCount:  2,
			wantToken:  "token-next",
		},
		{
			name:       "empty params lists default page",
			params:     ListMemosParams{},
			statusCode: http.StatusOK,
			respBody:   `{"memos":[],"nextPageToken":""}`,
			wantCount:  0,
		},
		{
			name:       "unauthorized returns error",
			params:     ListMemosParams{},
			statusCode: http.StatusUnauthorized,
			respBody:   `{}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/memos", r.URL.Path)
				if tt.params.PageSize > 0 {
					assert.Equal(t, "10", r.URL.Query().Get("pageSize"))
				}
				if tt.params.PageToken != "" {
					assert.Equal(t, "token-abc", r.URL.Query().Get("pageToken"))
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := NewMemos(srv.URL, "")
			resp, err := c.ListMemos(context.Background(), tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, resp.Memos, tt.wantCount)
			assert.Equal(t, tt.wantToken, resp.NextPageToken)
		})
	}
}

func TestUpdateMemo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		memoName   string
		content    string
		visibility string
		pinned     *bool
		fields     []string
		statusCode int
		respBody   string
		wantErr    bool
	}{
		{
			name:       "updates memo content",
			memoName:   "memos/1",
			content:    "updated content",
			visibility: "",
			pinned:     nil,
			fields:     []string{"content"},
			statusCode: http.StatusOK,
			respBody:   `{"name":"memos/1","content":"updated content","visibility":"PRIVATE"}`,
		},
		{
			name:       "updates memo with pinned flag",
			memoName:   "memos/2",
			content:    "",
			visibility: "",
			pinned:     new(true),
			fields:     []string{"pinned"},
			statusCode: http.StatusOK,
			respBody:   `{"name":"memos/2","pinned":true}`,
		},
		{
			name:       "not found returns error",
			memoName:   "memos/999",
			content:    "test",
			visibility: "",
			fields:     []string{"content"},
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/api/v1/"+tt.memoName, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := NewMemos(srv.URL, "")
			resp, err := c.UpdateMemo(context.Background(), tt.memoName, tt.content, tt.visibility, tt.pinned, tt.fields)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestDeleteMemo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		memoName   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "deletes memo successfully with 200",
			memoName:   "memos/1",
			statusCode: http.StatusOK,
		},
		{
			name:       "deletes memo successfully with 204",
			memoName:   "memos/2",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "not found returns error",
			memoName:   "memos/999",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/api/v1/"+tt.memoName, r.URL.Path)
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			c := NewMemos(srv.URL, "")
			err := c.DeleteMemo(context.Background(), tt.memoName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		respBody   string
		wantErr    bool
		wantUser   string
	}{
		{
			name:       "gets current user",
			statusCode: http.StatusOK,
			respBody:   `{"user":{"name":"users/alice","username":"alice","role":"USER","displayName":"Alice"}}`,
			wantUser:   "alice",
		},
		{
			name:       "unauthorized returns error",
			statusCode: http.StatusUnauthorized,
			respBody:   `{}`,
			wantErr:    true,
		},
		{
			name:       "server error returns error",
			statusCode: http.StatusInternalServerError,
			respBody:   `{}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/auth/me", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := NewMemos(srv.URL, "")
			resp, err := c.GetCurrentUser(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantUser, resp.Username)
		})
	}
}

func TestListRawEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		cursor     string
		statusCode int
		respBody   string
		wantErr    bool
		wantCount  int
		wantCursor string
	}{
		{
			name:       "lists raw events with cursor",
			cursor:     "page-token-1",
			statusCode: http.StatusOK,
			respBody:   `{"memos":[{"name":"memos/1","content":"first","visibility":"PRIVATE"}],"nextPageToken":"page-token-2"}`,
			wantCount:  1,
			wantCursor: "page-token-2",
		},
		{
			name:       "empty cursor lists first page",
			cursor:     "",
			statusCode: http.StatusOK,
			respBody:   `{"memos":[],"nextPageToken":""}`,
			wantCount:  0,
		},
		{
			name:       "server error returns error",
			cursor:     "",
			statusCode: http.StatusInternalServerError,
			respBody:   `{}`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/memos", r.URL.Path)
				if tt.cursor != "" {
					assert.Equal(t, tt.cursor, r.URL.Query().Get("pageToken"))
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := NewMemos(srv.URL, "")
			items, cursor, err := c.ListRawEvents(context.Background(), tt.cursor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, items, tt.wantCount)
			assert.Equal(t, tt.wantCursor, cursor)
		})
	}
}

func TestListRawEvents_ContextCanceled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"memos":[]}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewMemos(srv.URL, "")
	_, _, err := c.ListRawEvents(ctx, "")
	assert.Error(t, err)
}
