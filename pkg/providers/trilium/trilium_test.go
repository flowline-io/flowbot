package trilium

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

// unreachableAddr is a loopback address on a different IP than httptest.NewServer's 127.0.0.1.
// Using a separate IP prevents the kernel from reassigning a closed server's port to another
// parallel test server, which would cause the subsequent request to land on the wrong handler.
const unreachableAddr = "http://127.0.0.2:1"

// testServerURL returns an httptest server URL or unreachableAddr for connection-error cases.
func testServerURL(t *testing.T, handler http.HandlerFunc, connErr bool) string {
	t.Helper()
	if connErr {
		return unreachableAddr
	}
	if handler == nil {
		handler = func(_ http.ResponseWriter, _ *http.Request) {}
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestGetClient_Defaults(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantNil bool
	}{
		{
			name:    "empty config returns nil",
			configs: json.RawMessage(`{}`),
			wantNil: true,
		},
		{
			name:    "custom endpoint creates client",
			configs: json.RawMessage(`{"trilium":{"endpoint":"https://trilium.example.com","token":"abc123"}}`),
			wantNil: false,
		},
		{
			name:    "endpoint without token",
			configs: json.RawMessage(`{"trilium":{"endpoint":"https://trilium.example.com"}}`),
			wantNil: false,
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
		})
	}
}

func TestNewTrilium(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		token    string
		wantNil  bool
	}{
		{
			name:     "empty endpoint returns nil",
			endpoint: "",
			token:    "",
			wantNil:  true,
		},
		{
			name:     "endpoint with token",
			endpoint: "https://trilium.example.com",
			token:    "test-token",
			wantNil:  false,
		},
		{
			name:     "endpoint without token",
			endpoint: "https://trilium.example.com",
			token:    "",
			wantNil:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(tt.endpoint, tt.token)
			if tt.wantNil {
				assert.Nil(t, c)
				return
			}
			require.NotNil(t, c)
		})
	}
}

func TestGetAppInfo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name: "successful app info",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/etapi/app-info", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"appVersion":"0.63.7","dbVersion":1,"syncVersion":1,"buildDate":"2024-01-01","buildRevision":"abc","dataDirectory":"/data","instanceName":"test"}`))
			},
		},
		{
			name:    "connection error",
			wantErr: true,
			connErr: true,
		},
		{
			name: "server error response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":500,"code":"INTERNAL_ERROR","message":"something went wrong"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			resp, err := c.GetAppInfo(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, "0.63.7", resp.AppVersion)
		})
	}
}

func TestCreateNote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		req     CreateNoteDef
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name: "successful create note",
			req: CreateNoteDef{
				ParentNoteID: "root",
				Title:        "Test Note",
				Type:         "text",
				Content:      "<p>Hello</p>",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/etapi/create-note", r.URL.Path)
				assert.Equal(t, "test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"note":{"noteId":"note123","title":"Test Note","type":"text"},"branch":{"branchId":"branch456","noteId":"note123","parentNoteId":"root"}}`))
			},
		},
		{
			name:    "connection error",
			req:     CreateNoteDef{ParentNoteID: "root", Title: "Test", Type: "text", Content: "content"},
			wantErr: true,
			connErr: true,
		},
		{
			name: "server error on create",
			req: CreateNoteDef{
				ParentNoteID: "root",
				Title:        "Fail",
				Type:         "text",
				Content:      "content",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":400,"code":"BAD_REQUEST","message":"invalid parent note"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			resp, err := c.CreateNote(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, "note123", resp.Note.NoteID)
			assert.Equal(t, "branch456", resp.Branch.BranchID)
		})
	}
}

func TestGetNote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		noteID  string
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name:   "successful get note",
			noteID: "note123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/etapi/notes/note123", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"noteId":"note123","title":"My Note","type":"text"}`))
			},
		},
		{
			name:    "connection error",
			noteID:  "note123",
			wantErr: true,
			connErr: true,
		},
		{
			name:   "not found",
			noteID: "nonexistent",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":404,"code":"NOT_FOUND","message":"note not found"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			resp, err := c.GetNote(context.Background(), tt.noteID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "note123", resp.NoteID)
			assert.Equal(t, "My Note", resp.Title)
		})
	}
}

func TestPatchNote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		noteID  string
		req     PatchNoteRequest
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name:   "successful patch title",
			noteID: "note123",
			req:    PatchNoteRequest{Title: "Updated Title"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PATCH", r.Method)
				assert.Equal(t, "/etapi/notes/note123", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"noteId":"note123","title":"Updated Title","type":"text"}`))
			},
		},
		{
			name:    "connection error",
			noteID:  "note123",
			req:     PatchNoteRequest{Title: "Test"},
			wantErr: true,
			connErr: true,
		},
		{
			name:   "patch with bad request",
			noteID: "note123",
			req:    PatchNoteRequest{Type: "invalid"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"status":400,"code":"BAD_REQUEST","message":"invalid type"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			resp, err := c.PatchNote(context.Background(), tt.noteID, tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "Updated Title", resp.Title)
		})
	}
}

func TestDeleteNote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		noteID  string
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name:   "successful delete",
			noteID: "note123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/etapi/notes/note123", r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			name:    "connection error",
			noteID:  "note123",
			wantErr: true,
			connErr: true,
		},
		{
			name:   "not found on delete",
			noteID: "nonexistent",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":404,"code":"NOT_FOUND","message":"note not found"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			err := c.DeleteNote(context.Background(), tt.noteID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestSearchNotes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		params  SearchParams
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name:   "successful search",
			params: SearchParams{Search: "test query", Limit: 10},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/etapi/notes", r.URL.Path)
				assert.Equal(t, "test query", r.URL.Query().Get("search"))
				assert.Equal(t, "10", r.URL.Query().Get("limit"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"results":[{"noteId":"n1","title":"Result 1","type":"text"},{"noteId":"n2","title":"Result 2","type":"text"}]}`))
			},
		},
		{
			name:    "connection error",
			params:  SearchParams{Search: "*"},
			wantErr: true,
			connErr: true,
		},
		{
			name: "search with all params",
			params: SearchParams{
				Search:               "query",
				FastSearch:           true,
				IncludeArchivedNotes: true,
				AncestorNoteID:       "parent",
				AncestorDepth:        "eq1",
				OrderBy:              "title",
				OrderDirection:       "asc",
				Limit:                5,
				Debug:                true,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "query", q.Get("search"))
				assert.Equal(t, "true", q.Get("fastSearch"))
				assert.Equal(t, "true", q.Get("includeArchivedNotes"))
				assert.Equal(t, "parent", q.Get("ancestorNoteId"))
				assert.Equal(t, "eq1", q.Get("ancestorDepth"))
				assert.Equal(t, "title", q.Get("orderBy"))
				assert.Equal(t, "asc", q.Get("orderDirection"))
				assert.Equal(t, "5", q.Get("limit"))
				assert.Equal(t, "true", q.Get("debug"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"results":[]}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			resp, err := c.SearchNotes(context.Background(), tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestGetNoteContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		noteID  string
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name:   "successful get content",
			noteID: "note123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/etapi/notes/note123/content", r.URL.Path)
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte("<p>Hello World</p>"))
			},
		},
		{
			name:    "connection error",
			noteID:  "note123",
			wantErr: true,
			connErr: true,
		},
		{
			name:   "not found",
			noteID: "nonexistent",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":404,"code":"NOT_FOUND","message":"note not found"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			content, err := c.GetNoteContent(context.Background(), tt.noteID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "<p>Hello World</p>", content)
		})
	}
}

func TestUpdateNoteContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		noteID  string
		content string
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name:    "successful update content",
			noteID:  "note123",
			content: "<p>Updated</p>",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "/etapi/notes/note123/content", r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			name:    "connection error",
			noteID:  "note123",
			content: "test",
			wantErr: true,
			connErr: true,
		},
		{
			name:    "server error",
			noteID:  "note123",
			content: "bad",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":500,"code":"INTERNAL_ERROR","message":"fail"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			err := c.UpdateNoteContent(context.Background(), tt.noteID, tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestLogin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		password string
		handler  http.HandlerFunc
		wantErr  bool
		connErr  bool
	}{
		{
			name:     "successful login",
			password: "correct-password",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/etapi/auth/login", r.URL.Path)
				// Verify no auth header on login
				assert.Empty(t, r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"authToken":"etapi-token-12345"}`))
			},
		},
		{
			name:     "connection error",
			password: "test",
			wantErr:  true,
			connErr:  true,
		},
		{
			name:     "wrong password",
			password: "wrong",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"status":401,"code":"UNAUTHORIZED","message":"invalid password"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			token, err := c.Login(context.Background(), tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "etapi-token-12345", token)
		})
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name: "successful logout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/etapi/auth/logout", r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			name:    "connection error",
			wantErr: true,
			connErr: true,
		},
		{
			name: "server error on logout",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":500,"code":"INTERNAL_ERROR","message":"fail"}`))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			err := c.Logout(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCreateBranch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		req     BranchRequest
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name: "successful create branch",
			req: BranchRequest{
				NoteID:       "note1",
				ParentNoteID: "parent1",
				Prefix:       "copy",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/etapi/branches", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"branchId":"b1","noteId":"note1","parentNoteId":"parent1","prefix":"copy"}`))
			},
		},
		{
			name:    "connection error",
			req:     BranchRequest{NoteID: "n1", ParentNoteID: "p1"},
			wantErr: true,
			connErr: true,
		},
		{
			name: "branch already exists returns 200",
			req:  BranchRequest{NoteID: "n1", ParentNoteID: "p1"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"branchId":"existing","noteId":"n1","parentNoteId":"p1"}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			resp, err := c.CreateBranch(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestCreateAttribute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		req     CreateAttribute
		handler http.HandlerFunc
		wantErr bool
		connErr bool
	}{
		{
			name: "successful create label",
			req: CreateAttribute{
				NoteID: "note1",
				Type:   "label",
				Name:   "myLabel",
				Value:  "myValue",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/etapi/attributes", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"attributeId":"attr1","noteId":"note1","type":"label","name":"myLabel","value":"myValue"}`))
			},
		},
		{
			name:    "connection error",
			req:     CreateAttribute{NoteID: "n1", Type: "label", Name: "test"},
			wantErr: true,
			connErr: true,
		},
		{
			name: "create relation",
			req: CreateAttribute{
				NoteID: "note1",
				Type:   "relation",
				Name:   "relatedTo",
				Value:  "note2",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"attributeId":"attr2","noteId":"note1","type":"relation","name":"relatedTo","value":"note2"}`))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewTrilium(testServerURL(t, tt.handler, tt.connErr), "test-token")
			resp, err := c.CreateAttribute(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}
