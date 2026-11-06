package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
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
	link := mcpExternalInputLink(ident, requestID, correlation, spec, res.Resource)
	if link == nil {
		return
	}
	if err := store.PipelineStoreFromDB().RecordResourceLink(ctx, link); err != nil {
		flog.Error(fmt.Errorf("mcp record resource link %s: %w", spec.Name, err))
		return
	}
	flog.Info("mcp recorded resource link %s -> %s/%s correlation=%s",
		link.SourceApp, link.TargetCapability, link.TargetEntityID, link.CorrelationID)
}

func mcpExternalInputLink(ident Identity, requestID, correlation string, spec ToolSpec, meta *capability.ResourceMeta) *gen.ResourceLink {
	if meta == nil || strings.TrimSpace(meta.EntityID) == "" || strings.TrimSpace(meta.EventID) == "" {
		return nil
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil
	}
	link := &gen.ResourceLink{
		SourceEventID:    types.ResourceLinkMCPSourceEventID(requestID),
		TargetEventID:    meta.EventID,
		SourceApp:        types.ResourceLinkMCPAgentApp(ident.TokenID),
		TargetApp:        strings.TrimSpace(meta.App),
		SourceCapability: "mcp",
		TargetCapability: spec.Name,
		SourceEntityID:   requestID,
		TargetEntityID:   meta.EntityID,
		PipelineName:     spec.Name,
		RunKind:          resourcelink.RunKind(types.ResourceLinkRunKindMCP),
		Status:           resourcelink.Status(types.ResourceLinkStatusOK),
		CorrelationID:    correlation,
	}
	if link.TargetApp == "" {
		link.TargetApp = spec.Group
	}
	return link
}
