package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

func TestClipsListPage(t *testing.T) {
	app, _, dbClient := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	clipStore := store.NewClipStore(dbClient)
	require.NoError(t, clipStore.CreateClip(context.Background(), "listSlug1", "List Title", "List desc", "# body", "tester"))

	tests := []struct {
		name         string
		path         string
		withCookie   bool
		wantStatus   int
		wantContains []string
	}{
		{
			name:       "unauthenticated redirects",
			path:       "/service/web/clips",
			withCookie: false,
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "authenticated page lists clips",
			path:       "/service/web/clips",
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Clips — Flowbot",
				"List Title",
				"listSlug1",
				"/c/listSlug1",
				`data-testid="nav-clips"`,
			},
		},
		{
			name:       "list partial returns table",
			path:       "/service/web/clips/list",
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`data-testid="clips-table"`,
				"List Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			if tt.withCookie {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
				AttachCSRFForTest(req)
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
			html := string(body)
			for _, sub := range tt.wantContains {
				assert.Contains(t, html, sub)
			}
		})
	}
}

func TestClipRowsToListItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rows []*gen.Clip
		want int
	}{
		{name: "nil rows", rows: nil, want: 0},
		{name: "skips nil entry", rows: []*gen.Clip{nil, {Slug: "a", Title: "T"}}, want: 1},
		{name: "maps url", rows: []*gen.Clip{{Slug: "abc", Title: "X"}}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			items := clipRowsToListItems(tt.rows)
			assert.Len(t, items, tt.want)
			if tt.want == 1 {
				assert.Equal(t, "/c/"+tt.rows[len(tt.rows)-1].Slug, items[0].URL)
			}
		})
	}
}
