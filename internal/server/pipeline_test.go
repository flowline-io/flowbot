package server

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/pipeline"
)

func TestNewPipelineStepCallback(t *testing.T) {
	tests := []struct {
		name    string
		client  *redis.Client
		wantNil bool
	}{
		{name: "nil client returns nil", client: nil, wantNil: true},
		{name: "redis client returns callback", client: redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}), wantNil: false},
		{name: "miniredis client returns callback", client: func() *redis.Client {
			mr := miniredis.RunT(t)
			return redis.NewClient(&redis.Options{Addr: mr.Addr()})
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewPipelineStepCallback(tt.client)
			if tt.wantNil {
				assert.Nil(t, cb)
				return
			}
			require.NotNil(t, cb)
		})
	}
}

func TestPipelineStepCallbackPublish(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cb, ok := NewPipelineStepCallback(client).(*pipelineStepCallback)
	require.True(t, ok)

	tests := []struct {
		name   string
		invoke func()
		status string
	}{
		{
			name: "run start publishes start event",
			invoke: func() {
				cb.OnRunStart(context.Background(), 42, "pipe-a", "trigger", 3, []string{"step1"})
			},
			status: "start",
		},
		{
			name: "step done publishes output",
			invoke: func() {
				cb.OnStepDone(context.Background(), 43, "pipe-a", 0, "step1", map[string]any{"ok": true}, 12)
			},
			status: "done",
		},
		{
			name: "step error publishes error text",
			invoke: func() {
				cb.OnStepError(context.Background(), 44, "pipe-a", 1, "step2", assert.AnError, 5)
			},
			status: "error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.invoke()
			time.Sleep(50 * time.Millisecond)

			runID := int64(42)
			switch tt.status {
			case "done":
				runID = 43
			case "error":
				runID = 44
			}
			entries, err := client.XRange(context.Background(), pipeline.StreamName(runID), "-", "+").Result()
			require.NoError(t, err)
			require.NotEmpty(t, entries)

			raw, ok := entries[0].Values["data"].(string)
			require.True(t, ok)
			var evt pipeline.StepProgressEvent
			require.NoError(t, sonic.Unmarshal([]byte(raw), &evt))
			assert.Equal(t, tt.status, evt.Status)
		})
	}
}

func TestPipelineStepCallbackRunComplete(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cb, ok := NewPipelineStepCallback(client).(*pipelineStepCallback)
	require.True(t, ok)

	tests := []struct {
		name   string
		failed bool
		status string
	}{
		{name: "complete run", failed: false, status: "complete"},
		{name: "failed run", failed: true, status: "failed"},
		{name: "failed run with message", failed: true, status: "failed"},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID := int64(200 + i)
			cb.OnRunComplete(context.Background(), runID, "pipe-b", 99, tt.failed, "boom")
			time.Sleep(50 * time.Millisecond)
			entries, err := client.XRange(context.Background(), pipeline.StreamName(runID), "-", "+").Result()
			require.NoError(t, err)
			require.NotEmpty(t, entries)
			raw, ok := entries[0].Values["data"].(string)
			require.True(t, ok)
			var evt pipeline.StepProgressEvent
			require.NoError(t, sonic.Unmarshal([]byte(raw), &evt))
			assert.Equal(t, tt.status, evt.Status)
		})
	}
}

func TestBuildPollingState_NilDatabase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "nil database creates polling state with nil persistence"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state := buildPollingState()
			assert.NotNil(t, state)
		})
	}
}

// verify pollingPersistenceAdapter implements capability.Persistence.
var _ capability.Persistence = (*pollingPersistenceAdapter)(nil)
