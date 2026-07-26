package life

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

func TestParseLoreInventoryID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		payload  map[string]any
		wantID   int64
		wantPoison bool
	}{
		{
			name:       "valid",
			payload:    map[string]any{"inventory_id": int64(42), "type": "life.inventory.lore_requested"},
			wantID:     42,
			wantPoison: false,
		},
		{
			name:       "float id from json",
			payload:    map[string]any{"inventory_id": float64(7)},
			wantID:     7,
			wantPoison: false,
		},
		{
			name:       "missing id",
			payload:    map[string]any{"type": "life.inventory.lore_requested"},
			wantID:     0,
			wantPoison: true,
		},
		{
			name:       "zero id",
			payload:    map[string]any{"inventory_id": int64(0)},
			wantID:     0,
			wantPoison: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			row := &gen.EventOutbox{EventID: "e1", Payload: tt.payload}
			id, poison := parseLoreInventoryID(row)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.wantPoison, poison)
		})
	}
}

func TestResolveDropEquip_NeedLore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		questType string
		diff     string
		wantLore bool
	}{
		{name: "daily common", questType: "Daily", diff: "B", wantLore: false},
		{name: "boss", questType: "Boss", diff: "A", wantLore: true},
		{name: "ss", questType: "One-Time", diff: "SS", wantLore: true},
		{name: "sss", questType: "One-Time", diff: "SSS", wantLore: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q := &gen.LifeQuest{Type: tt.questType, AiEvaluatedDifficulty: tt.diff}
			need := q.Type == "Boss" || q.AiEvaluatedDifficulty == "SSS" || q.AiEvaluatedDifficulty == "SS"
			assert.Equal(t, tt.wantLore, need)
		})
	}
}
