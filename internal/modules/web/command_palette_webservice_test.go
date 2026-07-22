package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestCommandPaletteSearch(t *testing.T) {
	tests := []struct {
		name           string
		cookie         bool
		query          string
		seedPipeline   string
		seedSession    bool
		seedHomelab    string
		enableAgent    bool
		wantStatus     int
		wantPageID     string
		wantPipelineID string
		wantSessionID  string
		wantHomelabID  string
		wantEmpty      bool
	}{
		{
			name:       "unauthenticated redirects to login",
			cookie:     false,
			query:      "home",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "empty query returns empty groups",
			cookie:     true,
			query:      "",
			wantStatus: http.StatusOK,
			wantEmpty:  true,
		},
		{
			name:       "matches nav page",
			cookie:     true,
			query:      "pipelines",
			wantStatus: http.StatusOK,
			wantPageID: "page:pipelines",
		},
		{
			name:           "matches pipeline definition",
			cookie:         true,
			query:          "palette-pipe",
			seedPipeline:   "palette-pipe",
			wantStatus:     http.StatusOK,
			wantPipelineID: "pipeline:palette-pipe",
		},
		{
			name:          "matches active session when chatagent enabled",
			cookie:        true,
			query:         "palette-sess",
			seedSession:   true,
			enableAgent:   true,
			wantStatus:    http.StatusOK,
			wantSessionID: "session:palette-sess",
		},
		{
			name:          "matches homelab app",
			cookie:        true,
			query:         "palette-app",
			seedHomelab:   "palette-app",
			wantStatus:    http.StatusOK,
			wantHomelabID: "homelab:palette-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var app *fiber.App
			var ts *testStore
			if tt.seedPipeline != "" {
				var client *store.Client
				app, ts, client = setupTestAppWithDB(t)
				require.NoError(t, store.NewPipelineStore(client).CreateDefinition(context.Background(), tt.seedPipeline, "", ""))
			} else {
				app, ts = setupTestApp()
			}
			t.Cleanup(func() {
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
			})

			oldApps := homelab.DefaultRegistry.List()
			if tt.seedHomelab != "" {
				homelab.DefaultRegistry.Replace([]homelab.App{{Name: tt.seedHomelab}})
			} else {
				homelab.DefaultRegistry.Replace(nil)
			}
			t.Cleanup(func() { homelab.DefaultRegistry.Replace(oldApps) })

			origModel := pkgconfig.App.ChatAgent.ChatModel
			if tt.enableAgent {
				pkgconfig.App.ChatAgent.ChatModel = "test-model"
			} else {
				pkgconfig.App.ChatAgent.ChatModel = ""
			}
			t.Cleanup(func() { pkgconfig.App.ChatAgent.ChatModel = origModel })

			if tt.seedSession {
				now := time.Now().UTC()
				ts.chatSessions = []*gen.ChatSession{{
					ID:        1,
					Flag:      "palette-sess",
					UID:       "testuser",
					Title:     "palette-sess debug",
					State:     int(schema.ChatSessionActive),
					CreatedAt: now,
					UpdatedAt: now,
				}}
			}

			path := "/service/web/command-palette/search"
			if tt.query != "" {
				path += "?q=" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			if tt.cookie {
				addWebAuth(req)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantStatus != http.StatusOK {
				return
			}
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			var got commandPaletteResults
			require.NoError(t, sonic.Unmarshal(body, &got))
			if tt.wantEmpty {
				assert.Empty(t, got.Pages)
				assert.Empty(t, got.Pipelines)
				assert.Empty(t, got.Sessions)
				assert.Empty(t, got.Homelab)
				return
			}
			if tt.wantPageID != "" {
				assert.Contains(t, itemIDs(got.Pages), tt.wantPageID)
			}
			if tt.wantPipelineID != "" {
				assert.Contains(t, itemIDs(got.Pipelines), tt.wantPipelineID)
			}
			if tt.wantSessionID != "" {
				assert.Contains(t, itemIDs(got.Sessions), tt.wantSessionID)
			}
			if tt.wantHomelabID != "" {
				assert.Contains(t, itemIDs(got.Homelab), tt.wantHomelabID)
			}
		})
	}
}
