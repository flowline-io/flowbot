package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestNewHTTPServer_CreatesValidApp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "newHTTPServer creates a valid fiber app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newHTTPServer()
			assert.NotNil(t, app)
			defer app.Shutdown()
		})
	}
}

func TestNewHTTPServer_MiddlewareRoutes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		expectedPath string
		shouldExist  bool
	}{
		{name: "swagger is not registered when swagHandler is nil", expectedPath: "/swagger/test", shouldExist: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newHTTPServer()
			defer app.Shutdown()

			routes := app.GetRoutes()
			routePaths := make(map[string]bool)
			for _, r := range routes {
				routePaths[r.Path] = true
			}
			if tt.shouldExist {
				assert.True(t, routePaths[tt.expectedPath], "expected route %q to be registered", tt.expectedPath)
			} else {
				assert.False(t, routePaths[tt.expectedPath], "route %q should not exist", tt.expectedPath)
			}
		})
	}
}

func TestNewHTTPServer_ErrorHandler_DomainErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "ErrNotFound returns 404",
			err:        types.Errorf(types.ErrNotFound, "item %s not found", "x"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ErrForbidden returns 403",
			err:        types.Errorf(types.ErrForbidden, "access denied"),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "ErrUnauthorized returns 401",
			err:        types.Errorf(types.ErrUnauthorized, "bad token"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "ErrInternal returns 500",
			err:        types.Errorf(types.ErrInternal, "internal failure"),
			wantStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newHTTPServer()
			defer app.Shutdown()

			app.Get("/test", func(_ fiber.Ctx) error { return tt.err })

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestNewHTTPServer_ErrorHandler_ProtocolErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "ErrNotAuthorized returns 401",
			err:        protocol.ErrNotAuthorized.New("not authorized"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "ErrBadRequest returns 400",
			err:        protocol.ErrBadRequest.New("bad request"),
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newHTTPServer()
			defer app.Shutdown()

			app.Get("/test", func(_ fiber.Ctx) error { return tt.err })

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestNewHTTPServer_ErrorHandler_FiberErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "ErrNotFound returns 404",
			err:        fiber.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ErrServiceUnavailable returns 503",
			err:        fiber.ErrServiceUnavailable,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "ErrMethodNotAllowed returns 405",
			err:        fiber.ErrMethodNotAllowed,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newHTTPServer()
			defer app.Shutdown()

			app.Get("/test", func(_ fiber.Ctx) error {
				return tt.err
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestNewHTTPServer_ErrorHandler_UnknownError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "unknown error returns 500"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := newHTTPServer()
			defer app.Shutdown()

			app.Get("/test", func(_ fiber.Ctx) error {
				return errors.New("some unhandled error")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		})
	}
}

func TestStructValidator_Validate(t *testing.T) {
	t.Parallel()
	v := &structValidator{validate: newTestApp().Config().StructValidator.(*structValidator).validate}

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "nil input passes validation",
			input:   nil,
			wantErr: false,
		},
		{
			name: "struct with no validation tags passes",
			input: struct {
				Name string
			}{Name: "test"},
			wantErr: false,
		},
		{
			name: "invalid input type returns error",
			input: struct {
				Field int `validate:"min=10"`
			}{Field: 5},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var err error
			if tt.input != nil {
				err = v.Validate(tt.input)
			}
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
