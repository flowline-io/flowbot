package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func recordMCPExternalInputLink(ctx context.Context, ident Identity, spec ToolSpec, result any) {
	res, ok := result.(*capability.InvokeResult)
	if !ok || res == nil || res.Resource == nil {
		return
	}
	requestID := requestIDFrom(ctx)
	if requestID == "" {
		requestID = types.Id()
	}
	correlation := ""
	if lin, ok := types.LineageFrom(ctx); ok {
		correlation = lin.CorrelationID
	}
	link, ok := mcpExternalInputLink(ident, requestID, correlation, spec, res.Resource)
	if !ok {
		return
	}
	ps := store.PipelineStoreFromDB()
	if ps == nil {
		return
	}
	if err := store.NewPipelineRunStoreAdapter(ps).RecordResourceLink(ctx, link); err != nil {
		flog.Error(fmt.Errorf("mcp record resource link %s: %w", spec.Name, err))
		return
	}
	flog.Info("mcp recorded resource link %s -> %s/%s correlation=%s",
		link.SourceApp, link.TargetCapability, link.TargetEntityID, link.CorrelationID)
}

func mcpExternalInputLink(ident Identity, requestID, correlation string, spec ToolSpec, meta *capability.ResourceMeta) (model.ResourceLink, bool) {
	if meta == nil || strings.TrimSpace(meta.EntityID) == "" || strings.TrimSpace(meta.EventID) == "" {
		return model.ResourceLink{}, false
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return model.ResourceLink{}, false
	}
	targetApp := strings.TrimSpace(meta.App)
	if targetApp == "" {
		targetApp = spec.Group
	}
	return model.ResourceLink{
		SourceEventID:    types.ResourceLinkMCPSourceEventID(requestID),
		TargetEventID:    meta.EventID,
		SourceApp:        types.ResourceLinkMCPAgentApp(ident.TokenID),
		TargetApp:        targetApp,
		SourceCapability: "mcp",
		TargetCapability: spec.Name,
		SourceEntityID:   requestID,
		TargetEntityID:   meta.EntityID,
		PipelineName:     spec.Name,
		RunKind:          string(types.ResourceLinkRunKindMCP),
		Status:           string(types.ResourceLinkStatusOK),
		CorrelationID:    correlation,
	}, true
}
