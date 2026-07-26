package life

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
)

//go:embed seed/equipment.json
var equipmentSeedJSON []byte

//go:embed seed/loot_tables.json
var lootSeedJSON []byte

type equipmentSeed struct {
	Flag                string         `json:"flag"`
	Name                string         `json:"name"`
	Rarity              string         `json:"rarity"`
	SlotType            string         `json:"slot_type"`
	StatBuffs           map[string]any `json:"stat_buffs"`
	AIUnlockedPrivilege map[string]any `json:"ai_unlocked_privilege"`
	AILoreText          string         `json:"ai_lore_text"`
}

type lootSeed struct {
	DropTier       string   `json:"drop_tier"`
	BaseDropChance float64  `json:"base_drop_chance"`
	ItemPoolFlags  []string `json:"item_pool_flags"`
}

func seedCatalog(ctx context.Context, ls *store.LifeStore) error {
	var equipment []equipmentSeed
	if err := sonic.Unmarshal(equipmentSeedJSON, &equipment); err != nil {
		return fmt.Errorf("parse equipment seed: %w", err)
	}
	for _, e := range equipment {
		if _, err := ls.UpsertEquipment(ctx, e.Flag, e.Name, e.Rarity, e.SlotType, e.AILoreText, e.StatBuffs, e.AIUnlockedPrivilege); err != nil {
			return err
		}
	}
	var tables []lootSeed
	if err := sonic.Unmarshal(lootSeedJSON, &tables); err != nil {
		return fmt.Errorf("parse loot seed: %w", err)
	}
	for _, t := range tables {
		if err := ls.UpsertLootTable(ctx, t.DropTier, t.BaseDropChance, t.ItemPoolFlags); err != nil {
			return err
		}
	}
	return nil
}
