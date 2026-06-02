package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
)

func TestRelationsPage(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
		wantText   string
	}{
		{
			name:       "returns 200 with title",
			wantStatus: http.StatusOK,
			wantText:   "Relations",
		},
		{
			name:       "contains search input placeholder",
			wantStatus: http.StatusOK,
			wantText:   "Search by entity ID",
		},
		{
			name:       "contains empty state for detail panel",
			wantStatus: http.StatusOK,
			wantText:   "Select a node or edge to see details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			req := httptest.NewRequest(http.MethodGet, "/service/web/relations", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.Contains(t, string(body), tt.wantText)
		})
	}
}

func TestRelationsTree(t *testing.T) {
	tests := []struct {
		name       string
		nodeParam  string
		seedFn     func(ctx context.Context, client *store.Client) error
		wantStatus int
		wantText   string
	}{
		{
			name:       "missing node param shows empty state",
			nodeParam:  "",
			wantStatus: http.StatusOK,
			wantText:   "Search for a resource entity ID",
		},
		{
			name:       "invalid node format shows error",
			nodeParam:  "invalid",
			wantStatus: http.StatusBadRequest,
			wantText:   "Invalid node format",
		},
		{
			name:      "valid node returns tree with pipeline name",
			nodeParam: "github|issue|42",
			seedFn: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-1").
					SetTargetEventID("tgt-1").
					SetSourceApp("github").
					SetSourceCapability("issue").
					SetSourceEntityID("42").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("99").
					SetPipelineName("sync-issues").
					Save(ctx)
				return err
			},
			wantStatus: http.StatusOK,
			wantText:   "sync-issues",
		},
		{
			name:      "node with no relations shows none",
			nodeParam: "github|issue|999",
			seedFn: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-2").
					SetTargetEventID("tgt-2").
					SetSourceApp("forge").
					SetSourceCapability("issue").
					SetSourceEntityID("99").
					SetTargetApp("kanban").
					SetTargetCapability("task").
					SetTargetEntityID("10").
					SetPipelineName("other").
					Save(ctx)
				return err
			},
			wantStatus: http.StatusOK,
			wantText:   "None",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var app *fiber.App
			if tt.seedFn != nil {
				app, _, _ = setupTestAppForRelations(t, tt.seedFn)
			} else {
				app, _ = setupTestApp()
			}
			url := "/service/web/relations/tree"
			if tt.nodeParam != "" {
				url += "?node=" + tt.nodeParam
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.Contains(t, string(body), tt.wantText)
		})
	}
}

func TestRelationsSearch(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		seedFn     func(ctx context.Context, client *store.Client) error
		wantStatus int
		wantText   string
	}{
		{
			name:       "empty query returns empty body",
			query:      "",
			wantStatus: http.StatusOK,
			wantText:   "",
		},
		{
			name:  "matching query returns results",
			query: "42",
			seedFn: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-a").
					SetTargetEventID("tgt-a").
					SetSourceApp("github").
					SetSourceCapability("issue").
					SetSourceEntityID("42").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("99").
					SetPipelineName("sync").
					Save(ctx)
				return err
			},
			wantStatus: http.StatusOK,
			wantText:   "42",
		},
		{
			name:  "no match returns empty state",
			query: "nonexistent",
			seedFn: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-b").
					SetTargetEventID("tgt-b").
					SetSourceApp("github").
					SetSourceCapability("issue").
					SetSourceEntityID("42").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("99").
					SetPipelineName("sync").
					Save(ctx)
				return err
			},
			wantStatus: http.StatusOK,
			wantText:   "No resources found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var app *fiber.App
			if tt.seedFn != nil {
				app, _, _ = setupTestAppForRelations(t, tt.seedFn)
			} else {
				app, _ = setupTestApp()
			}
			url := "/service/web/relations/search?q=" + tt.query
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if tt.wantText != "" {
				assert.Contains(t, string(body), tt.wantText)
			}
		})
	}
}

func TestRelationsDetail(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantText   string
	}{
		{
			name:       "node detail returns metadata",
			query:      "type=node&app=github&capability=issue&entity_id=42",
			wantStatus: http.StatusOK,
			wantText:   "Resource Node",
		},
		{
			name:       "edge detail returns metadata",
			query:      "type=edge&source_app=github&source_entity=42&target_app=forge&target_entity=99&pipeline=sync",
			wantStatus: http.StatusOK,
			wantText:   "Relation Edge",
		},
		{
			name:       "unknown type returns error",
			query:      "type=unknown",
			wantStatus: http.StatusOK,
			wantText:   "Invalid detail type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			req := httptest.NewRequest(http.MethodGet, "/service/web/relations/detail?"+tt.query, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.Contains(t, string(body), tt.wantText)
		})
	}
}
