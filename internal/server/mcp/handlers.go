package mcp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/pkg/validate"
	"github.com/flowline-io/flowbot/pkg/workflow"
)

func executeTool(ctx context.Context, ident Identity, spec ToolSpec, args map[string]any, auditor audit.Auditor, idempotencyKey string) (any, string, error) {
	if args == nil {
		args = map[string]any{}
	}
	injectUID(spec, args, ident)
	correlation := ""
	if lin, ok := types.LineageFrom(ctx); ok {
		correlation = lin.CorrelationID
	}
	if correlation == "" && (spec.Mutation || idempotencyKey != "") {
		correlation = types.NewCorrelationID()
		ctx = types.WithLineage(ctx, correlation, "")
	}
	if err := normalizeListArgs(spec, args); err != nil {
		return nil, "", err
	}
	var (
		result any
		err    error
	)
	switch spec.Kind {
	case kindCapability:
		result, err = capability.Invoke(ctx, hub.CapabilityType(spec.Group), spec.Operation, args)
	case kindPipeline:
		result, err = execPipeline(ctx, ident, spec.Operation, args)
	case kindWorkflow:
		result, err = execWorkflow(ctx, spec.Operation, args)
	case kindFunction:
		result, err = execFunction(ctx, spec.Operation, args)
	case kindHub:
		result, err = execHub(ctx, spec.Operation, args)
	case kindAudit:
		result, err = execAuditList(ctx, ident, args)
	default:
		err = fmt.Errorf("unknown mcp tool kind %s", spec.Kind)
	}
	if spec.Mutation {
		recordMutation(ctx, ident, spec, args, auditor, idempotencyKey, err)
	}
	return result, correlation, err
}

func injectUID(spec ToolSpec, args map[string]any, ident Identity) {
	if ident.UID.IsZero() {
		return
	}
	for _, p := range spec.Input {
		if !serverParam(p.Name) {
			continue
		}
		args[p.Name] = string(ident.UID)
	}
}

func execPipeline(ctx context.Context, ident Identity, op string, args map[string]any) (any, error) {
	switch op {
	case "list":
		svc, err := requirePipeline()
		if err != nil {
			return nil, err
		}
		recs, err := svc.List(ctx)
		if err != nil {
			return nil, err
		}
		return pageToolList(ctx, "pipeline", args, recs)
	case "get":
		return withName(args, func(name string) (any, error) {
			svc, err := requirePipeline()
			if err != nil {
				return nil, err
			}
			return svc.Get(ctx, name)
		})
	case "run":
		return withName(args, func(name string) (any, error) {
			svc, err := requirePipeline()
			if err != nil {
				return nil, err
			}
			runID, err := svc.StartRunAsync(ctx, name, mapArg(args, "event"), string(ident.UID))
			if err != nil {
				return nil, err
			}
			return map[string]any{"run_id": runID}, nil
		})
	case "get_run":
		return withName(args, func(name string) (any, error) {
			svc, err := requirePipeline()
			if err != nil {
				return nil, err
			}
			runs, err := svc.ListRuns(ctx, name)
			if err != nil {
				return nil, err
			}
			return filterModelRuns(runs, args["run_id"])
		})
	default:
		return nil, fmt.Errorf("unknown pipeline operation %s", op)
	}
}

func requirePipeline() (*pipeline.Service, error) {
	svc := pipeline.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	return svc, nil
}

func execWorkflow(ctx context.Context, op string, args map[string]any) (any, error) {
	switch op {
	case "list":
		svc, err := requireWorkflow()
		if err != nil {
			return nil, err
		}
		recs, err := svc.List(ctx)
		if err != nil {
			return nil, err
		}
		return pageToolList(ctx, "workflow", args, recs)
	case "get":
		return withName(args, func(name string) (any, error) {
			svc, err := requireWorkflow()
			if err != nil {
				return nil, err
			}
			return svc.Get(ctx, name)
		})
	case "run":
		return withName(args, func(name string) (any, error) {
			svc, err := requireWorkflow()
			if err != nil {
				return nil, err
			}
			runID, err := svc.StartRunAsync(ctx, name, "manual", types.KV(mapArg(args, "input")))
			if err != nil {
				return nil, err
			}
			return map[string]any{"run_id": runID}, nil
		})
	case "get_run":
		return withName(args, func(name string) (any, error) {
			svc, err := requireWorkflow()
			if err != nil {
				return nil, err
			}
			runs, err := svc.ListRuns(ctx, name)
			if err != nil {
				return nil, err
			}
			return filterModelRuns(runs, args["run_id"])
		})
	default:
		return nil, fmt.Errorf("unknown workflow operation %s", op)
	}
}

func requireWorkflow() (*workflow.Service, error) {
	svc := workflow.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "workflow service not ready")
	}
	return svc, nil
}

func execFunction(ctx context.Context, op string, args map[string]any) (any, error) {
	switch op {
	case "list":
		svc, err := requireFunction()
		if err != nil {
			return nil, err
		}
		recs, err := svc.List(ctx)
		if err != nil {
			return nil, err
		}
		return pageToolList(ctx, "function", args, recs)
	case "get":
		return withName(args, func(name string) (any, error) {
			svc, err := requireFunction()
			if err != nil {
				return nil, err
			}
			return svc.GetPublic(ctx, name, nil)
		})
	case "run":
		return withName(args, func(name string) (any, error) {
			svc, err := requireFunction()
			if err != nil {
				return nil, err
			}
			req := functions.InvokeRequest{Name: name, Event: args["event"]}
			if v, ok := intArg(args["version"]); ok {
				ver := v
				req.Version = &ver
			}
			return svc.Invoke(ctx, req)
		})
	case "get_run":
		return withName(args, func(name string) (any, error) {
			svc, err := requireFunction()
			if err != nil {
				return nil, err
			}
			runs, err := svc.ListRuns(ctx, name)
			if err != nil {
				return nil, err
			}
			return filterModelRuns(runs, args["run_id"])
		})
	default:
		return nil, fmt.Errorf("unknown function operation %s", op)
	}
}

func requireFunction() (*functions.Service, error) {
	svc := functions.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	return svc, nil
}

func execHub(ctx context.Context, op string, args map[string]any) (any, error) {
	switch op {
	case "apps":
		return pageToolList(ctx, "hub", args, homelab.DefaultRegistry.List())
	case "health":
		return hub.NewChecker(hub.Default).Check(ctx), nil
	default:
		return nil, fmt.Errorf("unknown hub operation %s", op)
	}
}

func pageToolList(ctx context.Context, capName string, args map[string]any, items any) (any, error) {
	data, page := capability.ApplyFullListPage(ctx, capName, args, items, nil)
	data, page = capability.ClampListed(ctx, capName, args, data, page, nil)
	if data == nil {
		data = []any{}
	}
	return map[string]any{"items": data, "page": page}, nil
}

func normalizeListArgs(spec ToolSpec, args map[string]any) error {
	if !pkgmcp.IsListTool(spec.Group, spec.Operation) {
		return nil
	}
	if raw, ok := args["cursor"].(string); ok && len(raw) > validate.CursorMaxLen {
		return &types.Error{
			Kind:    types.ErrInvalidArgument,
			Message: fmt.Sprintf("cursor exceeds maximum length of %d", validate.CursorMaxLen),
			Field:   "cursor",
		}
	}
	limit := 20
	if raw, ok := args["limit"]; ok && raw != nil {
		n, conv := intArg(raw)
		if conv && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}
	args["limit"] = limit
	return nil
}

func recordMutation(ctx context.Context, ident Identity, spec ToolSpec, args map[string]any, auditor audit.Auditor, idempotencyKey string, callErr error) {
	if auditor == nil {
		return
	}
	request := map[string]any{"tool": spec.Name, "args": args}
	if idempotencyKey != "" {
		request["idempotency_key"] = idempotencyKey
		request["replayed"] = false
	}
	entry := audit.Entry{
		Subject: auditSubject(ident),
		Action:  "mcp.tool.call",
		Target:  audit.Target{Type: "mcp_tool", ID: spec.Name},
		Request: request,
	}
	if lin, ok := types.LineageFrom(ctx); ok {
		entry.CorrelationID = lin.CorrelationID
	}
	if callErr != nil {
		if err := auditor.RecordFailure(ctx, entry, callErr); err != nil {
			flog.Error(fmt.Errorf("mcp audit failure: %w", err))
		}
		return
	}
	if err := auditor.RecordSuccess(ctx, entry); err != nil {
		flog.Error(fmt.Errorf("mcp audit success: %w", err))
	}
}

func execAuditList(ctx context.Context, ident Identity, args map[string]any) (any, error) {
	subject := stringArg(args, "subject_id")
	opts := store.ListAuditOptions{
		Action:     stringArg(args, "action"),
		TargetType: stringArg(args, "target_type"),
		Result:     stringArg(args, "result"),
		Cursor:     stringArg(args, "cursor"),
	}
	if n, ok := intArg(args["limit"]); ok {
		opts.Limit = n
	}
	if raw := stringArg(args, "since"); raw != "" {
		ts, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, &types.Error{Kind: types.ErrInvalidArgument, Message: "since must be RFC3339", Field: "since"}
		}
		opts.Since = ts
	}
	if raw := stringArg(args, "until"); raw != "" {
		ts, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, &types.Error{Kind: types.ErrInvalidArgument, Message: "until must be RFC3339", Field: "until"}
		}
		opts.Until = ts
	}
	rows, cursor, err := store.AuditStoreFromDB().ListForCaller(ctx, store.AuditCaller{
		UID:       string(ident.UID),
		Admin:     auth.HasScope(ident.Scopes, auth.ScopeAdmin),
		SubjectID: subject,
	}, opts)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.Item())
	}
	return map[string]any{"items": items, "page": map[string]any{"cursor": cursor}}, nil
}

func auditSubject(ident Identity) *audit.Subject {
	return &audit.Subject{
		SubjectType: string(auth.SubjectToken),
		SubjectID:   string(ident.UID),
		UID:         string(ident.UID),
		IPAddress:   ident.IP,
		UserAgent:   ident.UserAgent,
	}
}

func stringArg(args map[string]any, key string) string {
	value, ok := args[key].(string)
	if !ok {
		return ""
	}
	return value
}

func withName(args map[string]any, fn func(name string) (any, error)) (any, error) {
	name, err := requiredString(args, "name")
	if err != nil {
		return nil, err
	}
	return fn(name)
}

func mapArg(args map[string]any, key string) map[string]any {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func validateWriteArgs(spec ToolSpec, args map[string]any) error {
	if err := normalizeListArgs(spec, args); err != nil {
		return err
	}
	for _, param := range spec.Input {
		if !param.Required || param.Name == "idempotency_key" {
			continue
		}
		raw, ok := args[param.Name]
		if !ok || raw == nil {
			return invalidArgument(param.Name)
		}
		if text, isString := raw.(string); isString && strings.TrimSpace(text) == "" {
			return invalidArgument(param.Name)
		}
	}
	return nil
}

func requiredString(args map[string]any, key string) (string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return "", invalidArgument(key)
	}
	s, ok := raw.(string)
	if !ok {
		s = fmt.Sprint(raw)
	}
	if s == "" || s == "<nil>" {
		return "", invalidArgument(key)
	}
	return s, nil
}

func invalidArgument(field string) error {
	return &types.Error{
		Kind:    types.ErrInvalidArgument,
		Message: field + " is required",
		Field:   field,
	}
}

func intArg(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case string:
		i, err := strconv.Atoi(n)
		return i, err == nil
	default:
		return 0, false
	}
}

func int64Arg(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

func filterModelRuns[T any](runs []T, runID any) (any, error) {
	want, ok := int64Arg(runID)
	if !ok {
		if runs == nil {
			runs = []T{}
		}
		return map[string]any{"runs": runs}, nil
	}
	for _, r := range runs {
		id, ok := runIDOf(r)
		if ok && id == want {
			return r, nil
		}
	}
	return nil, types.Errorf(types.ErrNotFound, "run %d not found", want)
}

func runIDOf(v any) (int64, bool) {
	switch r := v.(type) {
	case interface{ GetID() int64 }:
		return r.GetID(), true
	default:
		raw, err := sonic.Marshal(v)
		if err != nil {
			return 0, false
		}
		var row map[string]any
		if err := sonic.Unmarshal(raw, &row); err != nil {
			return 0, false
		}
		id, ok := int64Arg(row["id"])
		return id, ok
	}
}
