package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/version"
)

type identKey struct{}

type requestIDCtxKey struct{}

func withRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDCtxKey{}, id)
}

func requestIDFrom(ctx context.Context) string {
	id, ok := ctx.Value(requestIDCtxKey{}).(string)
	if !ok {
		return ""
	}
	return id
}

func contextForTool(c fiber.Ctx, r *http.Request, ident Identity) context.Context {
	base := context.Context(context.Background())
	if r != nil && r.Context() != nil {
		base = r.Context()
	}
	if fiberCtx, ok := adaptor.LocalContextFromHTTPRequest(r); ok && fiberCtx != nil {
		base = fiberCtx
	} else if c != nil {
		base = c.Context()
	}
	rid := requestid.FromContext(base)
	if rid == "" && c != nil {
		rid = requestid.FromContext(c)
	}
	if rid == "" && r != nil {
		rid = r.Header.Get("X-Request-ID")
	}
	return withIdentity(withRequestID(base, rid), ident)
}

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
	registerServerCard(app)
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
		body := c.Body()
		specs := catalogFromConfig()
		if raw, ok := preflightToolsCall(c.Context(), body, ident, specs); ok {
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.Send(raw)
		}
		if raw, ok := scopeDeniedResponse(body, ident, specs); ok {
			recordScopeRejection(c.Context(), body, ident, specs, auditor)
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.Send(raw)
		}
		return adaptor.HTTPHandlerWithContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r.WithContext(contextForTool(c, r, ident)))
		}))(c)
	})
}

type toolsCallEnvelope struct {
	ID     any             `json:"id"`
	Method string          `json:"method"`
	Params toolsCallParams `json:"params"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func preflightToolsCall(ctx context.Context, body []byte, ident Identity, specs []ToolSpec) ([]byte, bool) {
	var msg toolsCallEnvelope
	if err := sonic.Unmarshal(body, &msg); err != nil || msg.Method != "tools/call" || msg.Params.Name == "" {
		return nil, false
	}
	if argumentByteCount(msg.Params.Arguments) > maxArgumentBytes() {
		return rpcToolError(ctx, msg.ID, &types.Error{
			Kind:    types.ErrInvalidArgument,
			Message: "arguments exceed the configured size limit",
			Field:   "arguments",
		})
	}
	spec, found := findTool(specs, msg.Params.Name)
	allowed, retryAfter, err := allowToolCall(ctx, bucketsForTool(ident.TokenID, spec, found), time.Now())
	if err != nil {
		return rpcToolError(ctx, msg.ID, err)
	}
	if !allowed {
		return rpcToolError(ctx, msg.ID, &types.Error{
			Kind:       types.ErrRateLimited,
			Message:    "rate limit exceeded",
			RetryAfter: retryAfter,
		})
	}
	return nil, false
}

func findTool(specs []ToolSpec, name string) (ToolSpec, bool) {
	for _, item := range specs {
		if item.Name == name {
			return item, true
		}
	}
	return ToolSpec{}, false
}

func rpcToolError(ctx context.Context, id any, err error) ([]byte, bool) {
	raw, marshalErr := sonic.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  toolError(ctx, err),
	})
	if marshalErr != nil {
		return nil, false
	}
	return raw, true
}

func recordScopeRejection(ctx context.Context, body []byte, ident Identity, specs []ToolSpec, auditor audit.Auditor) {
	if auditor == nil {
		return
	}
	var msg toolsCallEnvelope
	if err := sonic.Unmarshal(body, &msg); err != nil {
		return
	}
	spec, found := findTool(specs, msg.Params.Name)
	if !found || !spec.Mutation {
		return
	}
	if err := auditor.RecordRejected(ctx, audit.Entry{
		Subject: auditSubject(ident),
		Action:  "mcp.tool.call",
		Target:  audit.Target{Type: "mcp_tool", ID: spec.Name},
		Request: map[string]any{
			"tool": spec.Name,
		},
	}, "permission_denied"); err != nil {
		flog.Error(fmt.Errorf("mcp audit rejected: %w", err))
	}
}

func scopeDeniedResponse(body []byte, ident Identity, specs []ToolSpec) ([]byte, bool) {
	var msg toolsCallEnvelope
	if err := sonic.Unmarshal(body, &msg); err != nil || msg.Method != "tools/call" || msg.Params.Name == "" {
		return nil, false
	}
	spec, found := findTool(specs, msg.Params.Name)
	if !found || spec.Allowed(ident.Scopes) {
		return nil, false
	}
	result := toolError(context.Background(), types.Errorf(types.ErrForbidden, "missing scope"))
	raw, err := sonic.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      msg.ID,
		"result":  result,
	})
	if err != nil {
		return nil, false
	}
	return raw, true
}

func newSDKServer(ident Identity, auditor audit.Auditor) *mcpsdk.Server {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "flowbot",
		Version: version.Buildtags,
	}, &mcpsdk.ServerOptions{Instructions: pkgmcp.ServerInstructions})
	specs := FilterByScopes(catalogFromConfig(), ident.Scopes)
	for _, spec := range specs {
		tool := &mcpsdk.Tool{
			Name:        spec.Name,
			Description: spec.Description,
			InputSchema: schemaFromParams(spec.Input),
			Annotations: toolAnnotations(spec),
		}
		if spec.Output != nil {
			tool.OutputSchema = spec.Output
		}
		srv.AddTool(tool, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			args := map[string]any{}
			if req != nil && req.Params != nil && len(req.Params.Arguments) > 0 {
				parsed := map[string]any{}
				if err := sonic.Unmarshal(req.Params.Arguments, &parsed); err != nil {
					return toolError(ctx, &types.Error{
						Kind:    types.ErrInvalidArgument,
						Message: "arguments must be a JSON object",
						Cause:   err,
					}), nil
				}
				if parsed != nil {
					args = parsed
				}
			}
			callIdent := identityFrom(ctx)
			if callIdent.UID.IsZero() {
				callIdent = ident
			}
			ctx = capability.WithCursorSecretBox(ctx)
			if pkgmcp.IsListTool(spec.Group, spec.Operation) {
				maxOut := config.App.MCP.MaxOutputBytes
				if maxOut <= 0 {
					maxOut = capability.DefaultMaxOutputBytes
				}
				ctx = capability.WithPageBudget(ctx, capability.PageBudget{
					Max:     maxOut,
					Measure: measureToolJSON(spec, traceFrom(ctx)),
				})
			}
			result, err := executeTool(ctx, callIdent, spec, args, auditor)
			if err != nil {
				flog.Error(err)
				return toolError(ctx, err), nil
			}
			out, err := toolOK(ctx, result, wrapListResult(spec))
			if err != nil {
				flog.Error(err)
				return toolError(ctx, err), nil
			}
			return out, nil
		})
	}
	return srv
}

func toolAnnotations(spec ToolSpec) *mcpsdk.ToolAnnotations {
	tier, idempotent := capability.Classify(spec.Group, spec.Operation)
	open := pkgmcp.OpenWorld(spec.Group)
	ann := &mcpsdk.ToolAnnotations{
		Title:         pkgmcp.Title(spec.Group, spec.Description),
		ReadOnlyHint:  tier == capability.TierRead,
		OpenWorldHint: &open,
	}
	if tier != capability.TierRead {
		destructive := tier == capability.TierDestructive
		ann.DestructiveHint = &destructive
		ann.IdempotentHint = idempotent
	}
	return ann
}

func wrapListResult(spec ToolSpec) bool {
	switch spec.Name {
	case "pipeline.list", "workflow.list", "function.list", "hub.apps":
		return true
	default:
		return false
	}
}

func toolOK(ctx context.Context, result any, wrapItems bool) (*mcpsdk.CallToolResult, error) {
	payload, err := pkgmcp.AttachTrace(result, traceFrom(ctx), wrapItems)
	if err != nil {
		return nil, err
	}
	return callResult(false, payload)
}

func toolError(ctx context.Context, err error) *mcpsdk.CallToolResult {
	payload := pkgmcp.ErrorEnvelope(ToToolError(err).Object(), traceFrom(ctx))
	res, marshalErr := callResult(true, payload)
	if marshalErr != nil {
		return internalToolError()
	}
	return res
}

func internalToolError() *mcpsdk.CallToolResult {
	raw, err := sonic.Marshal(pkgmcp.ErrorEnvelope(ToToolError(nil).Object(), map[string]any{}))
	text := ""
	if err == nil {
		text = string(raw)
	}
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}
}

func callResult(isError bool, payload any) (*mcpsdk.CallToolResult, error) {
	raw, err := sonic.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := sonic.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return &mcpsdk.CallToolResult{
		IsError:           isError,
		Content:           []mcpsdk.Content{&mcpsdk.TextContent{Text: string(raw)}},
		StructuredContent: generic,
	}, nil
}

func traceFrom(ctx context.Context) map[string]any {
	trace := map[string]any{}
	if id := requestIDFrom(ctx); id != "" {
		trace["request_id"] = id
	}
	sc := oteltrace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() && sc.HasTraceID() {
		trace["trace_id"] = sc.TraceID().String()
	}
	return trace
}

func measureToolJSON(spec ToolSpec, trace map[string]any) func(items []any, page map[string]any) (int, error) {
	return func(items []any, page map[string]any) (int, error) {
		doc := map[string]any{"page": page, "trace": trace}
		if spec.Kind == kindCapability {
			doc["capability"] = spec.Group
			doc["operation"] = spec.Operation
			doc["data"] = items
		} else {
			doc["items"] = items
		}
		raw, err := sonic.Marshal(doc)
		if err != nil {
			return 0, err
		}
		return len(raw), nil
	}
}

func schemaFromParams(params []hub.ParamDef) map[string]any {
	fields := make([]pkgmcp.Field, 0, len(params))
	for _, p := range params {
		fields = append(fields, pkgmcp.Field{
			Name: p.Name, Type: p.Type, Description: p.Description, Required: p.Required, Enum: p.Enum,
		})
	}
	return pkgmcp.SchemaFromFields(fields)
}
