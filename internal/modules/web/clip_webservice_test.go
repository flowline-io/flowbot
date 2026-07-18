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
)

func TestClipPage_AnonymousAndAuthed(t *testing.T) {
	app, _, dbClient := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	clipStore := store.NewClipStore(dbClient)
	slug := "KhpG3Hab"
	secret := "SECRET_MARKDOWN_BODY_TOKEN"
	err := clipStore.CreateClip(context.Background(), slug, "Clip Title", "Clip description for Slack",
		"# Heading\n\n"+secret+"\n", "tester")
	require.NoError(t, err)

	tests := []struct {
		name         string
		slug         string
		withCookie   bool
		wantStatus   int
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:       "anonymous sees meta not body",
			slug:       slug,
			withCookie: false,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Clip Title",
				"Clip description for Slack",
				`meta name="description"`,
				`property="og:title"`,
				`property="og:description"`,
				"Log in to read",
				"slug: KhpG3Hab",
				`data-testid="clip-brand"`,
			},
			wantAbsent: []string{secret, `href="/"`},
		},
		{
			name:       "authenticated sees markdown body",
			slug:       slug,
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Clip Title",
				secret,
				"Copy MD",
				"<h1",
			},
			wantAbsent: []string{"Log in to read"},
		},
		{
			name:       "missing clip returns 404 shell",
			slug:       "missing1",
			withCookie: false,
			wantStatus: http.StatusNotFound,
			wantContains: []string{
				"Clip not found",
				`meta name="description"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/c/"+tt.slug, http.NoBody)
			if tt.withCookie {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			html := string(body)
			for _, sub := range tt.wantContains {
				assert.Contains(t, html, sub)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, html, absent)
			}
		})
	}
}
