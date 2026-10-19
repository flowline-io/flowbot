package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
