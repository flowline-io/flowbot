package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
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
