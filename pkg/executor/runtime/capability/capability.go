// Package capability implements capability-based execution runtime.
package capability

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Prefix identifies tasks that should be handled by the capability runtime.
const (
	Prefix = "capability:"
)

// Runtime executes tasks by invoking registered ability capabilities.
type Runtime struct{}

// New creates a new capability runtime.
func New() *Runtime {
	return &Runtime{}
}

// Run parses the task's Run field as "capability:<type>.<operation>", decodes
// params from the CAPABILITY_PARAMS env var, and invokes the matching ability.
func (*Runtime) Run(ctx context.Context, t *types.Task) error {
	action := strings.TrimPrefix(t.Run, Prefix)
	dot := strings.LastIndex(action, ".")
	if dot < 0 {
		return fmt.Errorf("invalid capability action %q, expected capability:type.operation", t.Run)
	}

	capType := action[:dot]
	operation := action[dot+1:]

	params := make(map[string]any)
	if paramsJSON, ok := t.Env["CAPABILITY_PARAMS"]; ok && paramsJSON != "" {
		if err := sonic.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return fmt.Errorf("decode capability params: %w", err)
		}
	}

	// Forward optional identity hints from the task environment to the
	// capability as params so that implementations can use them for
	// authorization or scoping decisions.
	if uid, ok := t.Env["CAPABILITY_UID"]; ok {
		params["_uid"] = uid
	}
	if topic, ok := t.Env["CAPABILITY_TOPIC"]; ok {
		params["_topic"] = topic
	}

	result, err := ability.Invoke(ctx, hub.CapabilityType(capType), operation, params)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", capType, operation, err)
	}

	out, err := sonic.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode capability result: %w", err)
	}
	t.Result = string(out)
	return nil
}

// Stop is a no-op for the capability runtime since invocations are synchronous.
func (*Runtime) Stop(_ context.Context, _ *types.Task) error {
	return nil
}

// HealthCheck always returns nil because the capability runtime has no
// external dependencies to verify.
func (*Runtime) HealthCheck(_ context.Context) error {
	return nil
}

// Close is a no-op for the capability runtime.
func (*Runtime) Close() error {
	return nil
}
