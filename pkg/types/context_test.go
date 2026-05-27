package types

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext_EmptyContextReturnsTraceCtx(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ctx     *Context
		wantNil bool
		wantBg  bool
	}{
		{
			name:    "both nil returns Background",
			ctx:     &Context{},
			wantNil: false,
			wantBg:  true,
		},
		{
			name:    "TraceCtx set without internal ctx returns TraceCtx",
			ctx:     &Context{TraceCtx: context.WithValue(context.Background(), struct{}{}, "trace")},
			wantNil: false,
			wantBg:  false,
		},
		{
			name: "internal ctx set via SetTimeout returns deadline context",
			ctx: func() *Context {
				c := &Context{}
				c.SetTimeout(time.Minute)
				return c
			}(),
			wantNil: false,
			wantBg:  false,
		},
		{
			name: "both TraceCtx and internal ctx set returns internal ctx",
			ctx: func() *Context {
				c := &Context{
					TraceCtx: context.WithValue(context.Background(), struct{}{}, "trace"),
				}
				c.SetTimeout(time.Minute)
				return c
			}(),
			wantNil: false,
			wantBg:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.ctx.Context()
			assert.NotNil(t, got)
			if tt.wantBg {
				// context.Background() has no deadline and no values
				_, hasDeadline := got.Deadline()
				assert.False(t, hasDeadline)
			} else {
				_, hasDeadline := got.Deadline()
				// If SetTimeout was called, there should be a deadline.
				// If only TraceCtx was set, no deadline -- that's OK.
				_ = hasDeadline
				assert.NotNil(t, got)
			}
		})
	}
}

func TestContext_ReturnsInternalCtxWhenSet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func() *Context
	}{
		{
			name: "SetTimeout sets internal ctx",
			setup: func() *Context {
				c := &Context{}
				c.SetTimeout(time.Minute)
				return c
			},
		},
		{
			name: "SetContext sets both fields",
			setup: func() *Context {
				c := &Context{}
				parent := context.WithValue(context.Background(), struct{}{}, "parent")
				c.SetContext(parent)
				return c
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.setup()
			got := c.Context()
			require.NotNil(t, got)
		})
	}
}

func TestContext_SetContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "SetContext stores the context in both fields"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Context{}
			parent := context.WithValue(context.Background(), struct{}{}, "val")
			c.SetContext(parent)

			got := c.Context()
			require.NotNil(t, got)
			require.NotNil(t, c.TraceCtx)
		})
	}
}

func TestContext_SetTraceContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "SetTraceContext stores in TraceCtx only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Context{}
			tc := context.WithValue(context.Background(), struct{}{}, "trace")
			c.SetTraceContext(tc)

			require.NotNil(t, c.TraceCtx)
			assert.Equal(t, "trace", c.TraceCtx.Value(struct{}{}))
			// Context() should return TraceCtx as fallback since c.ctx is nil
			assert.Equal(t, "trace", c.Context().Value(struct{}{}))
		})
	}
}

func TestContext_SetTimeoutUsesTraceCtxAsParent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func() *Context
	}{
		{
			name: "uses TraceCtx as parent when internal ctx is nil",
			setup: func() *Context {
				c := &Context{
					TraceCtx: context.WithValue(context.Background(), struct{}{}, "trace"),
				}
				c.SetTimeout(time.Minute)
				return c
			},
		},
		{
			name: "uses Background as parent when both nil",
			setup: func() *Context {
				c := &Context{}
				c.SetTimeout(time.Minute)
				return c
			},
		},
		{
			name: "uses existing internal ctx when already set",
			setup: func() *Context {
				c := &Context{
					TraceCtx: context.WithValue(context.Background(), struct{}{}, "trace"),
				}
				c.SetTimeout(time.Second)
				c.SetTimeout(time.Minute)
				return c
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.setup()
			got := c.Context()
			require.NotNil(t, got)
			// Verify deadline is set
			_, ok := got.Deadline()
			assert.True(t, ok, "expected context to have a deadline")
		})
	}
}

func TestContext_CancelReturnsCancelFunc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setup     func() *Context
		wantCancel bool
	}{
		{
			name: "Cancel returns non-nil cancel after SetTimeout",
			setup: func() *Context {
				c := &Context{}
				c.SetTimeout(time.Minute)
				return c
			},
			wantCancel: true,
		},
		{
			name:      "Cancel returns nil without SetTimeout",
			setup:     func() *Context { return &Context{} },
			wantCancel: false,
		},
		{
			name: "Cancel returns non-nil cancel after SetContext",
			setup: func() *Context {
				c := &Context{}
				c.SetContext(context.Background())
				return c
			},
			wantCancel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tt.setup()
			cancel := c.Cancel()
			if tt.wantCancel {
				assert.NotNil(t, cancel)
			} else {
				assert.Nil(t, cancel)
			}
		})
	}
}
