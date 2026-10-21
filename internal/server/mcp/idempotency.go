package mcp

import (
	"context"
	"errors"

	"github.com/bytedance/sonic"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/idempotency"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

func toolInputSchema(spec ToolSpec) map[string]any {
	schema := schemaFromParams(spec.Input)
	tier, _ := capability.Classify(spec.Group, spec.Operation)
	if tier == capability.TierRead {
		return schema
	}
	return pkgmcp.WithIdempotencyKey(schema)
}

func runTool(ctx context.Context, ident Identity, spec ToolSpec, args map[string]any, auditor audit.Auditor) (*mcpsdk.CallToolResult, error) {
	key, err := idempotency.Take(args)
	if err != nil {
		return toolError(ctx, err), nil
	}
	tier, _ := capability.Classify(spec.Group, spec.Operation)
	if key != "" && tier != capability.TierRead {
		return runClaimedTool(ctx, ident, spec, args, auditor, key)
	}
	return executeAndWrap(ctx, ident, spec, args, auditor, "")
}

func runClaimedTool(ctx context.Context, ident Identity, spec ToolSpec, args map[string]any, auditor audit.Auditor, key string) (*mcpsdk.CallToolResult, error) {
	st := store.IdempotencyStoreFromDB()
	if !st.Ready() {
		return toolError(ctx, types.Errorf(types.ErrUnavailable, "idempotency store unavailable")), nil
	}
	if err := validateWriteArgs(spec, args); err != nil {
		return toolError(ctx, err), nil
	}
	paramHash, err := idempotency.Hash(args)
	if err != nil {
		return toolError(ctx, err), nil
	}
	correlation := types.NewCorrelationID()
	ctx = types.WithLineage(ctx, correlation, "")
	row, created, err := st.Begin(ctx, store.IdempotencyRef{
		TokenID:       ident.TokenID,
		Operation:     spec.Name,
		Key:           key,
		ParamHash:     paramHash,
		CorrelationID: correlation,
	})
	if err != nil {
		flog.Error(err)
		return toolError(ctx, err), nil
	}
	claim := store.IdempotencyView(row, paramHash, created)
	if claim.Outcome != idempotency.OutcomeRun {
		return respondClaim(ctx, ident, spec, auditor, key, claim)
	}
	executed, correlation, err := executePrepared(ctx, ident, spec, args, auditor, key)
	if err != nil {
		settleClaim(ctx, st, claim.ID, err)
		if !spec.Mutation {
			auditIdempotency(ctx, ident, spec, auditor, key, false, correlation, err)
		}
		flog.Error(err)
		return toolError(ctx, err), nil
	}
	raw, err := sonic.Marshal(executed)
	if err != nil {
		flog.Error(err)
		settleClaim(ctx, st, claim.ID, err)
	} else if err := st.Finish(ctx, claim.ID, raw, correlation, idempotency.MaxResultBytes()); err != nil {
		flog.Error(err)
	}
	if !spec.Mutation {
		auditIdempotency(ctx, ident, spec, auditor, key, false, correlation, nil)
	}
	out, err := toolOK(ctx, executed, wrapListResult(spec))
	if err != nil {
		flog.Error(err)
		return toolError(ctx, err), nil
	}
	return out, nil
}

func executeAndWrap(ctx context.Context, ident Identity, spec ToolSpec, args map[string]any, auditor audit.Auditor, idempotencyKey string) (*mcpsdk.CallToolResult, error) {
	executed, _, err := executePrepared(ctx, ident, spec, args, auditor, idempotencyKey)
	if err != nil {
		flog.Error(err)
		return toolError(ctx, err), nil
	}
	out, err := toolOK(ctx, executed, wrapListResult(spec))
	if err != nil {
		flog.Error(err)
		return toolError(ctx, err), nil
	}
	return out, nil
}

func executePrepared(ctx context.Context, ident Identity, spec ToolSpec, args map[string]any, auditor audit.Auditor, idempotencyKey string) (any, string, error) {
	ctx = capability.WithCursorSecretBox(ctx)
	if pkgmcp.IsListTool(spec.Group, spec.Operation) {
		ctx = capability.WithPageBudget(ctx, capability.PageBudget{
			Max:     idempotency.MaxResultBytes(),
			Measure: measureToolJSON(spec, traceFrom(ctx)),
		})
	}
	return executeTool(ctx, ident, spec, args, auditor, idempotencyKey)
}

func preflightIdempotency(ctx context.Context, ident Identity, spec ToolSpec, msg toolsCallEnvelope, auditor audit.Auditor) ([]byte, bool) {
	tier, _ := capability.Classify(spec.Group, spec.Operation)
	if tier == capability.TierRead {
		return nil, false
	}
	args, ok := toolArgs(msg)
	if !ok {
		return nil, false
	}
	if _, present := args["idempotency_key"]; !present {
		return nil, false
	}
	key, err := idempotency.Take(args)
	if err != nil {
		return rpcToolError(ctx, msg.ID, err)
	}
	if key == "" {
		return nil, false
	}
	st := store.IdempotencyStoreFromDB()
	if !st.Ready() {
		return rpcToolError(ctx, msg.ID, types.Errorf(types.ErrUnavailable, "idempotency store unavailable"))
	}
	paramHash, err := idempotency.Hash(args)
	if err != nil {
		return rpcToolError(ctx, msg.ID, err)
	}
	row, found, err := st.Lookup(ctx, store.IdempotencyRef{
		TokenID:   ident.TokenID,
		Operation: spec.Name,
		Key:       key,
		ParamHash: paramHash,
	})
	if err != nil {
		return rpcToolError(ctx, msg.ID, err)
	}
	if !found {
		if err := validateWriteArgs(spec, args); err != nil {
			return rpcToolError(ctx, msg.ID, err)
		}
		return nil, false
	}
	result, err := respondClaim(ctx, ident, spec, auditor, key, store.IdempotencyView(row, paramHash, false))
	if err != nil {
		return rpcToolError(ctx, msg.ID, err)
	}
	raw, err := sonic.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      msg.ID,
		"result":  result,
	})
	if err != nil {
		flog.Error(err)
		return nil, false
	}
	return raw, true
}

func toolArgs(msg toolsCallEnvelope) (map[string]any, bool) {
	args := map[string]any{}
	if len(msg.Params.Arguments) == 0 {
		return args, true
	}
	parsed := map[string]any{}
	if err := sonic.Unmarshal(msg.Params.Arguments, &parsed); err != nil {
		return nil, false
	}
	if parsed != nil {
		args = parsed
	}
	return args, true
}

func respondClaim(ctx context.Context, ident Identity, spec ToolSpec, auditor audit.Auditor, key string, claim idempotency.Claim) (*mcpsdk.CallToolResult, error) {
	if claim.Outcome == idempotency.OutcomeReplay {
		var payload any
		if err := sonic.Unmarshal([]byte(claim.ResultJSON), &payload); err != nil {
			err = idempotency.ErrorFor(idempotency.OutcomeUnknown)
			auditIdempotency(ctx, ident, spec, auditor, key, false, claim.CorrelationID, err)
			return toolError(ctx, err), nil
		}
		auditIdempotency(ctx, ident, spec, auditor, key, true, claim.CorrelationID, nil)
		ctx = withReplayed(ctx, claim.CorrelationID)
		out, err := toolOK(ctx, payload, wrapListResult(spec))
		if err != nil {
			return toolError(ctx, err), nil
		}
		return out, nil
	}
	err := idempotency.ErrorFor(claim.Outcome)
	auditIdempotency(ctx, ident, spec, auditor, key, false, claim.CorrelationID, err)
	return toolError(ctx, err), nil
}

func settleClaim(ctx context.Context, st *store.IdempotencyStore, id int64, callErr error) {
	var err error
	if errors.Is(callErr, types.ErrInvalidArgument) {
		err = st.Release(ctx, id)
	} else {
		err = st.Fail(ctx, id)
	}
	if err != nil {
		flog.Error(err)
	}
}

func auditIdempotency(ctx context.Context, ident Identity, spec ToolSpec, auditor audit.Auditor, key string, replayed bool, correlationID string, callErr error) {
	if auditor == nil || key == "" {
		return
	}
	entry := audit.Entry{
		Subject:       auditSubject(ident),
		Action:        "mcp.tool.call",
		Target:        audit.Target{Type: "mcp_tool", ID: spec.Name},
		CorrelationID: correlationID,
		Request: map[string]any{
			"tool":            spec.Name,
			"idempotency_key": key,
			"replayed":        replayed,
		},
	}
	var err error
	if callErr != nil {
		err = auditor.RecordFailure(ctx, entry, callErr)
	} else {
		err = auditor.RecordSuccess(ctx, entry)
	}
	if err != nil {
		flog.Error(err)
	}
}
