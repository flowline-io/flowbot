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
	privateSlug := "KhpG3Hab"
	publicSlug := "PubClip01"
	secretPrivate := "SECRET_PRIVATE_BODY"
	secretPublic := "SECRET_PUBLIC_BODY"
	err := clipStore.CreateClip(context.Background(), privateSlug, "Private Clip", "Private description",
		"# Private\n\n"+secretPrivate+"\n", "tester")
	require.NoError(t, err)
	err = clipStore.CreateClip(context.Background(), publicSlug, "Public Clip", "Public description",
		"# Public\n\n"+secretPublic+"\n", "tester")
	require.NoError(t, err)
	_, err = clipStore.UpdateClipVisibility(context.Background(), publicSlug, true)
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
			name:       "anonymous private clip returns 404",
			slug:       privateSlug,
			withCookie: false,
			wantStatus: http.StatusNotFound,
			wantContains: []string{
				"Clip not found",
				`meta name="description"`,
			},
			wantAbsent: []string{secretPrivate, "Private Clip", "Log in to read"},
		},
		{
			name:       "anonymous public clip sees body and copy",
			slug:       publicSlug,
			withCookie: false,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Public Clip",
				secretPublic,
				"Copy MD",
				`<h1`,
				`data-testid="clip-brand"`,
			},
			wantAbsent: []string{"Log in to read"},
		},
		{
			name:       "authenticated private clip sees body",
			slug:       privateSlug,
			withCookie: true,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Private Clip",
				secretPrivate,
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
