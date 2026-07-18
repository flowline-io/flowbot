package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFireflyiiiCreateTransaction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		req        *CreateTransactionRequest
		handler    http.HandlerFunc
		wantID     string
		wantErr    bool
		errContain string
	}{
		{
			name: "create success",
			req: &CreateTransactionRequest{
				Type: "withdrawal", Date: "2026-07-18", Amount: "10.00", Description: "Food",
				SourceName: "Cash", DestinationName: "Store",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/service/fireflyiii/transactions", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"id":"42","type":"withdrawal","amount":"10.00","description":"Food"}}}`))
			},
			wantID: "42",
		},
		{
			name:       "nil request",
			req:        nil,
			wantErr:    true,
			errContain: "request is required",
		},
		{
			name:       "missing type",
			req:        &CreateTransactionRequest{Date: "2026-07-18", Amount: "1", Description: "x", SourceName: "Cash", DestinationName: "Store"},
			wantErr:    true,
			errContain: "type is required",
		},
		{
			name:       "missing source",
			req:        &CreateTransactionRequest{Type: "withdrawal", Date: "2026-07-18", Amount: "1", Description: "x", DestinationName: "Store"},
			wantErr:    true,
			errContain: "source_id or source_name is required",
		},
		{
			name: "api error",
			req: &CreateTransactionRequest{
				Type: "withdrawal", Date: "2026-07-18", Amount: "1", Description: "x",
				SourceName: "Cash", DestinationName: "Store",
			},
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
			got, err := c.Fireflyiii.CreateTransaction(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.ID)
		})
	}
}

func TestFireflyiiiAboutUserHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "about success",
			run: func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/service/fireflyiii/about", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"version":"6.0.0","os":"Linux"}}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				info, err := c.Fireflyiii.About(context.Background())
				require.NoError(t, err)
				assert.Equal(t, "6.0.0", info.Version)
			},
		},
		{
			name: "current user success",
			run: func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/service/fireflyiii/user", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"id":"1","email":"a@b.c","role":"owner"}}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				user, err := c.Fireflyiii.CurrentUser(context.Background())
				require.NoError(t, err)
				assert.Equal(t, "1", user.ID)
				assert.Equal(t, "a@b.c", user.Email)
			},
		},
		{
			name: "health healthy",
			run: func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/service/fireflyiii/health", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":true}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				ok, err := c.Fireflyiii.Health(context.Background())
				require.NoError(t, err)
				assert.True(t, ok)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
