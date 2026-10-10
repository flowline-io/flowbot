package mcp

import (
	"context"
	"errors"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/version"
)

type identKey struct{}

func withIdentity(ctx context.Context, ident Identity) context.Context {
	return context.WithValue(ctx, identKey{}, ident)
}

func identityFrom(ctx context.Context) Identity {
	ident, ok := ctx.Value(identKey{}).(Identity)
	if !ok {
		return Identity{}
	}
	return ident
}

// Register mounts Streamable HTTP /mcp when mcp.enabled is true.
func Register(app *fiber.App, auditor audit.Auditor) {
	if app == nil || !config.App.MCP.Enabled {
		return
	}
	handler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return newSDKServer(identityFrom(r.Context()), auditor)
	}, &mcpsdk.StreamableHTTPOptions{Stateless: true})
	app.All("/mcp", func(c fiber.Ctx) error {
		switch c.Method() {
		case http.MethodGet, http.MethodDelete:
			return fiber.NewError(fiber.StatusMethodNotAllowed)
		}
		ident, err := authenticate(c)
		if err != nil {
			return err
		}
		return adaptor.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r.WithContext(withIdentity(r.Context(), ident)))
		}))(c)
	})
}

func newSDKServer(ident Identity, auditor audit.Auditor) *mcpsdk.Server {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "flowbot",
		Version: version.Buildtags,
	}, nil)
	specs := FilterByScopes(catalogFromConfig(), ident.Scopes)
	for _, spec := range specs {
		spec := spec
		tool := &mcpsdk.Tool{
			Name:        spec.Name,
			Description: spec.Description,
			InputSchema: schemaFromParams(spec.Input),
			Annotations: toolHints(spec.Mutation),
		}
		srv.AddTool(tool, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			args := map[string]any{}
			if req != nil && req.Params != nil && len(req.Params.Arguments) > 0 {
				parsed := map[string]any{}
				if err := sonic.Unmarshal(req.Params.Arguments, &parsed); err != nil {
					return toolError(err), nil
				}
				if parsed != nil {
					args = parsed
				}
			}
			callIdent := identityFrom(ctx)
			if callIdent.UID.IsZero() {
				callIdent = ident
			}
			result, err := executeTool(ctx, callIdent, spec, args, auditor)
			if err != nil {
				flog.Error(err)
				return toolError(err), nil
			}
			return toolOK(result)
		})
	}
	return srv
}

func toolHints(mutation bool) *mcpsdk.ToolAnnotations {
	if !mutation {
		return &mcpsdk.ToolAnnotations{ReadOnlyHint: true}
	}
	destructive := true
	return &mcpsdk.ToolAnnotations{DestructiveHint: &destructive}
}

func toolOK(result any) (*mcpsdk.CallToolResult, error) {
	text, err := marshalResult(result)
	if err != nil {
		return toolError(err), nil
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}, nil
}

func toolError(err error) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: clientToolMessage(err)}},
	}
}

func clientToolMessage(err error) string {
	if err == nil {
		return "tool error"
	}
	var te *types.Error
	if errors.As(err, &te) && te != nil {
		if te.Message != "" {
			return te.Message
		}
		if te.Kind != nil {
			return te.Kind.Error()
		}
	}
	return "tool error"
}

func schemaFromParams(params []hub.ParamDef) map[string]any {
	fields := make([]pkgmcp.Field, 0, len(params))
	for _, p := range params {
		fields = append(fields, pkgmcp.Field{
			Name: p.Name, Type: p.Type, Description: p.Description, Required: p.Required,
		})
	}
	return pkgmcp.SchemaFromFields(fields)
}

func marshalResult(result any) (string, error) {
	if result == nil {
		return "{}", nil
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	raw, err := sonic.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
