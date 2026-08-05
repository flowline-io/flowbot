package life

import (
	"context"
	"maps"
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
	"github.com/flowline-io/flowbot/pkg/types"
)

// Service orchestrates Life use cases for the web UI.
type Service struct {
	store *store.LifeStore
}

// NewService constructs a Life service.
func NewService(ls *store.LifeStore) *Service {
	return &Service{store: ls}
}

// EnsureProfile creates the profile graph for a user if needed.
func (s *Service) EnsureProfile(ctx context.Context, userID, nickname, defaultClass string) (*gen.LifeProfile, error) {
	if s == nil || s.store == nil {
		return nil, lifeUnavailable("service unavailable")
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

// CharacterView is the character / identity aggregate (also feeds Goals, Plan, Quests).
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
	goals, err := s.store.ListGoals(ctx, profileID, pkglife.GoalStatusActive)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[int64]*gen.LifeGoal, len(goals))
	for _, g := range goals {
		byID[g.ID] = g
	}
	titles := make([]string, 0, len(goals))
	for _, g := range goals {
		titles = append(titles, goalContextTitle(g, byID))
	}
	if goalFlag == "" {
		return nil, titles, nil
	}
	g, err := s.store.GetGoalByFlag(ctx, profileID, goalFlag)
	if err != nil {
		return nil, nil, err
	}
	if g == nil || g.Status != pkglife.GoalStatusActive {
		return nil, nil, lifeNotFound("goal not found")
	}
	return &g.ID, titles, nil
}

func goalContextTitle(g *gen.LifeGoal, byID map[int64]*gen.LifeGoal) string {
	if g == nil {
		return ""
	}
	if g.AreaID == nil {
		return g.Title
	}
	area, ok := byID[*g.AreaID]
	if !ok || area == nil {
		return g.Title
	}
	return g.Title + " · " + area.Title
}

// GoalView is a UI-ready goal row with optional Area parent labels resolved.
type GoalView struct {
	Flag      string
	Title     string
	Category  string
	Status    string
	AreaFlag  string
	AreaTitle string
}

// MapGoalViews maps goal entities to view rows, resolving Area parents within the slice.
func MapGoalViews(goals []*gen.LifeGoal) []GoalView {
	byID := make(map[int64]*gen.LifeGoal, len(goals))
	for _, g := range goals {
		if g != nil {
			byID[g.ID] = g
		}
	}
	rows := make([]GoalView, 0, len(goals))
	for _, g := range goals {
		if g == nil {
			continue
		}
		row := GoalView{Flag: g.Flag, Title: g.Title, Category: g.Category, Status: g.Status}
		if g.AreaID != nil {
			if area, ok := byID[*g.AreaID]; ok && area != nil {
				row.AreaFlag = area.Flag
				row.AreaTitle = area.Title
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func (s *Service) resolveAreaID(ctx context.Context, profileID int64, category, areaFlag string) (*int64, error) {
	if category != pkglife.GoalCategoryProject && category != pkglife.GoalCategoryResource {
		return nil, nil
	}
	areaFlag = strings.TrimSpace(areaFlag)
	if areaFlag == "" {
		return nil, nil
	}
	area, err := s.store.GetGoalByFlag(ctx, profileID, areaFlag)
	if err != nil {
		return nil, err
	}
	if area == nil || area.Category != pkglife.GoalCategoryArea || area.Status != pkglife.GoalStatusActive {
		return nil, lifeInvalid("area not found or not active")
	}
	return &area.ID, nil
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
		return nil, types.WrapError(types.ErrProvider, "life: evaluate quest", err)
	}
	return decodeQuestEvaluation(res.Data)
}

func decodeQuestEvaluation(data any) (*lifecap.QuestEvaluation, error) {
	if ev, ok := data.(*lifecap.QuestEvaluation); ok {
		return ev, nil
	}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return nil, lifeInvalid("unexpected evaluate payload")
	}
	ev := &lifecap.QuestEvaluation{}
	if err := sonic.Unmarshal(raw, ev); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "life: decode evaluate payload", err)
	}
	return ev, nil
}

func decodeQuestAdjudication(data any) (*lifecap.QuestAdjudication, error) {
	if ruling, ok := data.(*lifecap.QuestAdjudication); ok {
		return ruling, nil
	}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return nil, lifeInvalid("unexpected adjudication payload")
	}
	ruling := &lifecap.QuestAdjudication{}
	if err := sonic.Unmarshal(raw, ruling); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "life: decode adjudication payload", err)
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

// CreateGoal creates an Active PARA goal. areaFlag optionally links Project/Resource to an Active Area.
func (s *Service) CreateGoal(ctx context.Context, userID, title, category, areaFlag string) (*gen.LifeGoal, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, lifeInvalid("goal title required")
	}
	switch category {
	case pkglife.GoalCategoryProject, pkglife.GoalCategoryArea, pkglife.GoalCategoryResource:
	default:
		category = pkglife.GoalCategoryProject
	}
	areaID, err := s.resolveAreaID(ctx, p.ID, category, areaFlag)
	if err != nil {
		return nil, err
	}
	g, err := s.store.CreateGoal(ctx, p.ID, title, category, areaID)
	if err != nil {
		return nil, err
	}
	flog.InfoFields("life: goal created", map[string]any{
		"uid": userID, "profile_id": p.ID, "goal_flag": g.Flag, "category": category, "title": title,
		"area_flag": strings.TrimSpace(areaFlag),
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
		return lifeNotFound("goal not found")
	}
	switch status {
	case pkglife.GoalStatusActive, pkglife.GoalStatusPaused, pkglife.GoalStatusCompleted:
	default:
		return lifeInvalid("invalid goal status")
	}
	return s.store.UpdateGoalStatus(ctx, g.ID, status)
}

// UpdateGoal updates title, category, and optional Area parent for a goal by flag.
func (s *Service) UpdateGoal(ctx context.Context, userID, goalFlag, title, category, areaFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	g, err := s.store.GetGoalByFlag(ctx, p.ID, goalFlag)
	if err != nil || g == nil {
		return lifeNotFound("goal not found")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return lifeInvalid("goal title required")
	}
	switch category {
	case pkglife.GoalCategoryProject, pkglife.GoalCategoryArea, pkglife.GoalCategoryResource:
	default:
		category = g.Category
	}
	wasArea := g.Category == pkglife.GoalCategoryArea
	if wasArea && category != pkglife.GoalCategoryArea {
		if err := s.store.ClearGoalAreaRefs(ctx, g.ID); err != nil {
			return err
		}
	}
	areaID, err := s.resolveAreaID(ctx, p.ID, category, areaFlag)
	if err != nil {
		return err
	}
	if err := s.store.UpdateGoal(ctx, g.ID, title, category, areaID); err != nil {
		return err
	}
	flog.InfoFields("life: goal updated", map[string]any{
		"uid": userID, "profile_id": p.ID, "goal_flag": goalFlag, "category": category,
		"area_flag": strings.TrimSpace(areaFlag),
	})
	return nil
}

// DeleteGoal removes a goal by flag. Deleting an Area clears child area_id links first.
func (s *Service) DeleteGoal(ctx context.Context, userID, goalFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	g, err := s.store.GetGoalByFlag(ctx, p.ID, goalFlag)
	if err != nil || g == nil {
		return lifeNotFound("goal not found")
	}
	if g.Category == pkglife.GoalCategoryArea {
		if err := s.store.ClearGoalAreaRefs(ctx, g.ID); err != nil {
			return err
		}
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
		return lifeNotFound("quest not found")
	}
	if q.Status != "Pending" {
		return lifeConflict("quest not pending")
	}
	slots, err := s.store.GetEquippedSlots(ctx, p.ID)
	if err != nil {
		return err
	}
	until := time.Now().Add(pkglife.RustDuration)
	if err := s.store.PersistFailQuest(ctx, p.ID, q.ID, slotInventoryIDs(slots), until); err != nil {
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
	notifyQuestFailed(userID, q.Title)
	return nil
}

// DismissQuest removes a pending quest without rust or completion-rate penalty.
func (s *Service) DismissQuest(ctx context.Context, userID, questFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	q, err := s.store.GetQuestByFlag(ctx, p.ID, questFlag)
	if err != nil || q == nil {
		return lifeNotFound("quest not found")
	}
	if q.Status != "Pending" {
		return lifeConflict("quest not pending")
	}
	if err := s.store.MarkQuestStatus(ctx, q.ID, "Dismissed"); err != nil {
		return err
	}
	flog.InfoFields("life: quest dismissed", map[string]any{
		"uid": userID, "profile_id": p.ID, "quest_flag": questFlag,
	})
	return nil
}

// CompleteResult is returned to the UI after completion.
type CompleteResult struct {
	Quest              *gen.LifeQuest
	GainedExp          int
	GainedGold         int
	Dropped            bool
	ItemName           string
	ItemFlag           string
	ItemRarity         string
	Dice               float64
	PityForced         bool
	NewlyUnlocked      []UnlockedAchievement
	ProfileLevelBefore int
	ProfileLevelAfter  int
	SkillName          string
	SkillLevelBefore   int
	SkillLevelAfter    int
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
	DropChance   float64
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
	lootTable, err := s.store.GetLootTable(ctx, q.DropTier)
	if err != nil {
		return nil, err
	}
	baseChance := 0.15
	var pool []string
	if lootTable != nil {
		baseChance = lootTable.BaseDropChance
		pool = lootTable.ItemPoolFlags
	}
	roll, err := pkglife.RollUnit()
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
		RustInvIDs: slotInventoryIDs(slots),
		DropQuestID: q.ID,
		ActionExp: casc.GainedExp, ActionGold: casc.GainedGold, Dice: roll, DailyRespawn: daily,
		QuestType: q.Type, Difficulty: q.AiEvaluatedDifficulty,
		ResolveLootInTx: true, DropTier: q.DropTier,
		LootBaseChance: baseChance, LootPool: pool,
		ProfileBonus: p.BaseDropRateBonus, EquippedDropRate: buffs.DropRate,
	})
	if err != nil {
		return nil, err
	}
	if err := s.blendCompletionRate(ctx, p.ID, true); err != nil {
		flog.WarnFields("life: blend completion rate failed", map[string]any{
			"profile_id": p.ID, "error": err.Error(),
		})
	}
	result := fillCompleteResult(q, skill, p.Level, casc, persisted.Loot, persisted.Dice, persisted)
	logQuestCompleted(userID, p.ID, casc.Profile.Level, result)
	notifyQuestCompleted(userID, result)
	return result, nil
}

func fillCompleteResult(
	q *gen.LifeQuest,
	skill *gen.LifeSkill,
	profileLevelBefore int,
	casc pkglife.CascadeResult,
	loot pkglife.LootResult,
	roll float64,
	persisted *store.LifeCompleteResult,
) *CompleteResult {
	result := &CompleteResult{
		Quest: q, GainedExp: casc.GainedExp, GainedGold: casc.GainedGold,
		Dice: roll, PityForced: loot.ForcedPity,
		ProfileLevelBefore: profileLevelBefore,
		ProfileLevelAfter:  casc.Profile.Level,
		SkillLevelAfter:    casc.Skill.Level,
	}
	if skill != nil {
		result.SkillName = skill.Name
		result.SkillLevelBefore = skill.Level
	}
	q.Status = "Completed"
	if persisted != nil && persisted.Inventory != nil {
		result.Dropped = true
		result.ItemFlag = persisted.Inventory.Flag
		if persisted.Equipment != nil {
			result.ItemName = persisted.Equipment.Name
			result.ItemRarity = persisted.Equipment.Rarity
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
		return nil, nil, nil, lifeNotFound("quest not found")
	}
	if q.Status != "Pending" {
		return nil, nil, nil, lifeConflict("quest not pending")
	}
	skill, err := s.store.GetSkill(ctx, q.SkillID)
	if err != nil || skill == nil {
		return nil, nil, nil, lifeNotFound("skill missing")
	}
	char, err := s.store.GetCharacteristic(ctx, skill.CharacteristicID)
	if err != nil || char == nil {
		return nil, nil, nil, lifeNotFound("characteristic missing")
	}
	return q, skill, char, nil
}

// ListQuests lists quests for the user.
func (s *Service) ListQuests(ctx context.Context, userID, status string) ([]*gen.LifeQuest, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	return s.store.ListQuests(ctx, p.ID, status)
}

// ListCompletedQuestsPage returns one page of completed quests and the total count.
func (s *Service) ListCompletedQuestsPage(ctx context.Context, userID string, page, perPage int) ([]*gen.LifeQuest, int, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, 0, err
	}
	page, perPage, offset := normalizeLifeListPage(page, perPage)
	rows, total, err := s.store.ListQuestsPage(ctx, p.ID, "Completed", perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	if total > 0 {
		maxPage := (total + perPage - 1) / perPage
		if page > maxPage {
			page = maxPage
			offset = (page - 1) * perPage
			rows, total, err = s.store.ListQuestsPage(ctx, p.ID, "Completed", perPage, offset)
			if err != nil {
				return nil, 0, err
			}
		}
	}
	return rows, total, nil
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
	if len(quests) == 0 {
		return nil, nil
	}

	bundle, err := s.loadPendingQuestDMBundle(ctx, p, quests)
	if err != nil {
		return nil, err
	}
	out := make([]QuestDMView, 0, len(quests))
	for _, q := range quests {
		if q == nil {
			continue
		}
		out = append(out, assembleQuestDMView(p, q, bundle))
	}
	return out, nil
}

type pendingQuestDMBundle struct {
	evidenceByQuest map[int64][]*gen.LifeEvidence
	adjudications   map[int64]*gen.LifeAdjudication
	buffs           pkglife.BuffTotals
	lootByTier      map[string]*gen.LifeLootTable
}

func (s *Service) loadPendingQuestDMBundle(ctx context.Context, p *gen.LifeProfile, quests []*gen.LifeQuest) (*pendingQuestDMBundle, error) {
	questIDs, tiers := questIDsAndTiers(quests)
	evidenceRows, err := s.store.ListEvidenceByQuestIDs(ctx, p.ID, questIDs)
	if err != nil {
		return nil, err
	}
	adjudications, err := s.store.MapLatestAdjudicationsByQuestIDs(ctx, p.ID, questIDs)
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
	lootByTier, err := s.store.MapLootTablesByTiers(ctx, tiers)
	if err != nil {
		return nil, err
	}
	return &pendingQuestDMBundle{
		evidenceByQuest: groupEvidenceByQuest(evidenceRows),
		adjudications:   adjudications,
		buffs:           buffs,
		lootByTier:      lootByTier,
	}, nil
}

func questIDsAndTiers(quests []*gen.LifeQuest) ([]int64, []string) {
	questIDs := make([]int64, 0, len(quests))
	tiers := make([]string, 0, len(quests))
	tierSeen := make(map[string]struct{}, len(quests))
	for _, q := range quests {
		if q == nil {
			continue
		}
		questIDs = append(questIDs, q.ID)
		if _, ok := tierSeen[q.DropTier]; ok {
			continue
		}
		tierSeen[q.DropTier] = struct{}{}
		tiers = append(tiers, q.DropTier)
	}
	return questIDs, tiers
}

func groupEvidenceByQuest(rows []*gen.LifeEvidence) map[int64][]*gen.LifeEvidence {
	out := make(map[int64][]*gen.LifeEvidence)
	for _, row := range rows {
		if row == nil || row.QuestID == nil {
			continue
		}
		out[*row.QuestID] = append(out[*row.QuestID], row)
	}
	return out
}

func assembleQuestDMView(p *gen.LifeProfile, q *gen.LifeQuest, bundle *pendingQuestDMBundle) QuestDMView {
	view := QuestDMView{Quest: q}
	for _, row := range bundle.evidenceByQuest[q.ID] {
		view.Evidence = append(view.Evidence, mapQuestEvidenceView(row))
	}
	if adjudication := bundle.adjudications[q.ID]; adjudication != nil {
		mapped := mapQuestAdjudicationView(adjudication)
		view.Adjudication = &mapped
	}
	base := 0.15
	if lootTable := bundle.lootByTier[q.DropTier]; lootTable != nil {
		base = lootTable.BaseDropChance
	}
	view.DropChance = pkglife.PreviewDropChance(pkglife.LootInput{
		BaseDropChance: base, ProfileBonus: p.BaseDropRateBonus, EquippedDropRate: bundle.buffs.DropRate,
	})
	return view
}

// SubmitQuestEvidence stores a new evidence item for a pending quest.
func (s *Service) SubmitQuestEvidence(ctx context.Context, userID, questFlag, sourceType, content, sourceURL string) (*QuestEvidenceView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	q, err := s.store.GetQuestByFlag(ctx, p.ID, questFlag)
	if err != nil || q == nil {
		return nil, lifeNotFound("quest not found")
	}
	if q.Status != "Pending" {
		return nil, lifeConflict("quest not pending")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, lifeInvalid("evidence content required")
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
		return nil, lifeNotFound("quest not found")
	}
	if q.Status != "Pending" {
		return nil, lifeConflict("quest not pending")
	}
	evidenceRows, err := s.store.ListEvidenceByQuest(ctx, p.ID, q.ID)
	if err != nil {
		return nil, err
	}
	if len(evidenceRows) == 0 {
		return nil, lifeInvalid("evidence required before adjudication")
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
		return nil, types.WrapError(types.ErrProvider, "life: adjudicate quest", err)
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
		return lifeNotFound("quest not found")
	}
	adjudication, err := s.store.GetAdjudicationByFlag(ctx, p.ID, adjudicationFlag)
	if err != nil || adjudication == nil {
		return lifeNotFound("adjudication not found")
	}
	if adjudication.QuestID != q.ID {
		return lifeInvalid("adjudication does not match quest")
	}
	if adjudication.Status != "suggested" {
		return lifeConflict("adjudication not pending")
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
		return lifeInvalid("unsupported adjudication verdict")
	}
	return s.store.MarkAdjudicationApplied(ctx, adjudication.ID)
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

// InventoryPage is one backpack page plus currently equipped items for the board.
type InventoryPage struct {
	Items    []InventoryItem
	Equipped []InventoryItem
	Slots    *gen.LifeEquippedSlots
	Total    int
}

// ListInventory returns inventory with templates.
func (s *Service) ListInventory(ctx context.Context, userID string) ([]InventoryItem, *gen.LifeEquippedSlots, error) {
	page, err := s.ListInventoryPage(ctx, userID, 1, 0)
	if err != nil {
		return nil, nil, err
	}
	return page.Items, page.Slots, nil
}

// ListInventoryPage returns one backpack page, equipped items, slots, and total count.
// A non-positive perPage returns the full backpack on page 1.
func (s *Service) ListInventoryPage(ctx context.Context, userID string, page, perPage int) (*InventoryPage, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	slots, err := s.store.GetEquippedSlots(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	var rows []*gen.LifeInventory
	var total int
	if perPage < 1 {
		rows, total, err = s.store.ListInventoryPage(ctx, p.ID, 0, 0)
	} else {
		page, perPage, offset := normalizeLifeListPage(page, perPage)
		rows, total, err = s.store.ListInventoryPage(ctx, p.ID, perPage, offset)
		if err == nil && total > 0 {
			maxPage := (total + perPage - 1) / perPage
			if page > maxPage {
				page = maxPage
				offset = (page - 1) * perPage
				rows, total, err = s.store.ListInventoryPage(ctx, p.ID, perPage, offset)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	items, err := s.mapInventoryItems(ctx, rows)
	if err != nil {
		return nil, err
	}
	equippedRows, err := s.loadEquippedInventory(ctx, p.ID, slots)
	if err != nil {
		return nil, err
	}
	equipped, err := s.mapInventoryItems(ctx, equippedRows)
	if err != nil {
		return nil, err
	}
	return &InventoryPage{Items: items, Equipped: equipped, Slots: slots, Total: total}, nil
}

func (s *Service) mapInventoryItems(ctx context.Context, rows []*gen.LifeInventory) ([]InventoryItem, error) {
	out := make([]InventoryItem, 0, len(rows))
	for _, inv := range rows {
		eq, err := s.store.GetEquipment(ctx, inv.EquipmentID)
		if err != nil {
			return nil, err
		}
		out = append(out, InventoryItem{Inv: inv, Equip: eq})
	}
	return out, nil
}

func (s *Service) loadEquippedInventory(ctx context.Context, profileID int64, slots *gen.LifeEquippedSlots) ([]*gen.LifeInventory, error) {
	if slots == nil {
		return nil, nil
	}
	ids := make([]int64, 0, 6)
	for _, id := range []*int64{slots.HeadSlot, slots.WeaponSlot, slots.ArmorSlot, slots.ShoesSlot, slots.AccessorySlot, slots.ArtifactSlot} {
		if id != nil {
			ids = append(ids, *id)
		}
	}
	out := make([]*gen.LifeInventory, 0, len(ids))
	for _, id := range ids {
		inv, err := s.store.GetInventory(ctx, id)
		if err != nil {
			return nil, err
		}
		if inv == nil || inv.LifeProfileID != profileID {
			continue
		}
		out = append(out, inv)
	}
	return out, nil
}

// Equip wears an inventory item into its slot.
func (s *Service) Equip(ctx context.Context, userID, inventoryFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	inv, err := s.store.GetInventoryByFlag(ctx, p.ID, inventoryFlag)
	if err != nil || inv == nil {
		return lifeNotFound("inventory not found")
	}
	eq, err := s.store.GetEquipment(ctx, inv.EquipmentID)
	if err != nil || eq == nil {
		return lifeNotFound("equipment missing")
	}
	field := pkglife.SlotField(eq.SlotType)
	if field == "" {
		return lifeInvalid("bad slot type %q", eq.SlotType)
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
	if content == "" {
		return ""
	}
	runes := []rune(content)
	if len(runes) <= 120 {
		return content
	}
	return string(runes[:120])
}

// ListActionLogs returns recent completion audit rows for the user.
func (s *Service) ListActionLogs(ctx context.Context, userID string, limit int) ([]ActionLogView, error) {
	if limit < 1 {
		limit = 20
	}
	items, _, err := s.ListActionLogsPage(ctx, userID, 1, limit)
	return items, err
}

// ListActionLogsPage returns one page of completion audit rows and the total count.
func (s *Service) ListActionLogsPage(ctx context.Context, userID string, page, perPage int) ([]ActionLogView, int, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, 0, err
	}
	page, perPage, offset := normalizeLifeListPage(page, perPage)
	rows, total, err := s.store.ListActionLogsPage(ctx, p.ID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	if total > 0 {
		maxPage := (total + perPage - 1) / perPage
		if page > maxPage {
			page = maxPage
			offset = (page - 1) * perPage
			rows, total, err = s.store.ListActionLogsPage(ctx, p.ID, perPage, offset)
			if err != nil {
				return nil, 0, err
			}
		}
	}
	out, err := s.mapActionLogViews(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func normalizeLifeListPage(page, perPage int) (int, int, int) {
	if perPage < 1 {
		perPage = 10
	}
	if page < 1 {
		page = 1
	}
	return page, perPage, (page - 1) * perPage
}

func (s *Service) mapActionLogViews(ctx context.Context, rows []*gen.LifeActionLog) ([]ActionLogView, error) {
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
	invByID, eqByID, invIDs, now, err := s.loadEquippedMaps(ctx, slots)
	if err != nil {
		return empty, err
	}
	if invByID == nil {
		return empty, nil
	}
	var buffMaps []map[string]float64
	for _, id := range invIDs {
		inv := invByID[id]
		if inv == nil || pkglife.IsTarnished(inv.TarnishedUntil, now) {
			continue
		}
		eq := eqByID[inv.EquipmentID]
		if eq == nil {
			continue
		}
		buffMaps = append(buffMaps, pkglife.MergeBuffs(eq.StatBuffs, inv.InstanceBuffs))
	}
	return pkglife.SumEquippedBuffs(buffMaps), nil
}

func (s *Service) equippedPrivileges(ctx context.Context, slots *gen.LifeEquippedSlots) (map[string]any, error) {
	out := map[string]any{}
	invByID, eqByID, invIDs, now, err := s.loadEquippedMaps(ctx, slots)
	if err != nil {
		return nil, err
	}
	if invByID == nil {
		return out, nil
	}
	for _, id := range invIDs {
		inv := invByID[id]
		if inv == nil || pkglife.IsTarnished(inv.TarnishedUntil, now) {
			continue
		}
		eq := eqByID[inv.EquipmentID]
		if eq == nil {
			continue
		}
		maps.Copy(out, eq.AiUnlockedPrivilege)
	}
	return out, nil
}

func (s *Service) loadEquippedMaps(ctx context.Context, slots *gen.LifeEquippedSlots) (
	map[int64]*gen.LifeInventory, map[int64]*gen.LifeEquipment, []int64, time.Time, error,
) {
	now := time.Now()
	if slots == nil || pkglife.IsTarnished(slots.TarnishedUntil, now) {
		return nil, nil, nil, now, nil
	}
	invIDs := slotInventoryIDs(slots)
	if len(invIDs) == 0 {
		return nil, nil, nil, now, nil
	}
	invByID, err := s.store.MapInventoryByIDs(ctx, invIDs)
	if err != nil {
		return nil, nil, nil, now, err
	}
	equipIDs := make([]int64, 0, len(invByID))
	equipSeen := make(map[int64]struct{}, len(invByID))
	for _, id := range invIDs {
		inv := invByID[id]
		if inv == nil || pkglife.IsTarnished(inv.TarnishedUntil, now) {
			continue
		}
		if _, ok := equipSeen[inv.EquipmentID]; ok {
			continue
		}
		equipSeen[inv.EquipmentID] = struct{}{}
		equipIDs = append(equipIDs, inv.EquipmentID)
	}
	eqByID, err := s.store.MapEquipmentByIDs(ctx, equipIDs)
	if err != nil {
		return nil, nil, nil, now, err
	}
	return invByID, eqByID, invIDs, now, nil
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
		return 0, lifeNotFound("quest not found")
	}
	return s.dropChanceForQuest(ctx, p, q)
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
		return lifeNotFound("ai context missing after ensure")
	}
	rate := pkglife.BlendCompletionRate(ai.HistoricalCompletionRate, success)
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
