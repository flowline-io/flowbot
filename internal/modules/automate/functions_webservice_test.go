package automate

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

type callCatalog struct {
	defs map[string]*model.FunctionDefinition
	vers map[string]*model.FunctionDefinitionVersion
}

func newCallCatalog() *callCatalog {
	meta := "name: echo-fn\nhttp:\n  auth:\n    token: secret-token\n    hmac_secret: hmac-secret\nenv:\n  A: b\n"
	return &callCatalog{
		defs: map[string]*model.FunctionDefinition{
			"echo-fn": {ID: 1, Name: "echo-fn", Status: "published", Version: 2},
		},
		vers: map[string]*model.FunctionDefinitionVersion{
			"echo-fn": {
				FunctionName: "echo-fn",
				Version:      2,
				Metadata:     meta,
				Entrypoint:   "main.py",
				Source:       "print(1)",
			},
		},
	}
}

func (*callCatalog) Create(context.Context, string, string, string, string, string) error {
	return nil
}
func (c *callCatalog) GetByName(_ context.Context, name string) (*model.FunctionDefinition, error) {
	def, ok := c.defs[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *def
	return &cp, nil
}
func (*callCatalog) UpdateDraft(context.Context, string, string, string, string, int) (*model.FunctionDefinition, error) {
	return nil, types.ErrNotImplemented
}
func (*callCatalog) Publish(context.Context, string, int) (*model.FunctionDefinition, error) {
	return nil, types.ErrNotImplemented
}
func (*callCatalog) Delete(context.Context, string) (int64, error) { return 0, nil }
func (*callCatalog) ListPublished(context.Context) ([]*model.FunctionDefinition, error) {
	return nil, nil
}
func (*callCatalog) ListAll(context.Context) ([]*model.FunctionDefinition, error) {
	return nil, nil
}
func (c *callCatalog) GetVersion(_ context.Context, name string, version int) (*model.FunctionDefinitionVersion, error) {
	ver, ok := c.vers[name]
	if !ok || ver.Version != version {
		return nil, types.ErrNotFound
	}
	cp := *ver
	return &cp, nil
}
func (c *callCatalog) GetLatestPublished(_ context.Context, name string) (*model.FunctionDefinitionVersion, error) {
	ver, ok := c.vers[name]
	if !ok {
		return nil, types.ErrNotFound
	}
	cp := *ver
	return &cp, nil
}
func (*callCatalog) CreateRun(context.Context, string, int, *string) (*model.FunctionRun, error) {
	return &model.FunctionRun{ID: 1, FunctionName: "echo-fn", Version: 2, Status: "running"}, nil
}
func (*callCatalog) GetRunByIdempotency(context.Context, string, string) (*model.FunctionRun, error) {
	return nil, types.ErrNotFound
}
func (*callCatalog) CompleteRun(context.Context, int64, string, int64, *int, string, *string) (*model.FunctionRun, error) {
	return &model.FunctionRun{ID: 1, FunctionName: "echo-fn", Version: 2, Status: "succeeded"}, nil
}
func (*callCatalog) ListRuns(context.Context, string) ([]*model.FunctionRun, error) {
	return nil, nil
}

type failingExec struct{}

func (failingExec) ExecConfig(context.Context) (pkgexec.Config, error) {
	return pkgexec.Config{}, types.Errorf(types.ErrUnavailable, "exec unavailable in auth test")
}

func newFunctionsHandlerApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			switch {
			case errors.Is(err, types.ErrUnauthorized):
				code = fiber.StatusUnauthorized
			case errors.Is(err, types.ErrForbidden):
				code = fiber.StatusForbidden
			case errors.Is(err, types.ErrInvalidArgument):
				code = fiber.StatusBadRequest
			case errors.Is(err, types.ErrNotFound):
				code = fiber.StatusNotFound
			case errors.Is(err, types.ErrUnavailable):
				code = fiber.StatusServiceUnavailable
			case errors.Is(err, types.ErrConflict), errors.Is(err, types.ErrAlreadyExists):
				code = fiber.StatusConflict
			case errors.Is(err, types.ErrRateLimited):
				code = fiber.StatusTooManyRequests
			}
			return c.Status(code).SendString(err.Error())
		},
	})
}

func TestCallEndpointAuth(t *testing.T) {
	svc := pkgfunctions.NewService(newCallCatalog(), failingExec{})
	svc.SetChecker(dcg.AllowAllChecker{})
	prev := pkgfunctions.ActiveService()
	pkgfunctions.SetActiveService(svc)
	t.Cleanup(func() { pkgfunctions.SetActiveService(prev) })

	app := newFunctionsHandlerApp()
	app.Post("/service/automate/functions/call/:name", callFunction)
	app.Post("/service/automate/functions/call/:name/v/:version", callFunctionVersion)

	body := `{"hello":"world"}`
	mac := hmac.New(sha256.New, []byte("hmac-secret"))
	_, _ = mac.Write([]byte(body))
	hmacSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "valid token header",
			path:       "/service/automate/functions/call/echo-fn",
			headers:    map[string]string{"X-Webhook-Token": "secret-token"},
			wantStatus: fiber.StatusServiceUnavailable, // auth ok, exec unavailable
		},
		{
			name:       "valid query token",
			path:       "/service/automate/functions/call/echo-fn?token=secret-token",
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "valid hmac",
			path:       "/service/automate/functions/call/echo-fn",
			headers:    map[string]string{"X-Hub-Signature-256": hmacSig},
			wantStatus: fiber.StatusServiceUnavailable,
		},
		{
			name:       "missing auth",
			path:       "/service/automate/functions/call/echo-fn",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "wrong token",
			path:       "/service/automate/functions/call/echo-fn",
			headers:    map[string]string{"X-Webhook-Token": "wrong"},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "versioned path with token",
			path:       "/service/automate/functions/call/echo-fn/v/2",
			headers:    map[string]string{"X-Webhook-Token": "secret-token"},
			wantStatus: fiber.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			raw, _ := io.ReadAll(resp.Body)
			assert.Equal(t, tt.wantStatus, resp.StatusCode, "body=%s", raw)
		})
	}
}
