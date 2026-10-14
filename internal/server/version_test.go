package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/version"
)

func TestVersionHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "returns buildtags and buildstamp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			app.Get("/version", versionHandler)

			req := httptest.NewRequest(http.MethodGet, "/version", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var r protocol.Response
			require.NoError(t, sonic.Unmarshal(body, &r))
			assert.Equal(t, protocol.Success, r.Status)

			data, ok := r.Data.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, version.Buildtags, data["version"])
			assert.Equal(t, version.Buildstamp, data["buildstamp"])
		})
	}
}
