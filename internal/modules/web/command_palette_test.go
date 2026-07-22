package web

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestBuildCommandPaletteResults(t *testing.T) {
	t.Parallel()
	pages := []commandPaletteItem{
		{ID: "page:home", Title: "Home", Subtitle: "Dashboard", Href: "/service/web/home", Group: "pages"},
		{ID: "page:pipelines", Title: "Pipelines", Subtitle: "Automate", Href: "/service/web/pipelines", Group: "pages"},
		{ID: "page:agents", Title: "Agents", Subtitle: "Agent", Href: "/service/web/agents", Group: "pages"},
	}
	pipelines := []*gen.PipelineDefinition{
		{Name: "sync-notes", Description: "Sync memos"},
		{Name: "daily-backup", Description: "Backup volumes"},
		{Name: "alert-pipeline", Description: ""},
	}
	sessions := []chatagent.SessionSummary{
		{SessionID: "sess-alpha", Title: "Debug pipeline"},
		{SessionID: "sess-beta", Title: "Homelab inventory"},
		{SessionID: "sess-gamma", Title: ""},
	}
	apps := []homelab.App{
		{Name: "jellyfin"},
		{Name: "vaultwarden"},
		{Name: "paperless"},
	}

	tests := []struct {
		name           string
		q              string
		wantPages      []string
		wantPipelines  []string
		wantSessions   []string
		wantHomelab    []string
		wantEmptyAll   bool
		maxPerGroup    int
		extraPipelines int
	}{
		{
			name:         "empty query returns empty groups",
			q:            "",
			wantEmptyAll: true,
		},
		{
			name:          "case-insensitive page and pipeline match",
			q:             "PIPE",
			wantPages:     []string{"page:pipelines"},
			wantPipelines: []string{"pipeline:alert-pipeline"},
			wantSessions:  []string{"session:sess-alpha"},
		},
		{
			name:         "matches sessions by title",
			q:            "inventory",
			wantSessions: []string{"session:sess-beta"},
		},
		{
			name:         "matches home page title and homelab session substring",
			q:            "home",
			wantPages:    []string{"page:home"},
			wantSessions: []string{"session:sess-beta"},
		},
		{
			name:        "matches homelab app name",
			q:           "jelly",
			wantHomelab: []string{"homelab:jellyfin"},
		},
		{
			name:           "caps each group at eight",
			q:              "p",
			maxPerGroup:    8,
			extraPipelines: 12,
		},
		{
			name:         "whitespace-only query is empty",
			q:            "   ",
			wantEmptyAll: true,
		},
		{
			name:         "matches untitled session by id prefix",
			q:            "sess-gamma",
			wantSessions: []string{"session:sess-gamma"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pls := pipelines
			if tt.extraPipelines > 0 {
				pls = make([]*gen.PipelineDefinition, 0, tt.extraPipelines)
				for i := 0; i < tt.extraPipelines; i++ {
					pls = append(pls, &gen.PipelineDefinition{Name: "p-item-" + strconv.Itoa(i)})
				}
			}
			got := buildCommandPaletteResults(tt.q, pages, pls, sessions, apps)
			if tt.wantEmptyAll {
				assert.Empty(t, got.Pages)
				assert.Empty(t, got.Pipelines)
				assert.Empty(t, got.Sessions)
				assert.Empty(t, got.Homelab)
				return
			}
			if tt.extraPipelines > 0 {
				assert.LessOrEqual(t, len(got.Pipelines), 8)
				assert.NotEmpty(t, got.Pipelines)
				return
			}
			assert.Equal(t, tt.wantPages, itemIDs(got.Pages))
			assert.Equal(t, tt.wantPipelines, itemIDs(got.Pipelines))
			assert.Equal(t, tt.wantSessions, itemIDs(got.Sessions))
			assert.Equal(t, tt.wantHomelab, itemIDs(got.Homelab))
		})
	}
}

func TestCommandPaletteNavPages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		wantHref  string
		wantTitle string
	}{
		{name: "includes home", wantHref: "/service/web/home", wantTitle: "Home"},
		{name: "includes pipelines", wantHref: "/service/web/pipelines", wantTitle: "Pipelines"},
		{name: "includes agents", wantHref: "/service/web/agents", wantTitle: "Agents"},
		{name: "includes homelab registry", wantHref: "/service/web/homelab", wantTitle: "Registry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pages := commandPaletteNavPages()
			require.NotEmpty(t, pages)
			found := false
			for _, p := range pages {
				if p.Href == tt.wantHref && p.Title == tt.wantTitle {
					found = true
					assert.Equal(t, "pages", p.Group)
					break
				}
			}
			assert.True(t, found, "missing page %s (%s)", tt.wantTitle, tt.wantHref)
		})
	}
}

func itemIDs(items []commandPaletteItem) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}
