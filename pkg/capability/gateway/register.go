package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers CapGateway when gateway.enabled is true and a JobStore is set.
func Register() error {
	if !config.App.Gateway.Enabled {
		return nil
	}
	if jobStore() == nil {
		return nil
	}
	return capability.Register(buildSpec())
}

// CatalogSpec returns capability metadata for documentation (handlers must not be invoked).
func CatalogSpec() capability.Spec {
	return buildSpec()
}

func buildSpec() capability.Spec {
	return capability.Spec{
		Type:        hub.CapGateway,
		Description: "Local CLI gateway: delegate coarse run_cursor jobs to cmd/gateway workers",
		Instance:    struct{}{},
		Ops: []capability.OpDef{
			{
				Name: OpRun, Description: "Create a local CLI job and wait for the terminal result", Mutation: true,
				Input: []hub.ParamDef{
					{Name: "prompt", Type: "string", Required: true, Description: "Prompt for the local Cursor CLI"},
					{Name: "cwd", Type: "string", Required: false, Description: "Optional workspace path on the worker machine"},
					{Name: "uid", Type: "string", Required: false, Description: "Owner UID for audit"},
					{Name: "cli", Type: "string", Required: false, Description: "CLI id; only cursor is supported in v1"},
				},
				Handler: runInvoker,
			},
			{Name: OpHealth, Description: "Report whether a fresh gateway worker is online", Handler: healthInvoker},
			{
				Name: OpCancel, Description: "Cancel a gateway job by id", Mutation: true,
				Input:   []hub.ParamDef{{Name: "job_id", Type: "string", Required: true, Description: "Job id"}},
				Handler: cancelInvoker,
			},
		},
	}
}

func runTimeout() time.Duration {
	d := config.App.Gateway.RunTimeout
	if d <= 0 {
		return 30 * time.Minute
	}
	return d
}

func workerStaleAfter() time.Duration {
	d := config.App.Gateway.WorkerStaleAfter
	if d <= 0 {
		return 60 * time.Second
	}
	return d
}

func runInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	store := jobStore()
	if store == nil {
		return nil, types.Errorf(types.ErrUnavailable, "gateway job store not configured")
	}
	if err := requireFreshWorker(ctx, store); err != nil {
		return nil, err
	}
	in, err := parseRunParams(params)
	if err != nil {
		return nil, err
	}
	job, err := store.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	return waitForJob(ctx, store, job.JobID)
}

func requireFreshWorker(ctx context.Context, store JobStore) error {
	fresh, err := store.HasFreshWorker(ctx, workerStaleAfter())
	if err != nil {
		return err
	}
	if !fresh {
		return types.Errorf(types.ErrUnavailable, "no online gateway worker")
	}
	return nil
}

func parseRunParams(params map[string]any) (types.GatewayCreateJob, error) {
	prompt, err := capability.RequiredString(params, "prompt")
	if err != nil {
		return types.GatewayCreateJob{}, err
	}
	cli := types.GatewayCLICursor
	if raw, ok := capability.StringParam(params, "cli"); ok && strings.TrimSpace(raw) != "" {
		cli = types.GatewayCLI(strings.TrimSpace(raw))
	}
	if cli != types.GatewayCLICursor {
		return types.GatewayCreateJob{}, types.Errorf(types.ErrInvalidArgument, "unsupported gateway cli %q", cli)
	}
	cwd, _ := capability.StringParam(params, "cwd")
	uid, _ := capability.StringParam(params, "uid")
	return types.GatewayCreateJob{UID: uid, CLI: cli, Prompt: prompt, Cwd: cwd}, nil
}

func waitForJob(ctx context.Context, store JobStore, jobID string) (*capability.InvokeResult, error) {
	waitCtx, cancel := context.WithTimeout(ctx, runTimeout())
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-waitCtx.Done():
			if _, cerr := store.Cancel(context.WithoutCancel(ctx), jobID); cerr != nil {
				flog.Warn("gateway: cancel after wait end job_id=%s: %v", jobID, cerr)
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, types.Errorf(types.ErrTimeout, "gateway job timed out")
		case <-ticker.C:
			if err := store.ReclaimExpired(waitCtx); err != nil {
				flog.Warn("gateway: reclaim during wait job_id=%s: %v", jobID, err)
			}
			cur, gerr := store.Get(waitCtx, jobID)
			if gerr != nil {
				return nil, gerr
			}
			if cur == nil {
				return nil, types.Errorf(types.ErrNotFound, "gateway job disappeared")
			}
			if types.GatewayJobTerminal(cur.Status) {
				return &capability.InvokeResult{Data: cur}, nil
			}
		}
	}
}

func healthInvoker(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
	store := jobStore()
	if store == nil {
		return &capability.InvokeResult{Data: map[string]any{"healthy": false, "reason": "store not configured"}}, nil
	}
	fresh, err := store.HasFreshWorker(ctx, workerStaleAfter())
	if err != nil {
		return nil, fmt.Errorf("gateway health: %w", err)
	}
	return &capability.InvokeResult{Data: map[string]any{"healthy": fresh}}, nil
}

func cancelInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	store := jobStore()
	if store == nil {
		return nil, types.Errorf(types.ErrUnavailable, "gateway job store not configured")
	}
	jobID, err := capability.RequiredString(params, "job_id")
	if err != nil {
		return nil, err
	}
	job, err := store.Cancel(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return &capability.InvokeResult{Data: job}, nil
}
