package mcp

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/pkg/workflow"
)

func executeTool(ctx context.Context, ident Identity, spec ToolSpec, args map[string]any, auditor audit.Auditor) (any, error) {
	if args == nil {
		args = map[string]any{}
	}
	injectUID(spec, args, ident)
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
		result, err = execHub(ctx, spec.Operation)
	default:
		err = fmt.Errorf("unknown mcp tool kind %s", spec.Kind)
	}
	if spec.Mutation {
		recordMutation(ctx, ident, spec, args, auditor, err)
	}
	return result, err
}

func injectUID(spec ToolSpec, args map[string]any, ident Identity) {
	if ident.UID.IsZero() {
		return
	}
	for _, p := range spec.Input {
		if p.Name == "uid" {
			args["uid"] = string(ident.UID)
			return
		}
	}
}

func execPipeline(ctx context.Context, ident Identity, op string, args map[string]any) (any, error) {
	svc := pipeline.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "pipeline service not ready")
	}
	switch op {
	case "list":
		return svc.List(ctx)
	case "get":
		return withName(args, func(name string) (any, error) { return svc.Get(ctx, name) })
	case "run":
		return withName(args, func(name string) (any, error) {
			runID, err := svc.StartRunAsync(ctx, name, mapArg(args, "event"), string(ident.UID))
			if err != nil {
				return nil, err
			}
			return map[string]any{"run_id": runID}, nil
		})
	case "get_run":
		return withName(args, func(name string) (any, error) {
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

func execWorkflow(ctx context.Context, op string, args map[string]any) (any, error) {
	svc := workflow.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "workflow service not ready")
	}
	switch op {
	case "list":
		return svc.List(ctx)
	case "get":
		return withName(args, func(name string) (any, error) { return svc.Get(ctx, name) })
	case "run":
		return withName(args, func(name string) (any, error) {
			runID, err := svc.StartRunAsync(ctx, name, "manual", types.KV(mapArg(args, "input")))
			if err != nil {
				return nil, err
			}
			return map[string]any{"run_id": runID}, nil
		})
	case "get_run":
		return withName(args, func(name string) (any, error) {
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

func execFunction(ctx context.Context, op string, args map[string]any) (any, error) {
	svc := functions.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	switch op {
	case "list":
		return svc.List(ctx)
	case "get":
		return withName(args, func(name string) (any, error) { return svc.GetPublic(ctx, name, nil) })
	case "run":
		return withName(args, func(name string) (any, error) {
			req := functions.InvokeRequest{Name: name, Event: args["event"]}
			if v, ok := intArg(args["version"]); ok {
				ver := v
				req.Version = &ver
			}
			return svc.Invoke(ctx, req)
		})
	case "get_run":
		return withName(args, func(name string) (any, error) {
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

func execHub(ctx context.Context, op string) (any, error) {
	switch op {
	case "apps":
		return homelab.DefaultRegistry.List(), nil
	case "health":
		return hub.NewChecker(hub.Default).Check(ctx), nil
	default:
		return nil, fmt.Errorf("unknown hub operation %s", op)
	}
}

func recordMutation(ctx context.Context, ident Identity, spec ToolSpec, args map[string]any, auditor audit.Auditor, callErr error) {
	if auditor == nil {
		return
	}
	entry := audit.Entry{
		Subject: &audit.Subject{
			SubjectType: string(auth.SubjectToken),
			SubjectID:   string(ident.UID),
			UID:         string(ident.UID),
			IPAddress:   ident.IP,
			UserAgent:   ident.UserAgent,
		},
		Action:  "mcp.tool.call",
		Target:  audit.Target{Type: "mcp_tool", ID: spec.Name},
		Request: map[string]any{"tool": spec.Name, "args": args},
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

func requiredString(args map[string]any, key string) (string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return "", types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	s, ok := raw.(string)
	if !ok {
		s = fmt.Sprint(raw)
	}
	if s == "" || s == "<nil>" {
		return "", types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return s, nil
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
		return runs, nil
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
