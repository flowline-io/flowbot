package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/drone"
	"github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/workflow"
)

func TestInjectUIDOverwritesCallerValue(t *testing.T) {
	t.Parallel()
	args := map[string]any{"uid": "attacker"}
	injectUID(ToolSpec{
		Input: []hub.ParamDef{{Name: "uid"}},
	}, args, Identity{UID: types.Uid("token-user")})
	assert.Equal(t, "token-user", args["uid"])
}

func TestInjectUIDSkipsWhenNoUIDField(t *testing.T) {
	t.Parallel()
	args := map[string]any{"name": "x"}
	injectUID(ToolSpec{
		Input: []hub.ParamDef{{Name: "name"}},
	}, args, Identity{UID: types.Uid("token-user")})
	assert.Equal(t, "x", args["name"])
	assert.Nil(t, args["uid"])
}

func TestNamedToolRejectsMissingNameBeforeService(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		call func() error
	}{
		{"pipeline.get", func() error {
			_, err := execPipeline(context.Background(), Identity{}, "get", map[string]any{})
			return err
		}},
		{"pipeline.run", func() error {
			_, err := execPipeline(context.Background(), Identity{}, "run", map[string]any{})
			return err
		}},
		{"pipeline.get_run", func() error {
			_, err := execPipeline(context.Background(), Identity{}, "get_run", map[string]any{})
			return err
		}},
		{"workflow.get", func() error {
			_, err := execWorkflow(context.Background(), "get", map[string]any{})
			return err
		}},
		{"workflow.run", func() error {
			_, err := execWorkflow(context.Background(), "run", map[string]any{})
			return err
		}},
		{"workflow.get_run", func() error {
			_, err := execWorkflow(context.Background(), "get_run", map[string]any{})
			return err
		}},
		{"function.get", func() error {
			_, err := execFunction(context.Background(), "get", map[string]any{})
			return err
		}},
		{"function.run", func() error {
			_, err := execFunction(context.Background(), "run", map[string]any{})
			return err
		}},
		{"function.get_run", func() error {
			_, err := execFunction(context.Background(), "get_run", map[string]any{})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var te *types.Error
			require.ErrorAs(t, tc.call(), &te)
			assert.Equal(t, types.ErrInvalidArgument, te.Kind)
			assert.Equal(t, "name", te.Field)
		})
	}
}

func TestDroneDeployUsesTokenSubject(t *testing.T) {
	var got types.Uid
	require.NoError(t, capability.RegisterInvoker(hub.CapDrone, drone.OpDeploy,
		func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
			uid, _ := capability.StringParam(params, "uid")
			got = types.Uid(uid)
			return &capability.InvokeResult{Data: map[string]any{}}, nil
		}))
	t.Cleanup(func() { capability.UnregisterInvoker(hub.CapDrone, drone.OpDeploy) })

	_, _, err := executeTool(context.Background(), Identity{UID: "token-user"},
		droneToolSpec(t, drone.OpDeploy), map[string]any{"uid": "attacker"}, nil, "")
	require.NoError(t, err)
	assert.Equal(t, types.Uid("token-user"), got)
}

func TestDroneDeploySchemaHidesUID(t *testing.T) {
	schema := toolInputSchema(droneToolSpec(t, drone.OpDeploy))
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, props, "uid")
	assert.Contains(t, props, "idempotency_key")
}

func droneToolSpec(t *testing.T, operation string) ToolSpec {
	t.Helper()
	for _, op := range drone.CatalogSpec().Ops {
		if op.Name == operation {
			return ToolSpec{
				Name:      "drone." + op.Name,
				Group:     "drone",
				Operation: op.Name,
				Kind:      kindCapability,
				Mutation:  op.Mutation,
				Input:     op.Input,
			}
		}
	}
	t.Fatalf("drone operation %q not found", operation)
	return ToolSpec{}
}

func TestNamedToolUnavailableWhenServiceMissing(t *testing.T) {
	if pipeline.ActiveService() != nil || workflow.ActiveService() != nil || functions.ActiveService() != nil {
		t.Fatal("active service is set")
	}
	cases := []struct {
		name string
		call func() error
	}{
		{"pipeline.get", func() error {
			_, err := execPipeline(context.Background(), Identity{}, "get", map[string]any{"name": "daily"})
			return err
		}},
		{"workflow.get", func() error {
			_, err := execWorkflow(context.Background(), "get", map[string]any{"name": "daily"})
			return err
		}},
		{"function.get", func() error {
			_, err := execFunction(context.Background(), "get", map[string]any{"name": "daily"})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var te *types.Error
			require.ErrorAs(t, tc.call(), &te)
			assert.Equal(t, types.ErrUnavailable, te.Kind)
		})
	}
}
