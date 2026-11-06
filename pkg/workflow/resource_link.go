package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func (r *Runner) saveWorkflowResourceLink(ctx context.Context, run *model.WorkflowRun, wfName, stepID string, info ActionInfo, resultJSON string) {
	meta := resourceMetaFromCapabilityResult(resultJSON)
	if meta == nil || meta.EntityID == "" || meta.EventID == "" || run == nil || r.store == nil {
		return
	}
	corr := ""
	if lin, ok := types.LineageFrom(ctx); ok {
		corr = lin.CorrelationID
	}
	link := model.ResourceLink{
		SourceEventID:    types.ResourceLinkWorkflowSourceEventID(run.ID),
		TargetEventID:    meta.EventID,
		SourceApp:        "workflow",
		TargetApp:        strings.TrimSpace(meta.App),
		SourceCapability: "workflow",
		TargetCapability: info.CapType,
		SourceEntityID:   wfName,
		TargetEntityID:   meta.EntityID,
		PipelineRunID:    run.ID,
		PipelineName:     wfName,
		RunKind:          string(types.ResourceLinkRunKindWorkflow),
		Status:           string(types.ResourceLinkStatusOK),
		CorrelationID:    corr,
	}
	if link.TargetApp == "" {
		link.TargetApp = info.CapType
	}
	if err := r.store.RecordResourceLink(workflowStoreCtx(ctx), link); err != nil {
		flog.Error(fmt.Errorf("record workflow resource link %s step %s: %w", wfName, stepID, err))
		return
	}
	flog.Info("[workflow] recorded resource link %s -> %s/%s", link.SourceEventID, link.TargetCapability, link.TargetEntityID)
}

func (r *Runner) saveWorkflowFailedResourceLink(ctx context.Context, run *model.WorkflowRun, wfName, stepID string, info ActionInfo) {
	if run == nil || r.store == nil || run.ID == 0 {
		return
	}
	corr := ""
	if lin, ok := types.LineageFrom(ctx); ok {
		corr = lin.CorrelationID
	}
	capName := info.CapType
	if capName == "" {
		capName = info.Type
	}
	link := model.ResourceLink{
		SourceEventID:    types.ResourceLinkWorkflowSourceEventID(run.ID),
		TargetEventID:    types.ResourceLinkFailTargetEventID(run.ID, stepID),
		SourceApp:        "workflow",
		TargetApp:        capName,
		SourceCapability: "workflow",
		TargetCapability: capName,
		SourceEntityID:   wfName,
		PipelineRunID:    run.ID,
		PipelineName:     wfName,
		RunKind:          string(types.ResourceLinkRunKindWorkflow),
		Status:           string(types.ResourceLinkStatusFailed),
		CorrelationID:    corr,
	}
	if err := r.store.RecordResourceLink(workflowStoreCtx(ctx), link); err != nil {
		flog.Error(fmt.Errorf("record workflow failed resource link %s step %s: %w", wfName, stepID, err))
		return
	}
	flog.Info("[workflow] recorded failed resource link %s -> %s", link.SourceEventID, link.TargetEventID)
}

func resourceMetaFromCapabilityResult(resultJSON string) *capability.ResourceMeta {
	resultJSON = strings.TrimSpace(resultJSON)
	if resultJSON == "" {
		return nil
	}
	var ir capability.InvokeResult
	if err := sonic.Unmarshal([]byte(resultJSON), &ir); err != nil {
		return nil
	}
	return ir.Resource
}
