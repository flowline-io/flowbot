package life

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/module"
)

func TestModuleName(t *testing.T) {
	assert.Equal(t, "life", Name)
	assert.Implements(t, (*module.Handler)(nil), &handler)
}

func TestInit_Disabled(t *testing.T) {
	handler = moduleHandler{}
	lifeService = nil
	serviceListeners = nil

	err := handler.Init(json.RawMessage(`{"enabled":false}`))
	require.NoError(t, err)
	assert.False(t, handler.IsReady())
}

func TestInit_InvalidJSON(t *testing.T) {
	handler = moduleHandler{}
	err := handler.Init(json.RawMessage(`{invalid`))
	require.Error(t, err)
}

func TestInit_AlreadyInitialized(t *testing.T) {
	handler = moduleHandler{initialized: true}
	err := handler.Init(json.RawMessage(`{"enabled":true}`))
	require.Error(t, err)
}

func TestOnService_NotifiesListeners(t *testing.T) {
	handler = moduleHandler{}
	lifeService = nil
	serviceListeners = nil

	var got *Service
	OnService(func(s *Service) { got = s })
	assert.Nil(t, got)

	svc := &Service{}
	setActiveService(svc)
	assert.Same(t, svc, got)
	assert.Same(t, svc, ActiveService())
}

func TestSeedEquipmentJSON(t *testing.T) {
	t.Parallel()
	var equipment []equipmentSeed
	require.NoError(t, sonic.Unmarshal(equipmentSeedJSON, &equipment))
	require.GreaterOrEqual(t, len(equipment), 365)

	flags := make(map[string]struct{}, len(equipment))
	names := make(map[string]struct{}, len(equipment))
	slots := map[string]struct{}{
		"Head": {}, "Weapon": {}, "Armor": {}, "Shoes": {}, "Accessory": {}, "Artifact": {},
	}
	rarities := map[string]struct{}{
		"Common": {}, "Uncommon": {}, "Rare": {}, "Epic": {}, "Legendary": {}, "Mythic": {},
	}
	for _, e := range equipment {
		assert.NotEmpty(t, e.Flag)
		assert.NotEmpty(t, e.Name)
		assert.NotEmpty(t, e.AILoreText)
		_, okSlot := slots[e.SlotType]
		assert.True(t, okSlot, "slot %s", e.SlotType)
		_, okRarity := rarities[e.Rarity]
		assert.True(t, okRarity, "rarity %s", e.Rarity)
		_, dupFlag := flags[e.Flag]
		assert.False(t, dupFlag, "duplicate flag %s", e.Flag)
		_, dupName := names[e.Name]
		assert.False(t, dupName, "duplicate name %s", e.Name)
		flags[e.Flag] = struct{}{}
		names[e.Name] = struct{}{}
	}

	var tables []lootSeed
	require.NoError(t, sonic.Unmarshal(lootSeedJSON, &tables))
	require.Len(t, tables, 5)
	pool := map[string]struct{}{}
	for _, tb := range tables {
		assert.NotEmpty(t, tb.DropTier)
		assert.Greater(t, tb.BaseDropChance, 0.0)
		require.NotEmpty(t, tb.ItemPoolFlags)
		for _, f := range tb.ItemPoolFlags {
			_, ok := flags[f]
			assert.True(t, ok, "loot flag missing from equipment: %s", f)
			pool[f] = struct{}{}
		}
	}
	assert.Len(t, flags, len(pool), "every equipment flag must appear in at least one loot pool")
}
