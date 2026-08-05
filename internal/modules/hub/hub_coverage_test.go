package hub

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestParseQueryInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want int64
	}{
		{name: "empty", in: "", want: 0},
		{name: "digits", in: "42", want: 42},
		{name: "rejects non digits", in: "12a", want: 0},
		{name: "rejects negative sign", in: "-1", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseQueryInt(tt.in))
		})
	}
}

func TestFormatAppPorts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		ports []homelab.PortMapping
		want  string
	}{
		{name: "empty", ports: nil, want: "-"},
		{name: "host_port preferred", ports: []homelab.PortMapping{{HostPort: "8080", Host: "9"}}, want: "8080"},
		{name: "falls back to host", ports: []homelab.PortMapping{{Host: "3000"}}, want: "3000"},
		{name: "skips blank entries", ports: []homelab.PortMapping{{}, {HostPort: "80"}, {Host: "443"}}, want: "80, 443"},
		{name: "all blank", ports: []homelab.PortMapping{{}}, want: "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatAppPorts(tt.ports))
		})
	}
}

func TestQueryByTag_EmptySuccess(t *testing.T) {
	old := rcStore
	rcStore = store.NewResourceChainStore(nil)
	t.Cleanup(func() { rcStore = old })

	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/resource-chain", queryByTag)

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantSubstr string
	}{
		{name: "returns empty resources", query: "key=project&value=alpha", wantStatus: 200, wantSubstr: `"resources":[]`},
		{name: "accepts limit and cursor", query: "key=project&value=alpha&limit=5&cursor=x", wantStatus: 200, wantSubstr: `"tag"`},
		{name: "still validates missing key", query: "value=alpha", wantStatus: 400, wantSubstr: "key and value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(fiber.MethodGet, "/resource-chain?"+tt.query, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode, string(body))
			assert.Contains(t, string(body), tt.wantSubstr)
		})
	}
}

func TestCreateFireflyTransaction(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		register   bool
		wantStatus int
		wantSubstr string
	}{
		{name: "type required", body: `{"date":"2026-01-01","amount":"1","description":"x","source_id":"1","destination_id":"2"}`, wantStatus: 400, wantSubstr: "type is required"},
		{name: "source required", body: `{"type":"withdrawal","date":"2026-01-01","amount":"1","description":"x","destination_id":"2"}`, wantStatus: 400, wantSubstr: "source_id or source_name"},
		{name: "destination required", body: `{"type":"withdrawal","date":"2026-01-01","amount":"1","description":"x","source_id":"1"}`, wantStatus: 400, wantSubstr: "destination_id or destination_name"},
		{
			name:       "invokes capability",
			body:       `{"type":"withdrawal","date":"2026-01-01","amount":"1.5","description":"coffee","source_name":"wallet","destination_name":"shop"}`,
			register:   true,
			wantStatus: 200,
			wantSubstr: `"status":"ok"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.register {
				registerTestInvoker(t, hub.CapFireflyiii, capability.OpFinanceCreateTransaction, func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
					assert.Equal(t, "withdrawal", params["type"])
					assert.Equal(t, "coffee", params["description"])
					return &capability.InvokeResult{Data: map[string]any{"id": "tx-1"}}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Post("/transactions", createFireflyTransaction)
			req := httptest.NewRequest(fiber.MethodPost, "/transactions", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode, string(body))
			assert.Contains(t, string(body), tt.wantSubstr)
		})
	}
}

func TestUpdateEntriesStatus(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantOps    []string
		wantSubstr string
	}{
		{
			name:       "validation requires entry ids",
			body:       `{"entry_ids":[],"status":"read"}`,
			wantStatus: 400,
			wantSubstr: "",
		},
		{
			name:       "read maps to mark read",
			body:       `{"entry_ids":[1,2],"status":"read"}`,
			wantStatus: 200,
			wantOps:    []string{capability.OpReaderMarkEntryRead, capability.OpReaderMarkEntryRead},
			wantSubstr: `"success":true`,
		},
		{
			name:       "removed falls through to unread",
			body:       `{"entry_ids":[9],"status":"removed"}`,
			wantStatus: 200,
			wantOps:    []string{capability.OpReaderMarkEntryUnread},
			wantSubstr: `"success":true`,
		},
		{
			name:       "unread maps to mark unread",
			body:       `{"entry_ids":[3],"status":"unread"}`,
			wantStatus: 200,
			wantOps:    []string{capability.OpReaderMarkEntryUnread},
			wantSubstr: `"success":true`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotOps []string
			if tt.wantStatus == 200 {
				registerTestInvoker(t, hub.CapMiniflux, capability.OpReaderMarkEntryRead, func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
					gotOps = append(gotOps, capability.OpReaderMarkEntryRead)
					assert.NotNil(t, params["id"])
					return &capability.InvokeResult{}, nil
				})
				registerTestInvoker(t, hub.CapMiniflux, capability.OpReaderMarkEntryUnread, func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
					gotOps = append(gotOps, capability.OpReaderMarkEntryUnread)
					assert.NotNil(t, params["id"])
					return &capability.InvokeResult{}, nil
				})
			}
			app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
			app.Patch("/entries", updateEntriesStatus)
			req := httptest.NewRequest(fiber.MethodPatch, "/entries", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode, string(body))
			if tt.wantSubstr != "" {
				assert.Contains(t, string(body), tt.wantSubstr)
			}
			if tt.wantOps != nil {
				assert.Equal(t, tt.wantOps, gotOps)
			}
		})
	}
}
