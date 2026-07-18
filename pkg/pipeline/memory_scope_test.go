package pipeline

import (
	"maps"
	"testing"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestInjectAgentRunMemoryScope(t *testing.T) {
	tests := []struct {
		name         string
		step         Step
		existing     map[string]any
		pipelineName string
		want         string
		wantSet      bool
	}{
		{
			name:         "injects for agent run",
			step:         Step{Capability: hub.CapAgent, Operation: capability.OpAgentRun},
			existing:     map[string]any{"prompt": "hi"},
			pipelineName: "sync-bookmarks",
			want:         "sync-bookmarks",
			wantSet:      true,
		},
		{
			name:         "keeps explicit scope",
			step:         Step{Capability: hub.CapAgent, Operation: capability.OpAgentRun},
			existing:     map[string]any{"memory_scope": "custom"},
			pipelineName: "sync-bookmarks",
			want:         "custom",
			wantSet:      true,
		},
		{
			name:         "skips non agent step",
			step:         Step{Capability: hub.CapKarakeep, Operation: capability.OpBookmarkList},
			existing:     map[string]any{},
			pipelineName: "sync-bookmarks",
			wantSet:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{}
			maps.Copy(params, tt.existing)
			injectAgentRunMemoryScope(tt.step, params, tt.pipelineName)
			raw, ok := params["memory_scope"]
			if !tt.wantSet {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Equal(t, tt.want, raw)
		})
	}
}

func TestInjectEventUID(t *testing.T) {
	tests := []struct {
		name     string
		step     Step
		existing map[string]any
		eventUID string
		want     string
		wantSet  bool
	}{
		{
			name:     "injects into agent run from event uid",
			step:     Step{Capability: hub.CapAgent, Operation: capability.OpAgentRun},
			existing: map[string]any{"prompt": "hi"},
			eventUID: "user-admin",
			want:     "user-admin",
			wantSet:  true,
		},
		{
			name:     "injects into notify send from event uid",
			step:     Step{Capability: hub.CapNotify, Operation: "send"},
			existing: map[string]any{"template_id": "cron.output", "channels": []string{"testing"}},
			eventUID: "user-admin",
			want:     "user-admin",
			wantSet:  true,
		},
		{
			name:     "keeps explicit uid on notify send",
			step:     Step{Capability: hub.CapNotify, Operation: "send"},
			existing: map[string]any{"uid": "user-custom"},
			eventUID: "user-admin",
			want:     "user-custom",
			wantSet:  true,
		},
		{
			name:     "skips unrelated step",
			step:     Step{Capability: hub.CapKarakeep, Operation: capability.OpBookmarkList},
			existing: map[string]any{},
			eventUID: "user-admin",
			wantSet:  false,
		},
		{
			name:     "skips when event uid empty",
			step:     Step{Capability: hub.CapNotify, Operation: "send"},
			existing: map[string]any{},
			eventUID: "",
			wantSet:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{}
			maps.Copy(params, tt.existing)
			rc := NewRenderContext(types.DataEvent{UID: tt.eventUID})
			injectEventUID(tt.step, params, rc)
			raw, ok := params["uid"]
			if !tt.wantSet {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Equal(t, tt.want, raw)
		})
	}
}

func TestApplyDefinitionUID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		defUID   string
		eventUID string
		wantUID  string
	}{
		{
			name:     "applies definition uid when event empty",
			defUID:   "user-admin",
			eventUID: "",
			wantUID:  "user-admin",
		},
		{
			name:     "keeps event uid over definition",
			defUID:   "user-admin",
			eventUID: "user-event",
			wantUID:  "user-event",
		},
		{
			name:     "leaves empty when both missing",
			defUID:   "",
			eventUID: "",
			wantUID:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			event := types.DataEvent{UID: tt.eventUID}
			applyDefinitionUID(Definition{UID: tt.defUID}, &event)
			assert.Equal(t, tt.wantUID, event.UID)
		})
	}
}
