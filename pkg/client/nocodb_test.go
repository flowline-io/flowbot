package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNocodbListBases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantLen    int
		wantErr    bool
		errContain string
	}{
		{
			name: "list success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/service/nocodb/bases", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[{"id":"b1","title":"Home"}],"page":{"limit":25,"has_more":false}}}`))
			},
			wantLen: 1,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"server error"}`))
			},
			wantErr:    true,
			errContain: "server error",
		},
		{
			name: "empty list",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[]}}`))
			},
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			got, err := c.Nocodb.ListBases(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Len(t, got.Items, tt.wantLen)
		})
	}
}

func TestNocodbRecordsCRUD(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		fn         func(*Client) error
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "create success",
			fn: func(c *Client) error {
				rec, err := c.Nocodb.CreateRecord(context.Background(), "t1", map[string]any{"Name": "a"})
				if err != nil {
					return err
				}
				assert.Equal(t, "9", rec.ID)
				return nil
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/service/nocodb/tables/t1/records", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"id":"9","fields":{"Name":"a"}}}}`))
			},
		},
		{
			name: "create missing fields",
			fn: func(c *Client) error {
				_, err := c.Nocodb.CreateRecord(context.Background(), "t1", nil)
				return err
			},
			wantErr:    true,
			errContain: "fields are required",
		},
		{
			name: "delete success",
			fn: func(c *Client) error {
				return c.Nocodb.DeleteRecord(context.Background(), "t1", "1")
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"deleted":"1"}}}`))
			},
		},
		{
			name: "list records with query",
			fn: func(c *Client) error {
				got, err := c.Nocodb.ListRecords(context.Background(), "t1", NocoListRecordsQuery{Limit: 5})
				if err != nil {
					return err
				}
				assert.Len(t, got.Items, 1)
				return nil
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "5", r.URL.Query().Get("limit"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[{"id":"1","fields":{"Name":"a"}}],"page":{"limit":5,"has_more":false}}}`))
			},
		},
		{
			name: "list negative limit",
			fn: func(c *Client) error {
				_, err := c.Nocodb.ListRecords(context.Background(), "t1", NocoListRecordsQuery{Limit: -1})
				return err
			},
			wantErr:    true,
			errContain: "limit must be non-negative",
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
			err := tt.fn(c)
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

func TestNocodbHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		wantOK  bool
		wantErr bool
	}{
		{name: "healthy", body: `{"status":"ok","data":{"data":true}}`, wantOK: true},
		{name: "unhealthy", body: `{"status":"ok","data":{"data":false}}`, wantOK: false},
		{name: "api error", body: `{"status":"failed","message":"down"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tt.wantErr {
					w.WriteHeader(http.StatusBadGateway)
				}
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			c := NewClient(server.URL, "token")
			ok, err := c.Nocodb.Health(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
