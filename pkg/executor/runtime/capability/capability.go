package capability

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	Prefix = "capability:"
)

type Runtime struct{}

func New() *Runtime {
	return &Runtime{}
}

func (r *Runtime) Run(ctx context.Context, t *types.Task) error {
	action := strings.TrimPrefix(t.Run, Prefix)
	dot := strings.LastIndex(action, ".")
	if dot < 0 {
		return fmt.Errorf("invalid capability action %q, expected capability:type.operation", t.Run)
	}

	capType := hub.CapabilityType(action[:dot])
	operation := action[dot+1:]

	params := make(map[string]any)
	if paramsJSON, ok := t.Env["CAPABILITY_PARAMS"]; ok && paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return fmt.Errorf("decode capability params: %w", err)
		}
	}

	authCtx := auth.SystemWorkflowContext()
	if uid, ok := t.Env["CAPABILITY_UID"]; ok {
		authCtx.UID = types.Uid(uid)
	}
	if topic, ok := t.Env["CAPABILITY_TOPIC"]; ok {
		authCtx.Topic = topic
	}
	_ = authCtx

	result, err := ability.Invoke(ctx, capType, operation, params)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", capType, operation, err)
	}

	out, _ := json.Marshal(result)
	t.Result = string(out)
	return nil
}

func (r *Runtime) Stop(ctx context.Context, t *types.Task) error {
	return nil
}

func (r *Runtime) HealthCheck(ctx context.Context) error {
	return nil
}
