package hooks_test

import (
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainTransformContext(t *testing.T) {
	t.Parallel()

	inner := func(messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
		return append(messages, userMessage("inner")), nil
	}
	outer := func(messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
		return append(messages, userMessage("outer")), nil
	}

	tests := []struct {
		name    string
		inner   msg.TransformContextFn
		outer   msg.TransformContextFn
		wantLen int
		wantErr bool
	}{
		{name: "chains inner then outer", inner: inner, outer: outer, wantLen: 3},
		{name: "nil outer returns inner", inner: inner, outer: nil, wantLen: 2},
		{name: "nil inner returns outer", inner: nil, outer: outer, wantLen: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chained := hooks.ChainTransformContext(tt.inner, tt.outer)
			require.NotNil(t, chained)
			got, err := chained([]msg.AgentMessage{userMessage("seed")})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestChainBeforeToolCall(t *testing.T) {
	t.Parallel()

	blockInner := func(msg.BeforeToolContext) (*msg.BeforeToolResult, error) {
		return &msg.BeforeToolResult{Block: true, Reason: "inner"}, nil
	}
	passInner := func(msg.BeforeToolContext) (*msg.BeforeToolResult, error) {
		return nil, nil
	}
	blockOuter := func(msg.BeforeToolContext) (*msg.BeforeToolResult, error) {
		return &msg.BeforeToolResult{Block: true, Reason: "outer"}, nil
	}
	failInner := func(msg.BeforeToolContext) (*msg.BeforeToolResult, error) {
		return nil, errors.New("inner failed")
	}

	tests := []struct {
		name       string
		inner      msg.BeforeToolCallFn
		outer      msg.BeforeToolCallFn
		wantBlock  bool
		wantReason string
		wantErr    bool
	}{
		{name: "inner block stops outer", inner: blockInner, outer: blockOuter, wantBlock: true, wantReason: "inner"},
		{name: "outer blocks when inner passes", inner: passInner, outer: blockOuter, wantBlock: true, wantReason: "outer"},
		{name: "inner error propagates", inner: failInner, outer: blockOuter, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chained := hooks.ChainBeforeToolCall(tt.inner, tt.outer)
			result, err := chained(msg.BeforeToolContext{})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantBlock {
				require.NotNil(t, result)
				assert.True(t, result.Block)
				assert.Equal(t, tt.wantReason, result.Reason)
				return
			}
			assert.Nil(t, result)
		})
	}
}

func TestChainAfterToolCall(t *testing.T) {
	t.Parallel()

	isErr := true
	patchInner := func(_ msg.AfterToolContext) (*msg.AfterToolResult, error) {
		return &msg.AfterToolResult{Parts: []msg.ContentPart{msg.TextPart{Text: "patched"}}}, nil
	}
	terminateInner := func(_ msg.AfterToolContext) (*msg.AfterToolResult, error) {
		return &msg.AfterToolResult{Terminate: true}, nil
	}
	patchOuter := func(_ msg.AfterToolContext) (*msg.AfterToolResult, error) {
		return &msg.AfterToolResult{IsError: &isErr}, nil
	}
	failOuter := func(msg.AfterToolContext) (*msg.AfterToolResult, error) {
		return nil, errors.New("outer failed")
	}

	tests := []struct {
		name          string
		inner         msg.AfterToolCallFn
		outer         msg.AfterToolCallFn
		wantErr       bool
		wantError     bool
		wantTerminate bool
	}{
		{name: "outer patch applies after inner", inner: patchInner, outer: patchOuter, wantError: true},
		{name: "inner terminate preserved on outer", inner: terminateInner, outer: patchOuter, wantTerminate: true},
		{name: "outer error propagates", inner: patchInner, outer: failOuter, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chained := hooks.ChainAfterToolCall(tt.inner, tt.outer)
			ctx := msg.AfterToolContext{Result: msg.ToolResultMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "raw"}}}}
			result, err := chained(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.wantError {
				require.NotNil(t, result.IsError)
				assert.True(t, *result.IsError)
			}
			if tt.wantTerminate {
				assert.True(t, result.Terminate)
			}
		})
	}
}

func TestChainPrepareNextTurn(t *testing.T) {
	t.Parallel()

	inner := func(_ msg.TurnContext) (*msg.TurnUpdate, error) {
		return &msg.TurnUpdate{ModelName: "inner-model"}, nil
	}
	outer := func(ctx msg.TurnContext) (*msg.TurnUpdate, error) {
		assert.Equal(t, "inner-model", ctx.Context.ModelName)
		return &msg.TurnUpdate{ModelName: "outer-model"}, nil
	}
	failInner := func(msg.TurnContext) (*msg.TurnUpdate, error) {
		return nil, errors.New("turn failed")
	}

	tests := []struct {
		name      string
		inner     msg.PrepareNextTurnFn
		outer     msg.PrepareNextTurnFn
		wantModel string
		wantErr   bool
	}{
		{name: "inner context flows to outer", inner: inner, outer: outer, wantModel: "outer-model"},
		{name: "nil outer returns inner result", inner: inner, outer: nil, wantModel: "inner-model"},
		{name: "inner error stops chain", inner: failInner, outer: outer, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chained := hooks.ChainPrepareNextTurn(tt.inner, tt.outer)
			ctx := msg.TurnContext{Context: &msg.Context{}}
			update, err := chained(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, update)
			assert.Equal(t, tt.wantModel, update.ModelName)
		})
	}
}

func TestMergeHookFields(t *testing.T) {
	t.Parallel()

	transform := func([]msg.AgentMessage) ([]msg.AgentMessage, error) { return nil, nil }

	tests := []struct {
		name string
		dst  *msg.Config
		src  *msg.Config
		want bool
	}{
		{name: "copies hook callbacks", dst: &msg.Config{}, src: &msg.Config{TransformContext: transform}, want: true},
		{name: "nil dst is no-op", dst: nil, src: &msg.Config{TransformContext: transform}, want: false},
		{name: "nil src is no-op", dst: &msg.Config{}, src: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hooks.MergeHookFields(tt.dst, tt.src)
			if !tt.want {
				return
			}
			require.NotNil(t, tt.dst.TransformContext)
		})
	}
}
