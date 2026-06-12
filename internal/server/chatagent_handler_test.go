package server

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunChatAgentContextTimeout(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "not canceled when watermill parent is canceled"},
		{name: "uses configured run timeout deadline"},
		{name: "deadline expires after timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := config.App.ChatAgent
			config.App.ChatAgent = config.ChatAgentConfig{RunTimeout: 20 * time.Millisecond}
			t.Cleanup(func() { config.App.ChatAgent = orig })

			parent, parentCancel := context.WithCancel(context.Background())
			parentCancel()
			require.ErrorIs(t, parent.Err(), context.Canceled)

			runTimeout := chatagent.RunTimeout()
			ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
			defer cancel()

			require.NoError(t, ctx.Err())
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			assert.LessOrEqual(t, time.Until(deadline), runTimeout)

			time.Sleep(runTimeout + 10*time.Millisecond)
			require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
		})
	}
}
