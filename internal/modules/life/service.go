package life

import (
	"context"
	"fmt"
	"maps"
	"math/rand"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	caplife "github.com/flowline-io/flowbot/pkg/capability"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
)

// Service orchestrates Life use cases for the web UI.
type Service struct {
	store *store.LifeStore
	rng   *rand.Rand
}

// NewService constructs a Life service.
func NewService(ls *store.LifeStore) *Service {
	return &Service{
		store: ls,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// EnsureProfile creates the profile graph for a user if needed.
func (s *Service) EnsureProfile(ctx context.Context, userID, nickname, defaultClass string) (*gen.LifeProfile, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("life: service unavailable")
	}
	p, err := s.store.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p != nil {
		if err := s.ensureDefaultCharacteristics(ctx, p.ID); err != nil {
			return nil, err
		}
		return p, nil
	}
	if defaultClass == "" {
		defaultClass = "Architect"
	}
	p, err = s.store.CreateProfile(ctx, userID, nickname, defaultClass)
	if err != nil {
		return nil, err
	}
	if err := s.ensureDefaultCharacteristics(ctx, p.ID); err != nil {
		return nil, err
	}
	if err := s.store.EnsureAIContext(ctx, p.ID); err != nil {
		return nil, err
	}
	if _, err := s.store.EnsureEquippedSlots(ctx, p.ID); err != nil {
		return nil, err
	}
	flog.InfoFields("life: profile created", map[string]any{
		"uid":        userID,
		"profile_id": p.ID,
		"class":      defaultClass,
	})
	return p, nil
}

func (s *Service) ensureDefaultCharacteristics(ctx context.Context, profileID int64) error {
	existing, err := s.store.ListCharacteristics(ctx, profileID)
	if err != nil {
		return err
	}
	have := map[string]struct{}{}
	for _, c := range existing {
		have[c.Code] = struct{}{}
	}
	for _, def := range pkglife.DefaultCharacteristics {
		if _, ok := have[def.Code]; ok {
			continue
		}
		if _, err := s.store.CreateCharacteristic(ctx, profileID, def.Code, def.Name); err != nil {
			return err
		}
		flog.InfoFields("life: characteristic backfilled", map[string]any{
			"profile_id": profileID,
			"code":       def.Code,
		})
	}
	return nil
}

// CharacterView is the character page model.
type CharacterView struct {
	Profile         *gen.LifeProfile
	Characteristics []*gen.LifeCharacteristic
	Skills          []*gen.LifeSkill
	Slots           *gen.LifeEquippedSlots
	Buffs           pkglife.BuffTotals
	Goals           []*gen.LifeGoal
	PlanTree        []*PlanNodeView
}

// GetCharacter loads character aggregates.
func (s *Service) GetCharacter(ctx context.Context, userID string) (*CharacterView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	chars, err := s.store.ListCharacteristics(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	skills, err := s.store.ListSkills(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	slots, err := s.store.GetEquippedSlots(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	buffs, err := s.equippedBuffs(ctx, slots)
	if err != nil {
		return nil, err
	}
	goals, err := s.store.ListGoals(ctx, p.ID, "")
	if err != nil {
		return nil, err
	}
	planTree, err := s.ListPlanTree(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &CharacterView{
		Profile: p, Characteristics: chars, Skills: skills, Slots: slots, Buffs: buffs, Goals: goals, PlanTree: planTree,
	}, nil
}

// SetClassType updates class.
func (s *Service) SetClassType(ctx context.Context, userID, classType string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	if err := s.store.UpdateProfileClass(ctx, p.ID, classType); err != nil {
		return err
	}
	flog.InfoFields("life: class updated", map[string]any{
		"uid":        userID,
		"profile_id": p.ID,
		"class":      classType,
	})
	return nil
}

// CreateQuestFromPrompt evaluates and persists a quest.
// goalFlag is optional; when set, the quest is linked to that active goal.
func (s *Service) CreateQuestFromPrompt(ctx context.Context, userID, prompt, goalFlag string) (*gen.LifeQuest, *lifecap.QuestEvaluation, float64, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, nil, 0, err
	}
	goalID, goalTitles, err := s.resolveGoalBinding(ctx, p.ID, goalFlag)
	if err != nil {
		return nil, nil, 0, err
	}
	ev, err := s.evaluateQuestPrompt(ctx, p.ID, prompt, goalTitles)
	if err != nil {
		return nil, nil, 0, err
	}
	skill, err := s.resolveSkillForEvaluation(ctx, p.ID, ev)
	if err != nil {
		return nil, nil, 0, err
	}
	q, err := s.store.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         p.ID,
		SkillID:               skill.ID,
		GoalID:                goalID,
		Title:                 ev.Title,
		Prompt:                strings.TrimSpace(prompt),
		Type:                  ev.QuestType,
		AiEvaluatedDifficulty: ev.Difficulty,
		AiEvaluatedFear:       ev.Fear,
		BaseExpReward:         ev.BaseExp,
		BaseGoldReward:        ev.BaseGold,
		DropTier:              ev.DropTier,
	})
	if err != nil {
		return nil, nil, 0, err
	}
	chance, err := s.dropChanceForQuest(ctx, p, q)
	if err != nil {
		flog.WarnFields("life: drop chance preview failed", map[string]any{
			"profile_id": p.ID, "quest_flag": q.Flag, "error": err.Error(),
		})
		chance = 0
	}
	flog.InfoFields("life: quest created", map[string]any{
		"uid":         userID,
		"profile_id":  p.ID,
		"quest_flag":  q.Flag,
		"title":       q.Title,
		"prompt":      q.Prompt,
		"type":        q.Type,
		"difficulty":  q.AiEvaluatedDifficulty,
		"drop_tier":   q.DropTier,
		"drop_chance": chance,
		"skill":       ev.SkillName,
		"base_exp":    q.BaseExpReward,
		"base_gold":   q.BaseGoldReward,
	})
	return q, ev, chance, nil
}

func (s *Service) resolveGoalBinding(ctx context.Context, profileID int64, goalFlag string) (*int64, []string, error) {
	goals, err := s.store.ListGoals(ctx, profileID, "Active")
	if err != nil {
		return nil, nil, err
	}
	titles := make([]string, 0, len(goals))
	for _, g := range goals {
		titles = append(titles, g.Title)
	}
	if goalFlag == "" {
		return nil, titles, nil
	}
	g, err := s.store.GetGoalByFlag(ctx, profileID, goalFlag)
	if err != nil {
		return nil, nil, err
	}
	if g == nil || g.Status != "Active" {
		return nil, nil, fmt.Errorf("life: goal not found")
	}
	return &g.ID, titles, nil
}

func (s *Service) evaluateQuestPrompt(ctx context.Context, profileID int64, prompt string, goalTitles []string) (*lifecap.QuestEvaluation, error) {
	if err := s.store.EnsureAIContext(ctx, profileID); err != nil {
		return nil, err
	}
	aiCtx, err := s.store.GetAIContext(ctx, profileID)
	if err != nil {
		return nil, err
	}
	slots, err := s.store.GetEquippedSlots(ctx, profileID)
	if err != nil {
		return nil, err
	}
	privs, err := s.equippedPrivileges(ctx, slots)
	if err != nil {
		return nil, err
	}
	params := map[string]any{
		"prompt":          prompt,
		"active_goals":    goalTitles,
		"privileges":      privs,
		"breakdown_depth": privilegeDepth(privs),
	}
	if aiCtx != nil {
		params["ai_personality"] = aiCtx.AiDmPersonality
		params["completion_rate"] = aiCtx.HistoricalCompletionRate
		params["mood"] = aiCtx.RecentMoodAndBurnout
	}
	res, err := caplife.Invoke(ctx, hub.CapLife, lifecap.OpEvaluateQuest, params)
	if err != nil {
		return nil, fmt.Errorf("life: evaluate quest: %w", err)
	}
	return decodeQuestEvaluation(res.Data)
}

func decodeQuestEvaluation(data any) (*lifecap.QuestEvaluation, error) {
	if ev, ok := data.(*lifecap.QuestEvaluation); ok {
		return ev, nil
	}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("life: unexpected evaluate payload")
	}
	ev := &lifecap.QuestEvaluation{}
	if err := sonic.Unmarshal(raw, ev); err != nil {
		return nil, fmt.Errorf("life: decode evaluate payload: %w", err)
	}
	return ev, nil
}

func decodeQuestAdjudication(data any) (*lifecap.QuestAdjudication, error) {
	if ruling, ok := data.(*lifecap.QuestAdjudication); ok {
		return ruling, nil
	}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("life: unexpected adjudication payload")
	}
	ruling := &lifecap.QuestAdjudication{}
	if err := sonic.Unmarshal(raw, ruling); err != nil {
		return nil, fmt.Errorf("life: decode adjudication payload: %w", err)
	}
	return ruling, nil
}

func (s *Service) resolveSkillForEvaluation(ctx context.Context, profileID int64, ev *lifecap.QuestEvaluation) (*gen.LifeSkill, error) {
	chars, err := s.store.ListCharacteristics(ctx, profileID)
	if err != nil {
		return nil, err
	}
	var charID int64
	for _, c := range chars {
		if c.Code == ev.StatCode {
			charID = c.ID
			break
		}
	}
	if charID == 0 && len(chars) > 0 {
		charID = chars[0].ID
	}
	skill, err := s.store.GetSkillByName(ctx, profileID, ev.SkillName)
	if err != nil {
		return nil, err
	}
	if skill != nil {
		return skill, nil
	}
	return s.store.CreateSkill(ctx, profileID, charID, ev.SkillName, 0.5)
}

// CreateGoal creates an Active PARA goal.
func (s *Service) CreateGoal(ctx context.Context, userID, title, category string) (*gen.LifeGoal, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("life: goal title required")
	}
	switch category {
	case "Project", "Area", "Resource":
	default:
		category = "Project"
	}
	g, err := s.store.CreateGoal(ctx, p.ID, title, category)
	if err != nil {
		return nil, err
	}
	flog.InfoFields("life: goal created", map[string]any{
		"uid": userID, "profile_id": p.ID, "goal_flag": g.Flag, "category": category, "title": title,
	})
	return g, nil
}

// SetGoalStatus updates a goal status by flag.
func (s *Service) SetGoalStatus(ctx context.Context, userID, goalFlag, status string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	g, err := s.store.GetGoalByFlag(ctx, p.ID, goalFlag)
	if err != nil || g == nil {
		return fmt.Errorf("life: goal not found")
	}
	switch status {
	case "Active", "Paused", "Completed":
	default:
		return fmt.Errorf("life: invalid goal status")
	}
	return s.store.UpdateGoalStatus(ctx, g.ID, status)
}

// UpdateGoal updates title and category for a goal by flag.
func (s *Service) UpdateGoal(ctx context.Context, userID, goalFlag, title, category string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	g, err := s.store.GetGoalByFlag(ctx, p.ID, goalFlag)
	if err != nil || g == nil {
		return fmt.Errorf("life: goal not found")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("life: goal title required")
	}
	switch category {
	case "Project", "Area", "Resource":
	default:
		category = g.Category
	}
	if err := s.store.UpdateGoal(ctx, g.ID, title, category); err != nil {
		return err
	}
	flog.InfoFields("life: goal updated", map[string]any{
		"uid": userID, "profile_id": p.ID, "goal_flag": goalFlag, "category": category,
	})
	return nil
}

// DeleteGoal removes a goal by flag.
func (s *Service) DeleteGoal(ctx context.Context, userID, goalFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	g, err := s.store.GetGoalByFlag(ctx, p.ID, goalFlag)
	if err != nil || g == nil {
		return fmt.Errorf("life: goal not found")
	}
	if err := s.store.DeleteGoal(ctx, g.ID); err != nil {
		return err
	}
	flog.InfoFields("life: goal deleted", map[string]any{
		"uid": userID, "profile_id": p.ID, "goal_flag": goalFlag,
	})
	return nil
}

// FailQuest marks a pending quest failed and applies 24h equipment rust.
func (s *Service) FailQuest(ctx context.Context, userID, questFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	q, err := s.store.GetQuestByFlag(ctx, p.ID, questFlag)
	if err != nil || q == nil {
		return fmt.Errorf("life: quest not found")
	}
	if q.Status != "Pending" {
		return fmt.Errorf("life: quest not pending")
	}
	if err := s.store.MarkQuestStatus(ctx, q.ID, "Failed"); err != nil {
		return err
	}
	until := time.Now().Add(pkglife.RustDuration)
	if err := s.applyRust(ctx, p.ID, until); err != nil {
		return err
	}
	if err := s.blendCompletionRate(ctx, p.ID, false); err != nil {
		flog.WarnFields("life: blend completion rate failed", map[string]any{
			"profile_id": p.ID, "error": err.Error(),
		})
	}
	flog.InfoFields("life: quest failed", map[string]any{
		"uid": userID, "profile_id": p.ID, "quest_flag": questFlag, "tarnished_until": until,
	})
	return nil
}

// CompleteResult is returned to the UI after completion.
type CompleteResult struct {
	Quest         *gen.LifeQuest
	GainedExp     int
	GainedGold    int
	Dropped       bool
	ItemName      string
	ItemFlag      string
	Dice          float64
	PityForced    bool
	NewlyUnlocked []UnlockedAchievement
}

// UnlockedAchievement is one memorial unlock surfaced to the UI.
type UnlockedAchievement struct {
	Flag string
	Name string
}

// QuestEvidenceView is one quest evidence item for the UI.
type QuestEvidenceView struct {
	Flag       string
	SourceType string
	Content    string
	SourceURL  string
	Summary    string
	CreatedAt  time.Time
}

// QuestAdjudicationView is the latest AI ruling for one quest.
type QuestAdjudicationView struct {
	Flag               string
	Status             string
	Verdict            string
	Reason             string
	SuggestedExp       int
	SuggestedGold      int
	SuggestedNextSteps []string
	CreatedAt          time.Time
}

// QuestDMView bundles a pending quest with its evidence and latest ruling.
type QuestDMView struct {
	Quest        *gen.LifeQuest
	Evidence     []QuestEvidenceView
	Adjudication *QuestAdjudicationView
}

// CompleteQuest runs cascade + loot + action log in one store transaction.
func (s *Service) CompleteQuest(ctx context.Context, userID, questFlag string) (*CompleteResult, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	q, skill, char, err := s.loadCompletableQuest(ctx, p.ID, questFlag)
	if err != nil {
		return nil, err
	}
	slots, err := s.store.GetEquippedSlots(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	buffs, err := s.equippedBuffs(ctx, slots)
	if err != nil {
		return nil, err
	}
	goldReward := max(int(float64(q.BaseGoldReward)*buffs.GoldMult), 0)
	casc := pkglife.ApplyCascade(pkglife.CascadeInput{
		BaseExp: q.BaseExpReward, BaseGold: goldReward,
		Skill:          pkglife.StatSnapshot{Level: skill.Level, CurrentExp: skill.CurrentExp},
		Characteristic: pkglife.StatSnapshot{Level: char.Level, CurrentExp: char.CurrentExp},
		Profile:        pkglife.StatSnapshot{Level: p.Level, CurrentExp: p.Exp},
		ProfileGold:    p.Gold, ExpToCharacteristicRatio: skill.ExpToCharacteristicRatio,
	})
	pity := p.PityByTier
	if pity == nil {
		pity = map[string]int{}
	}
	roll, loot, pool, err := s.rollLoot(ctx, p, q, buffs, pity)
	if err != nil {
		return nil, err
	}
	pity[q.DropTier] = loot.NextPity
	dropEquipID, needLore, loreStatus, err := s.resolveDropEquip(ctx, q, loot, pool)
	if err != nil {
		return nil, err
	}
	var daily *gen.LifeQuest
	if q.Type == "Daily" {
		daily = q
	}
	persisted, err := s.store.PersistCompleteQuest(ctx, store.LifeCompletePersist{
		ProfileID: p.ID, QuestID: q.ID, SkillID: skill.ID, CharID: char.ID,
		SkillLevel: casc.Skill.Level, SkillExp: casc.Skill.CurrentExp,
		CharLevel: casc.Characteristic.Level, CharExp: casc.Characteristic.CurrentExp,
		ProfLevel: casc.Profile.Level, ProfExp: casc.Profile.CurrentExp, ProfGold: casc.ProfileGold,
		Pity: pity, RustInvIDs: slotInventoryIDs(slots),
		DropEquipID: dropEquipID, DropQuestID: q.ID, LoreStatus: loreStatus, NeedLore: needLore,
		ActionExp: casc.GainedExp, ActionGold: casc.GainedGold, Dice: roll, DailyRespawn: daily,
		QuestType: q.Type, Difficulty: q.AiEvaluatedDifficulty,
	})
	if err != nil {
		return nil, err
	}
	if err := s.blendCompletionRate(ctx, p.ID, true); err != nil {
		flog.WarnFields("life: blend completion rate failed", map[string]any{
			"profile_id": p.ID, "error": err.Error(),
		})
	}
	result := fillCompleteResult(q, casc, loot, roll, persisted)
	logQuestCompleted(userID, p.ID, casc.Profile.Level, result)
	return result, nil
}

func (s *Service) resolveDropEquip(ctx context.Context, q *gen.LifeQuest, loot pkglife.LootResult, pool []string) (equipID int64, needLore bool, loreStatus string, err error) {
	loreStatus = "none"
	if !loot.Dropped {
		return 0, false, loreStatus, nil
	}
	eqFlag := pool[loot.PoolIndex]
	eq, err := s.store.GetEquipmentByFlag(ctx, eqFlag)
	if err != nil || eq == nil {
		return 0, false, "", fmt.Errorf("life: equipment %s missing", eqFlag)
	}
	needLore = q.Type == "Boss" || q.AiEvaluatedDifficulty == "SSS" || q.AiEvaluatedDifficulty == "SS"
	if needLore {
		loreStatus = "pending"
	}
	return eq.ID, needLore, loreStatus, nil
}

func fillCompleteResult(q *gen.LifeQuest, casc pkglife.CascadeResult, loot pkglife.LootResult, roll float64, persisted *store.LifeCompleteResult) *CompleteResult {
	result := &CompleteResult{
		Quest: q, GainedExp: casc.GainedExp, GainedGold: casc.GainedGold,
		Dice: roll, PityForced: loot.ForcedPity,
	}
	q.Status = "Completed"
	if persisted != nil && persisted.Inventory != nil {
		result.Dropped = true
		result.ItemFlag = persisted.Inventory.Flag
		if persisted.Equipment != nil {
			result.ItemName = persisted.Equipment.Name
		}
	}
	if persisted != nil {
		for _, a := range persisted.NewlyUnlocked {
			if a == nil {
				continue
			}
			result.NewlyUnlocked = append(result.NewlyUnlocked, UnlockedAchievement{Flag: a.Flag, Name: a.Name})
		}
	}
	return result
}

func logQuestCompleted(userID string, profileID int64, level int, result *CompleteResult) {
	fields := map[string]any{
		"uid": userID, "profile_id": profileID, "quest_flag": result.Quest.Flag, "title": result.Quest.Title,
		"gained_exp": result.GainedExp, "gained_gold": result.GainedGold, "dice": result.Dice,
		"dropped": result.Dropped, "pity_forced": result.PityForced, "level": level,
	}
	if result.Dropped {
		fields["item_name"] = result.ItemName
		fields["item_flag"] = result.ItemFlag
	}
	flog.InfoFields("life: quest completed", fields)
}

func (s *Service) loadCompletableQuest(ctx context.Context, profileID int64, questFlag string) (*gen.LifeQuest, *gen.LifeSkill, *gen.LifeCharacteristic, error) {
	q, err := s.store.GetQuestByFlag(ctx, profileID, questFlag)
	if err != nil {
		return nil, nil, nil, err
	}
	if q == nil {
		return nil, nil, nil, fmt.Errorf("life: quest not found")
	}
	if q.Status != "Pending" {
		return nil, nil, nil, fmt.Errorf("life: quest not pending")
	}
	skill, err := s.store.GetSkill(ctx, q.SkillID)
	if err != nil || skill == nil {
		return nil, nil, nil, fmt.Errorf("life: skill missing")
	}
	char, err := s.store.GetCharacteristic(ctx, skill.CharacteristicID)
	if err != nil || char == nil {
		return nil, nil, nil, fmt.Errorf("life: characteristic missing")
	}
	return q, skill, char, nil
}

func (s *Service) rollLoot(ctx context.Context, p *gen.LifeProfile, q *gen.LifeQuest, buffs pkglife.BuffTotals, pity map[string]int) (float64, pkglife.LootResult, []string, error) {
	lootTable, err := s.store.GetLootTable(ctx, q.DropTier)
	if err != nil {
		return 0, pkglife.LootResult{}, nil, err
	}
	baseChance := 0.15
	pool := []string{}
	if lootTable != nil {
		baseChance = lootTable.BaseDropChance
		pool = lootTable.ItemPoolFlags
	}
	roll := pkglife.RollUnit(s.rng)
	loot := pkglife.ResolveLoot(pkglife.LootInput{
		BaseDropChance: baseChance, ProfileBonus: p.BaseDropRateBonus, EquippedDropRate: buffs.DropRate,
		PityCount: pity[q.DropTier], PityThreshold: pkglife.DefaultPityThreshold, Roll: roll, PoolSize: len(pool),
	})
	return roll, loot, pool, nil
}

// ListQuests lists quests for the user.
func (s *Service) ListQuests(ctx context.Context, userID, status string) ([]*gen.LifeQuest, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	return s.store.ListQuests(ctx, p.ID, status)
}

// ListPendingQuestDMViews returns pending quests with evidence and latest rulings.
func (s *Service) ListPendingQuestDMViews(ctx context.Context, userID string) ([]QuestDMView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	quests, err := s.store.ListQuests(ctx, p.ID, "Pending")
	if err != nil {
		return nil, err
	}
	out := make([]QuestDMView, 0, len(quests))
	for _, q := range quests {
		view, err := s.buildQuestDMView(ctx, p.ID, q)
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	return out, nil
}

// SubmitQuestEvidence stores a new evidence item for a pending quest.
func (s *Service) SubmitQuestEvidence(ctx context.Context, userID, questFlag, sourceType, content, sourceURL string) (*QuestEvidenceView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	q, err := s.store.GetQuestByFlag(ctx, p.ID, questFlag)
	if err != nil || q == nil {
		return nil, fmt.Errorf("life: quest not found")
	}
	if q.Status != "Pending" {
		return nil, fmt.Errorf("life: quest not pending")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("life: evidence content required")
	}
	sourceType = normalizeEvidenceSourceType(sourceType)
	row, err := s.store.CreateEvidence(ctx, store.LifeEvidenceInput{
		ProfileID:  p.ID,
		QuestID:    &q.ID,
		SourceType: sourceType,
		Content:    content,
		SourceURL:  strings.TrimSpace(sourceURL),
		Summary:    summarizeEvidence(content),
	})
	if err != nil {
		return nil, err
	}
	view := mapQuestEvidenceView(row)
	return &view, nil
}

// AdjudicateQuest asks the Life DM for a suggested quest ruling.
func (s *Service) AdjudicateQuest(ctx context.Context, userID, questFlag string) (*QuestAdjudicationView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	q, err := s.store.GetQuestByFlag(ctx, p.ID, questFlag)
	if err != nil || q == nil {
		return nil, fmt.Errorf("life: quest not found")
	}
	if q.Status != "Pending" {
		return nil, fmt.Errorf("life: quest not pending")
	}
	evidenceRows, err := s.store.ListEvidenceByQuest(ctx, p.ID, q.ID)
	if err != nil {
		return nil, err
	}
	if len(evidenceRows) == 0 {
		return nil, fmt.Errorf("life: evidence required before adjudication")
	}
	if err := s.store.EnsureAIContext(ctx, p.ID); err != nil {
		return nil, err
	}
	aiCtx, err := s.store.GetAIContext(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	_, activeGoalTitles, err := s.resolveGoalBinding(ctx, p.ID, "")
	if err != nil {
		return nil, err
	}
	actionRows, err := s.store.ListActionLogs(ctx, p.ID, 5)
	if err != nil {
		return nil, err
	}
	params := map[string]any{
		"quest_title":       q.Title,
		"quest_prompt":      q.Prompt,
		"quest_type":        q.Type,
		"difficulty":        q.AiEvaluatedDifficulty,
		"base_exp":          q.BaseExpReward,
		"base_gold":         q.BaseGoldReward,
		"active_goals":      activeGoalTitles,
		"evidence":          buildQuestEvidencePayload(evidenceRows),
		"recent_action_log": buildRecentActionLogPayload(actionRows),
	}
	if aiCtx != nil {
		params["ai_personality"] = aiCtx.AiDmPersonality
		params["completion_rate"] = aiCtx.HistoricalCompletionRate
		params["mood"] = aiCtx.RecentMoodAndBurnout
	}
	res, err := caplife.Invoke(ctx, hub.CapLife, lifecap.OpAdjudicateQuest, params)
	if err != nil {
		return nil, fmt.Errorf("life: adjudicate quest: %w", err)
	}
	ruling, err := decodeQuestAdjudication(res.Data)
	if err != nil {
		return nil, err
	}
	row, err := s.store.CreateAdjudication(ctx, store.LifeAdjudicationInput{
		ProfileID:          p.ID,
		QuestID:            q.ID,
		Status:             "suggested",
		Verdict:            ruling.Verdict,
		Reason:             ruling.Reason,
		SuggestedExp:       ruling.SuggestedExp,
		SuggestedGold:      ruling.SuggestedGold,
		SuggestedNextSteps: ruling.SuggestedNextSteps,
		EvidenceSnapshot:   buildQuestEvidencePayload(evidenceRows),
	})
	if err != nil {
		return nil, err
	}
	view := mapQuestAdjudicationView(row)
	return &view, nil
}

// ApplyQuestAdjudication accepts a suggested ruling and applies its effect.
func (s *Service) ApplyQuestAdjudication(ctx context.Context, userID, questFlag, adjudicationFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	q, err := s.store.GetQuestByFlag(ctx, p.ID, questFlag)
	if err != nil || q == nil {
		return fmt.Errorf("life: quest not found")
	}
	adjudication, err := s.store.GetAdjudicationByFlag(ctx, p.ID, adjudicationFlag)
	if err != nil || adjudication == nil {
		return fmt.Errorf("life: adjudication not found")
	}
	if adjudication.QuestID != q.ID {
		return fmt.Errorf("life: adjudication does not match quest")
	}
	if adjudication.Status != "suggested" {
		return fmt.Errorf("life: adjudication not pending")
	}
	switch adjudication.Verdict {
	case "completed":
		if _, err := s.CompleteQuest(ctx, userID, questFlag); err != nil {
			return err
		}
	case "failed":
		if err := s.FailQuest(ctx, userID, questFlag); err != nil {
			return err
		}
	case "partial", "needs_more_evidence":
		// Keep the quest pending; applying the ruling just acknowledges the feedback.
	default:
		return fmt.Errorf("life: unsupported adjudication verdict")
	}
	return s.store.MarkAdjudicationApplied(ctx, adjudication.ID)
}

func (s *Service) buildQuestDMView(ctx context.Context, profileID int64, q *gen.LifeQuest) (QuestDMView, error) {
	view := QuestDMView{Quest: q}
	if q == nil {
		return view, nil
	}
	evidenceRows, err := s.store.ListEvidenceByQuest(ctx, profileID, q.ID)
	if err != nil {
		return view, err
	}
	view.Evidence = make([]QuestEvidenceView, 0, len(evidenceRows))
	for _, row := range evidenceRows {
		view.Evidence = append(view.Evidence, mapQuestEvidenceView(row))
	}
	adjudication, err := s.store.GetLatestAdjudicationByQuest(ctx, profileID, q.ID)
	if err != nil {
		return view, err
	}
	if adjudication != nil {
		mapped := mapQuestAdjudicationView(adjudication)
		view.Adjudication = &mapped
	}
	return view, nil
}

// InventoryItem pairs inventory with equipment template.
type InventoryItem struct {
	Inv   *gen.LifeInventory
	Equip *gen.LifeEquipment
}

// AchievementView is one memorial achievement row for the UI.
type AchievementView struct {
	Flag         string
	Name         string
	Description  string
	Unlocked     bool
	UnlockedAt   *time.Time
	ShowProgress bool
	Current      int
	Target       int
	Retired      bool
}

// ListAchievements returns catalog rows with progress and unlock state for the user.
func (s *Service) ListAchievements(ctx context.Context, userID string) ([]AchievementView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	catalog, err := s.store.ListAchievements(ctx)
	if err != nil {
		return nil, err
	}
	progressRows, err := s.store.ListAchievementProgress(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	progress := make(map[string]int, len(progressRows))
	for _, row := range progressRows {
		progress[row.ConditionKey] = row.CurrentCount
	}
	unlockRows, err := s.store.ListAchievementUnlocks(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	unlockedAt := make(map[string]time.Time, len(unlockRows))
	for _, row := range unlockRows {
		unlockedAt[row.AchievementFlag] = row.UnlockedAt
	}
	out := make([]AchievementView, 0, len(catalog))
	for _, row := range catalog {
		at, unlocked := unlockedAt[row.Flag]
		if !row.Active && !unlocked {
			continue
		}
		def := pkglife.AchievementDef{
			Flag: row.Flag, Name: row.Name, Description: row.Description, Active: row.Active,
			Kind: row.Kind, QuestType: row.QuestType, Difficulty: row.Difficulty, Threshold: row.Threshold,
		}
		key := pkglife.AchievementConditionKey(row.QuestType, row.Difficulty)
		view := AchievementView{
			Flag: row.Flag, Name: row.Name, Description: row.Description,
			Unlocked: unlocked, ShowProgress: pkglife.AchievementShowsProgress(def),
			Current: progress[key], Target: row.Threshold, Retired: unlocked && !row.Active,
		}
		if view.Target < 1 {
			view.Target = 1
		}
		if unlocked {
			ts := at
			view.UnlockedAt = &ts
		}
		out = append(out, view)
	}
	return out, nil
}

// ListInventory returns inventory with templates.
func (s *Service) ListInventory(ctx context.Context, userID string) ([]InventoryItem, *gen.LifeEquippedSlots, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.store.ListInventory(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	slots, err := s.store.GetEquippedSlots(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	out := make([]InventoryItem, 0, len(rows))
	for _, inv := range rows {
		eq, err := s.store.GetEquipment(ctx, inv.EquipmentID)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, InventoryItem{Inv: inv, Equip: eq})
	}
	return out, slots, nil
}

// Equip wears an inventory item into its slot.
func (s *Service) Equip(ctx context.Context, userID, inventoryFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	inv, err := s.store.GetInventoryByFlag(ctx, p.ID, inventoryFlag)
	if err != nil || inv == nil {
		return fmt.Errorf("life: inventory not found")
	}
	eq, err := s.store.GetEquipment(ctx, inv.EquipmentID)
	if err != nil || eq == nil {
		return fmt.Errorf("life: equipment missing")
	}
	field := pkglife.SlotField(eq.SlotType)
	if field == "" {
		return fmt.Errorf("life: bad slot type %q", eq.SlotType)
	}
	id := inv.ID
	if err := s.store.SetEquippedSlot(ctx, p.ID, field, &id); err != nil {
		return err
	}
	flog.InfoFields("life: item equipped", map[string]any{
		"uid":            userID,
		"profile_id":     p.ID,
		"inventory_flag": inventoryFlag,
		"slot":           field,
		"equipment":      eq.Name,
		"rarity":         eq.Rarity,
	})
	return nil
}

// Unequip clears a named slot field.
func (s *Service) Unequip(ctx context.Context, userID, slotField string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	if err := s.store.SetEquippedSlot(ctx, p.ID, slotField, nil); err != nil {
		return err
	}
	flog.InfoFields("life: slot unequipped", map[string]any{
		"uid":        userID,
		"profile_id": p.ID,
		"slot":       slotField,
	})
	return nil
}

// ProcessPendingLore consumes lore outbox rows.
func (s *Service) ProcessPendingLore(ctx context.Context) (int, error) {
	if s == nil || s.store == nil {
		return 0, nil
	}
	rows, err := s.store.ListPendingLoreOutbox(ctx, 10)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, row := range rows {
		ok, err := s.processOneLoreOutbox(ctx, row)
		if err != nil {
			return n, err
		}
		if ok {
			n++
		}
	}
	if n > 0 {
		flog.InfoFields("life: lore batch processed", map[string]any{"count": n})
	}
	return n, nil
}

func (s *Service) processOneLoreOutbox(ctx context.Context, row *gen.EventOutbox) (bool, error) {
	invID, poison := parseLoreInventoryID(row)
	if poison {
		return false, s.discardLoreOutbox(ctx, row, invID, "poison payload")
	}
	inv, err := s.store.GetInventory(ctx, invID)
	if err != nil {
		return false, err
	}
	if inv == nil {
		return false, s.discardLoreOutbox(ctx, row, 0, "missing inventory")
	}
	name, rarity, questTitle, err := s.lorePromptInputs(ctx, inv)
	if err != nil {
		return false, err
	}
	res, err := caplife.Invoke(ctx, hub.CapLife, lifecap.OpGenerateInstanceLore, map[string]any{
		"quest_title": questTitle, "equipment_name": name, "rarity": rarity,
	})
	if err != nil {
		flog.WarnFields("life: lore generation failed; will retry", map[string]any{
			"inventory_id": inv.ID, "event_id": row.EventID, "error": err.Error(),
		})
		return false, nil
	}
	lore, err := decodeInstanceLore(res.Data)
	if err != nil {
		flog.WarnFields("life: lore response decode failed", map[string]any{
			"inventory_id": inv.ID, "error": err.Error(),
		})
		return false, s.discardLoreOutbox(ctx, row, inv.ID, "malformed lore response")
	}
	if err := s.store.UpdateInventoryLore(ctx, inv.ID, lore.Name, lore.Lore, "ready"); err != nil {
		flog.WarnFields("life: lore persist failed", map[string]any{
			"inventory_id": inv.ID, "error": err.Error(),
		})
		return false, nil
	}
	if err := s.store.MarkOutboxPublished(ctx, row.EventID); err != nil {
		flog.WarnFields("life: lore outbox publish failed", map[string]any{
			"event_id": row.EventID, "error": err.Error(),
		})
		return false, nil
	}
	flog.InfoFields("life: lore ready", map[string]any{
		"inventory_id": inv.ID, "instance_name": lore.Name, "event_id": row.EventID,
	})
	return true, nil
}

// discardLoreOutbox marks a poison/unrecoverable lore row published and fails pending inventory lore.
func (s *Service) discardLoreOutbox(ctx context.Context, row *gen.EventOutbox, invID int64, reason string) error {
	if invID > 0 {
		inv, err := s.store.GetInventory(ctx, invID)
		if err != nil {
			return err
		}
		if inv != nil && inv.LoreStatus == "pending" {
			if err := s.store.UpdateInventoryLoreStatus(ctx, inv.ID, "failed"); err != nil {
				return err
			}
		}
	}
	if err := s.store.MarkOutboxPublished(ctx, row.EventID); err != nil {
		return err
	}
	flog.WarnFields("life: lore outbox discarded", map[string]any{
		"event_id": row.EventID, "inventory_id": invID, "reason": reason,
	})
	return nil
}

func parseLoreInventoryID(row *gen.EventOutbox) (invID int64, poison bool) {
	raw, err := sonic.Marshal(row.Payload)
	if err != nil {
		flog.WarnFields("life: lore payload marshal failed", map[string]any{
			"event_id": row.EventID, "error": err.Error(),
		})
		return 0, true
	}
	var payload map[string]any
	if err := sonic.Unmarshal(raw, &payload); err != nil {
		flog.WarnFields("life: lore payload unmarshal failed", map[string]any{
			"event_id": row.EventID, "error": err.Error(),
		})
		return 0, true
	}
	id, ok := asInt64(payload["inventory_id"])
	if !ok || id == 0 {
		return 0, true
	}
	return id, false
}

func (s *Service) lorePromptInputs(ctx context.Context, inv *gen.LifeInventory) (name, rarity, questTitle string, err error) {
	name, rarity = "Relic", "Rare"
	eq, err := s.store.GetEquipment(ctx, inv.EquipmentID)
	if err != nil {
		return "", "", "", err
	}
	if eq != nil {
		name, rarity = eq.Name, eq.Rarity
	}
	questTitle = "a hard-won quest"
	if inv.SourceQuestID == nil {
		return name, rarity, questTitle, nil
	}
	q, err := s.store.GetQuest(ctx, *inv.SourceQuestID)
	if err != nil {
		return "", "", "", err
	}
	if q != nil && q.Title != "" {
		questTitle = q.Title
	}
	return name, rarity, questTitle, nil
}

func decodeInstanceLore(data any) (*lifecap.InstanceLore, error) {
	raw, err := sonic.Marshal(data)
	if err != nil {
		return nil, err
	}
	lore := &lifecap.InstanceLore{}
	if err := sonic.Unmarshal(raw, lore); err != nil {
		return nil, err
	}
	return lore, nil
}

func mapQuestEvidenceView(row *gen.LifeEvidence) QuestEvidenceView {
	if row == nil {
		return QuestEvidenceView{}
	}
	return QuestEvidenceView{
		Flag:       row.Flag,
		SourceType: row.SourceType,
		Content:    row.Content,
		SourceURL:  row.SourceURL,
		Summary:    row.Summary,
		CreatedAt:  row.CreatedAt,
	}
}

func mapQuestAdjudicationView(row *gen.LifeAdjudication) QuestAdjudicationView {
	if row == nil {
		return QuestAdjudicationView{}
	}
	return QuestAdjudicationView{
		Flag:               row.Flag,
		Status:             row.Status,
		Verdict:            row.Verdict,
		Reason:             row.Reason,
		SuggestedExp:       row.SuggestedExp,
		SuggestedGold:      row.SuggestedGold,
		SuggestedNextSteps: append([]string(nil), row.SuggestedNextSteps...),
		CreatedAt:          row.CreatedAt,
	}
}

func buildQuestEvidencePayload(rows []*gen.LifeEvidence) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, map[string]any{
			"source_type": row.SourceType,
			"content":     row.Content,
			"source_url":  row.SourceURL,
			"summary":     row.Summary,
		})
	}
	return out
}

func buildRecentActionLogPayload(rows []*gen.LifeActionLog) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		payload := map[string]any{
			"source_type": row.SourceType,
			"summary":     row.Summary,
			"gained_exp":  row.GainedExp,
			"gained_gold": row.GainedGold,
		}
		if row.QuestID != nil {
			payload["quest_id"] = *row.QuestID
		}
		out = append(out, payload)
	}
	return out
}

func normalizeEvidenceSourceType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "link":
		return "link"
	case "artifact":
		return "artifact"
	default:
		return "note"
	}
}

func summarizeEvidence(content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= 120 {
		return content
	}
	return content[:120]
}

// ListActionLogs returns recent completion audit rows for the user.
func (s *Service) ListActionLogs(ctx context.Context, userID string, limit int) ([]ActionLogView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListActionLogs(ctx, p.ID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ActionLogView, 0, len(rows))
	for _, row := range rows {
		view := ActionLogView{
			Flag: row.Flag, SourceType: row.SourceType, GainedExp: row.GainedExp, GainedGold: row.GainedGold,
			CreatedAt: row.CreatedAt,
		}
		if row.DiceRollResult != nil {
			view.Dice = *row.DiceRollResult
			view.HasDice = true
		}
		view.Dropped = row.DroppedInventoryID != nil
		view.QuestTitle = strings.TrimSpace(row.Summary)
		if row.QuestID != nil {
			q, qerr := s.store.GetQuest(ctx, *row.QuestID)
			if qerr != nil {
				return nil, qerr
			}
			if q != nil {
				view.QuestTitle = q.Title
				view.QuestFlag = q.Flag
			}
		}
		out = append(out, view)
	}
	return out, nil
}

// ActionLogView is one audit row for the UI.
type ActionLogView struct {
	Flag       string
	SourceType string
	QuestFlag  string
	QuestTitle string
	GainedExp  int
	GainedGold int
	Dice       float64
	HasDice    bool
	Dropped    bool
	CreatedAt  time.Time
}

func (s *Service) equippedBuffs(ctx context.Context, slots *gen.LifeEquippedSlots) (pkglife.BuffTotals, error) {
	empty := pkglife.BuffTotals{Stats: map[string]float64{}, GoldMult: 1}
	if slots == nil {
		return empty, nil
	}
	now := time.Now()
	if pkglife.IsTarnished(slots.TarnishedUntil, now) {
		return empty, nil
	}
	ids := slotInventoryIDPtrs(slots)
	var buffMaps []map[string]float64
	for _, id := range ids {
		if id == nil {
			continue
		}
		inv, err := s.store.GetInventory(ctx, *id)
		if err != nil {
			return empty, err
		}
		if inv == nil || pkglife.IsTarnished(inv.TarnishedUntil, now) {
			continue
		}
		eq, err := s.store.GetEquipment(ctx, inv.EquipmentID)
		if err != nil {
			return empty, err
		}
		if eq == nil {
			continue
		}
		buffMaps = append(buffMaps, pkglife.MergeBuffs(eq.StatBuffs, inv.InstanceBuffs))
	}
	return pkglife.SumEquippedBuffs(buffMaps), nil
}

func (s *Service) equippedPrivileges(ctx context.Context, slots *gen.LifeEquippedSlots) (map[string]any, error) {
	out := map[string]any{}
	if slots == nil {
		return out, nil
	}
	now := time.Now()
	if pkglife.IsTarnished(slots.TarnishedUntil, now) {
		return out, nil
	}
	for _, id := range slotInventoryIDPtrs(slots) {
		if id == nil {
			continue
		}
		inv, err := s.store.GetInventory(ctx, *id)
		if err != nil {
			return nil, err
		}
		if inv == nil || pkglife.IsTarnished(inv.TarnishedUntil, now) {
			continue
		}
		eq, err := s.store.GetEquipment(ctx, inv.EquipmentID)
		if err != nil {
			return nil, err
		}
		if eq == nil {
			continue
		}
		maps.Copy(out, eq.AiUnlockedPrivilege)
	}
	return out, nil
}

func privilegeDepth(privs map[string]any) string {
	if privs == nil {
		return ""
	}
	if v, ok := privs["ai_breakdown_depth"].(string); ok {
		return v
	}
	return ""
}

func (s *Service) dropChanceForQuest(ctx context.Context, p *gen.LifeProfile, q *gen.LifeQuest) (float64, error) {
	slots, err := s.store.GetEquippedSlots(ctx, p.ID)
	if err != nil {
		return 0, err
	}
	buffs, err := s.equippedBuffs(ctx, slots)
	if err != nil {
		return 0, err
	}
	lootTable, err := s.store.GetLootTable(ctx, q.DropTier)
	if err != nil {
		return 0, err
	}
	base := 0.15
	if lootTable != nil {
		base = lootTable.BaseDropChance
	}
	return pkglife.PreviewDropChance(pkglife.LootInput{
		BaseDropChance: base, ProfileBonus: p.BaseDropRateBonus, EquippedDropRate: buffs.DropRate,
	}), nil
}

// PreviewDropChance returns the effective drop chance for a pending quest.
func (s *Service) PreviewDropChance(ctx context.Context, userID, questFlag string) (float64, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return 0, err
	}
	q, err := s.store.GetQuestByFlag(ctx, p.ID, questFlag)
	if err != nil || q == nil {
		return 0, fmt.Errorf("life: quest not found")
	}
	return s.dropChanceForQuest(ctx, p, q)
}

func (s *Service) applyRust(ctx context.Context, profileID int64, until time.Time) error {
	if err := s.store.SetEquippedSlotsTarnishedUntil(ctx, profileID, &until); err != nil {
		return err
	}
	slots, err := s.store.GetEquippedSlots(ctx, profileID)
	if err != nil || slots == nil {
		return err
	}
	for _, id := range slotInventoryIDPtrs(slots) {
		if id == nil {
			continue
		}
		if err := s.store.SetInventoryTarnishedUntil(ctx, *id, &until); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) blendCompletionRate(ctx context.Context, profileID int64, success bool) error {
	if err := s.store.EnsureAIContext(ctx, profileID); err != nil {
		return err
	}
	ai, err := s.store.GetAIContext(ctx, profileID)
	if err != nil {
		return err
	}
	if ai == nil {
		return fmt.Errorf("life: ai context missing after ensure")
	}
	sample := 0.0
	if success {
		sample = 1.0
	}
	rate := ai.HistoricalCompletionRate*0.9 + sample*0.1
	mood := ai.RecentMoodAndBurnout
	if mood == nil {
		mood = map[string]any{}
	}
	if success {
		mood["last_outcome"] = "completed"
	} else {
		mood["last_outcome"] = "failed"
	}
	return s.store.UpdateAIContext(ctx, profileID, rate, mood, ai.AiDmPersonality)
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	default:
		return 0, false
	}
}

func slotInventoryIDPtrs(slots *gen.LifeEquippedSlots) []*int64 {
	if slots == nil {
		return nil
	}
	return []*int64{
		slots.HeadSlot, slots.WeaponSlot, slots.ArmorSlot,
		slots.ShoesSlot, slots.AccessorySlot, slots.ArtifactSlot,
	}
}

func slotInventoryIDs(slots *gen.LifeEquippedSlots) []int64 {
	ptrs := slotInventoryIDPtrs(slots)
	out := make([]int64, 0, len(ptrs))
	for _, id := range ptrs {
		if id != nil {
			out = append(out, *id)
		}
	}
	return out
}
