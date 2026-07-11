package pipeline

import (
	"maps"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
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
			step:         Step{Capability: hub.CapAgent, Operation: ability.OpAgentRun},
			existing:     map[string]any{"prompt": "hi"},
			pipelineName: "sync-bookmarks",
			want:         "sync-bookmarks",
			wantSet:      true,
		},
		{
			name:         "keeps explicit scope",
			step:         Step{Capability: hub.CapAgent, Operation: ability.OpAgentRun},
			existing:     map[string]any{"memory_scope": "custom"},
			pipelineName: "sync-bookmarks",
			want:         "custom",
			wantSet:      true,
		},
		{
			name:         "skips non agent step",
			step:         Step{Capability: hub.CapBookmark, Operation: ability.OpBookmarkList},
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

func TestInjectAgentRunUID(t *testing.T) {
	tests := []struct {
		name     string
		step     Step
		existing map[string]any
		eventUID string
		want     string
		wantSet  bool
	}{
		{
			name:     "injects from event uid",
			step:     Step{Capability: hub.CapAgent, Operation: ability.OpAgentRun},
			existing: map[string]any{"prompt": "hi"},
			eventUID: "user-admin",
			want:     "user-admin",
			wantSet:  true,
		},
		{
			name:     "keeps explicit uid",
			step:     Step{Capability: hub.CapAgent, Operation: ability.OpAgentRun},
			existing: map[string]any{"uid": "user-custom"},
			eventUID: "user-admin",
			want:     "user-custom",
			wantSet:  true,
		},
		{
			name:     "skips non agent step",
			step:     Step{Capability: hub.CapBookmark, Operation: ability.OpBookmarkList},
			existing: map[string]any{},
			eventUID: "user-admin",
			wantSet:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{}
			maps.Copy(params, tt.existing)
			rc := NewRenderContext(types.DataEvent{UID: tt.eventUID})
			injectAgentRunUID(tt.step, params, rc)
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
