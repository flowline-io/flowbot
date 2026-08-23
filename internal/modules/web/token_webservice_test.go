package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeTokenParam(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid flag returns it",
			url:     "/test/fb_test123",
			want:    "fb_test123",
			wantErr: false,
		},
		{
			name:    "empty flag returns error",
			url:     "/test-empty",
			wantErr: true,
		},
		{
			name:    "token with special characters",
			url:     "/test/fb_token_with_underscores",
			want:    "fb_token_with_underscores",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var got string
			var gotErr error
			app.Get("/test/:flag", func(ctx fiber.Ctx) error {
				got, gotErr = decodeTokenParam(ctx)
				return nil
			})
			app.Get("/test-empty", func(ctx fiber.Ctx) error {
				got, gotErr = decodeTokenParam(ctx)
				return nil
			})
			req := httptest.NewRequest(http.MethodGet, tt.url, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			if tt.wantErr {
				assert.Error(t, gotErr)
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
