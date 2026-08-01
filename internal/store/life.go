package store

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventoutbox"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeachievement"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeachievementprogress"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeachievementunlock"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactiondependency"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactionlog"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactionspec"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeadjudication"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeaicontext"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifecharacteristic"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeequipment"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeequippedslots"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeevidence"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifegoal"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeinventory"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeloottable"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeplannode"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeprofile"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifequest"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeskill"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Life lore outbox event type (payload["type"]).
const LifeLoreRequestedType = "life.inventory.lore_requested"

// LifeStore persists Life domain entities.
type LifeStore struct {
	client *gen.Client
	inTx   bool
}

// LifePlanActionSpecInput is the write shape for action metadata.
type LifePlanActionSpecInput struct {
	TaskType              string
	TrackingMode          string
	IsRepeatable          bool
	RepeatTrigger         string
	SuggestedCadence      string
	IsIdentityBuilding    bool
	Reason                string
	NeedsUserConfirmation bool
	Difficulty            string
	BaseExpReward         int
	BaseGoldReward        int
	ConfirmedAt           *time.Time
}

// LifeCreatePlanNodeInput is the write shape for creating one plan node.
type LifeCreatePlanNodeInput struct {
	ProfileID             int64
	ParentID              *int64
	NodeType              string
	Title                 string
	Description           string
	Status                string
	SortOrder             int
	ActionSpec            *LifePlanActionSpecInput
	DependencyPlanNodeIDs []int64
}

// LifeEvidenceInput is the write shape for quest evidence.
type LifeEvidenceInput struct {
	ProfileID  int64
	QuestID    *int64
	SourceType string
	Content    string
	SourceURL  string
	Summary    string
}

// LifeAdjudicationInput is the write shape for a suggested quest ruling.
type LifeAdjudicationInput struct {
	ProfileID          int64
	QuestID            int64
	Status             string
	Verdict            string
	Reason             string
	SuggestedExp       int
	SuggestedGold      int
	SuggestedNextSteps []string
	EvidenceSnapshot   []map[string]any
}

// NewLifeStore creates a LifeStore with the given ent client.
func NewLifeStore(client *gen.Client) *LifeStore {
	return &LifeStore{client: client}
}

// LifeStoreFromDB returns a LifeStore using the global database client.
func LifeStoreFromDB() *LifeStore {
	return NewLifeStore(ClientFromDB())
}

// Client returns the underlying ent client for transactional use-cases.
func (s *LifeStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// WithTx runs a callback inside one ent transaction.
func (s *LifeStore) WithTx(ctx context.Context, fn func(*LifeStore) error) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	if s.inTx {
		return fn(s)
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("life: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := fn(&LifeStore{client: tx.Client(), inTx: true}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("life: commit tx: %w", err)
	}
	committed = true
	return nil
}

func (s *LifeStore) ready() bool {
	return s != nil && s.client != nil
}

func lifePageBounds(limit, offset int) (lim, off int, ok bool) {
	if limit <= 0 {
		return 0, 0, false
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset, true
}

// GetProfileByUserID returns the life profile for a platform user id.
func (s *LifeStore) GetProfileByUserID(ctx context.Context, userID string) (*gen.LifeProfile, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeProfile.Query().Where(lifeprofile.UserIDEQ(userID)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("life: get profile: %w", err)
	}
	return row, nil
}

// GetProfileByID returns a profile by primary key.
func (s *LifeStore) GetProfileByID(ctx context.Context, id int64) (*gen.LifeProfile, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeProfile.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("life: get profile by id: %w", err)
	}
	return row, nil
}

// CreateProfile inserts a new life profile.
func (s *LifeStore) CreateProfile(ctx context.Context, userID, nickname, classType string) (*gen.LifeProfile, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	if classType == "" {
		classType = "Architect"
	}
	row, err := s.client.LifeProfile.Create().
		SetFlag(types.Id()).
		SetUserID(userID).
		SetNickname(nickname).
		SetClassType(classType).
		SetPityByTier(map[string]int{}).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create profile: %w", err)
	}
	return row, nil
}

// UpdateProfileStats updates level/exp/gold/pity on a profile.
func (s *LifeStore) UpdateProfileStats(ctx context.Context, id int64, level int, exp int64, gold int, pity map[string]int) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	u := s.client.LifeProfile.UpdateOneID(id).
		SetLevel(level).
		SetExp(exp).
		SetGold(gold)
	if pity != nil {
		u = u.SetPityByTier(pity)
	}
	_, err := u.Save(ctx)
	if err != nil {
		return fmt.Errorf("life: update profile stats: %w", err)
	}
	return nil
}

// UpdateProfileClass sets class_type.
func (s *LifeStore) UpdateProfileClass(ctx context.Context, id int64, classType string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeProfile.UpdateOneID(id).SetClassType(classType).Save(ctx)
	if err != nil {
		return fmt.Errorf("life: update class: %w", err)
	}
	return nil
}

// CreateCharacteristic inserts a characteristic row.
func (s *LifeStore) CreateCharacteristic(ctx context.Context, profileID int64, code, name string) (*gen.LifeCharacteristic, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	row, err := s.client.LifeCharacteristic.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(profileID).
		SetCode(code).
		SetName(name).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create characteristic: %w", err)
	}
	return row, nil
}

// ListCharacteristics returns characteristics for a profile.
func (s *LifeStore) ListCharacteristics(ctx context.Context, profileID int64) ([]*gen.LifeCharacteristic, error) {
	if !s.ready() {
		return nil, nil
	}
	return s.client.LifeCharacteristic.Query().
		Where(lifecharacteristic.LifeProfileIDEQ(profileID)).
		All(ctx)
}

// GetCharacteristic returns one characteristic by id.
func (s *LifeStore) GetCharacteristic(ctx context.Context, id int64) (*gen.LifeCharacteristic, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeCharacteristic.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// UpdateCharacteristicStats updates level/exp.
func (s *LifeStore) UpdateCharacteristicStats(ctx context.Context, id int64, level int, exp int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeCharacteristic.UpdateOneID(id).SetLevel(level).SetCurrentExp(exp).Save(ctx)
	return err
}

// GetSkillByName returns a skill by profile + name.
func (s *LifeStore) GetSkillByName(ctx context.Context, profileID int64, name string) (*gen.LifeSkill, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeSkill.Query().
		Where(lifeskill.LifeProfileIDEQ(profileID), lifeskill.NameEQ(name)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// CreateSkill inserts a skill.
func (s *LifeStore) CreateSkill(ctx context.Context, profileID, characteristicID int64, name string, ratio float64) (*gen.LifeSkill, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	if ratio <= 0 {
		ratio = 0.5
	}
	row, err := s.client.LifeSkill.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(profileID).
		SetCharacteristicID(characteristicID).
		SetName(name).
		SetExpToCharacteristicRatio(ratio).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create skill: %w", err)
	}
	return row, nil
}

// ListSkills returns skills for a profile.
func (s *LifeStore) ListSkills(ctx context.Context, profileID int64) ([]*gen.LifeSkill, error) {
	if !s.ready() {
		return nil, nil
	}
	return s.client.LifeSkill.Query().Where(lifeskill.LifeProfileIDEQ(profileID)).All(ctx)
}

// GetSkill returns a skill by id.
func (s *LifeStore) GetSkill(ctx context.Context, id int64) (*gen.LifeSkill, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeSkill.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// UpdateSkillStats updates skill level/exp.
func (s *LifeStore) UpdateSkillStats(ctx context.Context, id int64, level int, exp int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeSkill.UpdateOneID(id).SetLevel(level).SetCurrentExp(exp).Save(ctx)
	return err
}

// EnsureAIContext creates AI context if missing.
func (s *LifeStore) EnsureAIContext(ctx context.Context, profileID int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeAIContext.Query().Where(lifeaicontext.LifeProfileIDEQ(profileID)).Only(ctx)
	if err == nil {
		return nil
	}
	if !gen.IsNotFound(err) {
		return err
	}
	_, err = s.client.LifeAIContext.Create().
		SetLifeProfileID(profileID).
		SetRecentMoodAndBurnout(map[string]any{}).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("life: ensure ai context: %w", err)
	}
	return nil
}

// GetAIContext returns the AI context row for a profile.
func (s *LifeStore) GetAIContext(ctx context.Context, profileID int64) (*gen.LifeAIContext, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeAIContext.Query().Where(lifeaicontext.LifeProfileIDEQ(profileID)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// UpdateAIContext writes completion rate, mood, and personality.
func (s *LifeStore) UpdateAIContext(ctx context.Context, profileID int64, rate float64, mood map[string]any, personality string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	u := s.client.LifeAIContext.Update().Where(lifeaicontext.LifeProfileIDEQ(profileID)).
		SetHistoricalCompletionRate(rate)
	if mood != nil {
		u = u.SetRecentMoodAndBurnout(mood)
	}
	if personality != "" {
		u = u.SetAiDmPersonality(personality)
	}
	_, err := u.Save(ctx)
	return err
}

// CreateGoal inserts an active PARA goal. areaID is optional parent Area for Project/Resource.
func (s *LifeStore) CreateGoal(ctx context.Context, profileID int64, title, category string, areaID *int64) (*gen.LifeGoal, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	b := s.client.LifeGoal.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(profileID).
		SetTitle(title).
		SetCategory(category).
		SetStatus("Active")
	if areaID != nil {
		b = b.SetAreaID(*areaID)
	}
	row, err := b.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create goal: %w", err)
	}
	return row, nil
}

// GetGoalByFlag returns a goal by flag for a profile.
func (s *LifeStore) GetGoalByFlag(ctx context.Context, profileID int64, flag string) (*gen.LifeGoal, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeGoal.Query().
		Where(lifegoal.LifeProfileIDEQ(profileID), lifegoal.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// CreatePlanNode inserts one life plan node and its action spec when needed.
func (s *LifeStore) CreatePlanNode(ctx context.Context, in LifeCreatePlanNodeInput) (*gen.LifePlanNode, *gen.LifeActionSpec, error) {
	if !s.ready() {
		return nil, nil, fmt.Errorf("life: store not available")
	}
	var row *gen.LifePlanNode
	var spec *gen.LifeActionSpec
	err := s.WithTx(ctx, func(txStore *LifeStore) error {
		var innerErr error
		row, innerErr = createPlanNodeWithClient(ctx, txStore.client, in)
		if innerErr != nil {
			return innerErr
		}
		if in.ActionSpec != nil {
			if in.ActionSpec.TaskType == "checkpoint" && len(in.DependencyPlanNodeIDs) == 0 {
				return fmt.Errorf("life: checkpoint dependencies required")
			}
			if in.ActionSpec.TaskType != "checkpoint" && len(in.DependencyPlanNodeIDs) > 0 {
				return fmt.Errorf("life: only checkpoint actions can have dependencies")
			}
			spec, innerErr = createActionSpecWithClient(ctx, txStore.client, row.ID, *in.ActionSpec)
			if innerErr != nil {
				return innerErr
			}
			if len(in.DependencyPlanNodeIDs) > 0 {
				if innerErr := txStore.createActionDependencies(ctx, in.ProfileID, row.ID, in.DependencyPlanNodeIDs); innerErr != nil {
					return innerErr
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return row, spec, nil
}

// ListPlanNodes returns plan nodes for one profile in stable tree order.
func (s *LifeStore) ListPlanNodes(ctx context.Context, profileID int64) ([]*gen.LifePlanNode, error) {
	if !s.ready() {
		return nil, nil
	}
	rows, err := s.client.LifePlanNode.Query().
		Where(lifeplannode.LifeProfileIDEQ(profileID)).
		Order(
			gen.Asc(lifeplannode.FieldParentID),
			gen.Asc(lifeplannode.FieldSortOrder),
			gen.Asc(lifeplannode.FieldCreatedAt),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list plan nodes: %w", err)
	}
	return rows, nil
}

// ListActionSpecs returns all action specs keyed by plan node id.
func (s *LifeStore) ListActionSpecs(ctx context.Context, profileID int64) ([]*gen.LifeActionSpec, error) {
	if !s.ready() {
		return nil, nil
	}
	nodes, err := s.ListPlanNodes(ctx, profileID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(nodes))
	for _, row := range nodes {
		ids = append(ids, row.ID)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.client.LifeActionSpec.Query().
		Where(lifeactionspec.PlanNodeIDIn(ids...)).
		Order(gen.Asc(lifeactionspec.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list action specs: %w", err)
	}
	return rows, nil
}

// GetPlanNodeByFlag returns one plan node scoped to profile.
func (s *LifeStore) GetPlanNodeByFlag(ctx context.Context, profileID int64, flag string) (*gen.LifePlanNode, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifePlanNode.Query().
		Where(lifeplannode.LifeProfileIDEQ(profileID), lifeplannode.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("life: get plan node: %w", err)
	}
	return row, nil
}

// GetActionSpecByPlanNodeID returns one action spec by plan node id.
func (s *LifeStore) GetActionSpecByPlanNodeID(ctx context.Context, planNodeID int64) (*gen.LifeActionSpec, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeActionSpec.Query().
		Where(lifeactionspec.PlanNodeIDEQ(planNodeID)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("life: get action spec: %w", err)
	}
	return row, nil
}

// ConfirmHabitAction promotes a habit candidate to a confirmed habit.
func (s *LifeStore) ConfirmHabitAction(ctx context.Context, planNodeID int64) (*gen.LifeActionSpec, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	spec, err := s.GetActionSpecByPlanNodeID(ctx, planNodeID)
	if err != nil {
		return nil, err
	}
	if spec == nil {
		return nil, nil
	}
	now := time.Now()
	row, err := s.client.LifeActionSpec.UpdateOneID(spec.ID).
		SetTaskType("habit").
		SetNeedsUserConfirmation(false).
		SetConfirmedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: confirm habit: %w", err)
	}
	return row, nil
}

// UpdatePlanNode edits mutable fields on one plan node.
func (s *LifeStore) UpdatePlanNode(ctx context.Context, id int64, title, description, status string, sortOrder int) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	u := s.client.LifePlanNode.UpdateOneID(id).
		SetTitle(title).
		SetDescription(description).
		SetSortOrder(sortOrder)
	if status != "" {
		u = u.SetStatus(status)
	}
	if _, err := u.Save(ctx); err != nil {
		return fmt.Errorf("life: update plan node: %w", err)
	}
	return nil
}

// DeletePlanNode removes one plan node and its descendants.
func (s *LifeStore) DeletePlanNode(ctx context.Context, profileID, id int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	rows, err := s.ListPlanNodes(ctx, profileID)
	if err != nil {
		return err
	}
	toDelete := collectPlanDescendantIDs(rows, id)
	if len(toDelete) == 0 {
		return nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("life: begin delete plan tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.LifeActionSpec.Delete().Where(lifeactionspec.PlanNodeIDIn(toDelete...)).Exec(ctx); err != nil {
		return fmt.Errorf("life: delete action specs: %w", err)
	}
	if _, err := tx.LifeActionDependency.Delete().
		Where(
			lifeactiondependency.Or(
				lifeactiondependency.ActionPlanNodeIDIn(toDelete...),
				lifeactiondependency.DependsOnPlanNodeIDIn(toDelete...),
			),
		).
		Exec(ctx); err != nil {
		return fmt.Errorf("life: delete action dependencies: %w", err)
	}
	if _, err := tx.LifePlanNode.Delete().Where(lifeplannode.IDIn(toDelete...)).Exec(ctx); err != nil {
		return fmt.Errorf("life: delete plan nodes: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("life: commit delete plan: %w", err)
	}
	committed = true
	return nil
}

// UpdateGoalStatus sets goal status (Active / Paused / Completed).
func (s *LifeStore) UpdateGoalStatus(ctx context.Context, id int64, status string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeGoal.UpdateOneID(id).SetStatus(status).Save(ctx)
	return err
}

// MarkQuestStatus sets quest status and completed_at when Completed.
func (s *LifeStore) MarkQuestStatus(ctx context.Context, id int64, status string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	u := s.client.LifeQuest.UpdateOneID(id).SetStatus(status)
	if status == "Completed" {
		now := time.Now()
		u = u.SetCompletedAt(now)
	}
	_, err := u.Save(ctx)
	return err
}

// SetInventoryTarnishedUntil sets or clears rust on an inventory row.
func (s *LifeStore) SetInventoryTarnishedUntil(ctx context.Context, id int64, until *time.Time) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	u := s.client.LifeInventory.UpdateOneID(id)
	if until == nil {
		u = u.ClearTarnishedUntil()
	} else {
		u = u.SetTarnishedUntil(*until)
	}
	_, err := u.Save(ctx)
	return err
}

// SetEquippedSlotsTarnishedUntil sets or clears rust on equipped slots.
func (s *LifeStore) SetEquippedSlotsTarnishedUntil(ctx context.Context, profileID int64, until *time.Time) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	u := s.client.LifeEquippedSlots.Update().Where(lifeequippedslots.LifeProfileIDEQ(profileID))
	if until == nil {
		u = u.ClearTarnishedUntil()
	} else {
		u = u.SetTarnishedUntil(*until)
	}
	_, err := u.Save(ctx)
	return err
}

// EnsureEquippedSlots creates empty slots row if missing.
func (s *LifeStore) EnsureEquippedSlots(ctx context.Context, profileID int64) (*gen.LifeEquippedSlots, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	row, err := s.client.LifeEquippedSlots.Query().Where(lifeequippedslots.LifeProfileIDEQ(profileID)).Only(ctx)
	if err == nil {
		return row, nil
	}
	if !gen.IsNotFound(err) {
		return nil, err
	}
	return s.client.LifeEquippedSlots.Create().SetLifeProfileID(profileID).Save(ctx)
}

// GetEquippedSlots returns equipped slots for a profile.
func (s *LifeStore) GetEquippedSlots(ctx context.Context, profileID int64) (*gen.LifeEquippedSlots, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeEquippedSlots.Query().Where(lifeequippedslots.LifeProfileIDEQ(profileID)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// SetEquippedSlot writes one slot inventory id (nil clears).
func (s *LifeStore) SetEquippedSlot(ctx context.Context, profileID int64, slotField string, inventoryID *int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	slots, err := s.EnsureEquippedSlots(ctx, profileID)
	if err != nil {
		return err
	}
	u := s.client.LifeEquippedSlots.UpdateOneID(slots.ID)
	if err := applyEquippedSlotUpdate(u, slotField, inventoryID); err != nil {
		return err
	}
	_, err = u.Save(ctx)
	return err
}

func applyEquippedSlotUpdate(u *gen.LifeEquippedSlotsUpdateOne, slotField string, inventoryID *int64) error {
	setters := map[string]func(*gen.LifeEquippedSlotsUpdateOne, *int64) *gen.LifeEquippedSlotsUpdateOne{
		"head_slot": func(u *gen.LifeEquippedSlotsUpdateOne, id *int64) *gen.LifeEquippedSlotsUpdateOne {
			if id == nil {
				return u.ClearHeadSlot()
			}
			return u.SetHeadSlot(*id)
		},
		"weapon_slot": func(u *gen.LifeEquippedSlotsUpdateOne, id *int64) *gen.LifeEquippedSlotsUpdateOne {
			if id == nil {
				return u.ClearWeaponSlot()
			}
			return u.SetWeaponSlot(*id)
		},
		"armor_slot": func(u *gen.LifeEquippedSlotsUpdateOne, id *int64) *gen.LifeEquippedSlotsUpdateOne {
			if id == nil {
				return u.ClearArmorSlot()
			}
			return u.SetArmorSlot(*id)
		},
		"shoes_slot": func(u *gen.LifeEquippedSlotsUpdateOne, id *int64) *gen.LifeEquippedSlotsUpdateOne {
			if id == nil {
				return u.ClearShoesSlot()
			}
			return u.SetShoesSlot(*id)
		},
		"accessory_slot": func(u *gen.LifeEquippedSlotsUpdateOne, id *int64) *gen.LifeEquippedSlotsUpdateOne {
			if id == nil {
				return u.ClearAccessorySlot()
			}
			return u.SetAccessorySlot(*id)
		},
		"artifact_slot": func(u *gen.LifeEquippedSlotsUpdateOne, id *int64) *gen.LifeEquippedSlotsUpdateOne {
			if id == nil {
				return u.ClearArtifactSlot()
			}
			return u.SetArtifactSlot(*id)
		},
	}
	fn, ok := setters[slotField]
	if !ok {
		return fmt.Errorf("life: unknown slot %q", slotField)
	}
	fn(u, inventoryID)
	return nil
}

// ListGoals returns goals for a profile, optional status filter.
func (s *LifeStore) ListGoals(ctx context.Context, profileID int64, status string) ([]*gen.LifeGoal, error) {
	if !s.ready() {
		return nil, nil
	}
	q := s.client.LifeGoal.Query().Where(lifegoal.LifeProfileIDEQ(profileID)).Order(gen.Desc(lifegoal.FieldCreatedAt))
	if status != "" {
		q = q.Where(lifegoal.StatusEQ(status))
	}
	return q.All(ctx)
}

// CreateQuest inserts a quest.
func (s *LifeStore) CreateQuest(ctx context.Context, q *gen.LifeQuest) (*gen.LifeQuest, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	b := s.client.LifeQuest.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(q.LifeProfileID).
		SetSkillID(q.SkillID).
		SetTitle(q.Title).
		SetPrompt(q.Prompt).
		SetType(q.Type).
		SetAiEvaluatedDifficulty(q.AiEvaluatedDifficulty).
		SetAiEvaluatedFear(q.AiEvaluatedFear).
		SetBaseExpReward(q.BaseExpReward).
		SetBaseGoldReward(q.BaseGoldReward).
		SetDropTier(q.DropTier).
		SetStatus("Pending")
	if q.GoalID != nil {
		b = b.SetGoalID(*q.GoalID)
	}
	row, err := b.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create quest: %w", err)
	}
	return row, nil
}

// ListQuests lists quests for a profile, optional status filter.
func (s *LifeStore) ListQuests(ctx context.Context, profileID int64, status string) ([]*gen.LifeQuest, error) {
	rows, _, err := s.ListQuestsPage(ctx, profileID, status, 0, 0)
	return rows, err
}

// ListQuestsPage returns a page of quests and the total matching count.
// Completed quests are ordered by completed_at descending; others by created_at descending.
// A non-positive limit returns all matching rows (offset ignored).
func (s *LifeStore) ListQuestsPage(ctx context.Context, profileID int64, status string, limit, offset int) ([]*gen.LifeQuest, int, error) {
	if !s.ready() {
		return nil, 0, nil
	}
	q := s.client.LifeQuest.Query().Where(lifequest.LifeProfileIDEQ(profileID))
	if status != "" {
		q = q.Where(lifequest.StatusEQ(status))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: count quests: %w", err)
	}
	if status == "Completed" {
		q = q.Order(gen.Desc(lifequest.FieldCompletedAt), gen.Desc(lifequest.FieldID))
	} else {
		q = q.Order(gen.Desc(lifequest.FieldCreatedAt), gen.Desc(lifequest.FieldID))
	}
	if lim, off, ok := lifePageBounds(limit, offset); ok {
		q = q.Limit(lim).Offset(off)
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: list quests page: %w", err)
	}
	return rows, total, nil
}

// GetQuestByFlag returns a quest by flag scoped to profile.
func (s *LifeStore) GetQuestByFlag(ctx context.Context, profileID int64, flag string) (*gen.LifeQuest, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeQuest.Query().
		Where(lifequest.LifeProfileIDEQ(profileID), lifequest.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// CreateEvidence inserts one quest evidence row.
func (s *LifeStore) CreateEvidence(ctx context.Context, in LifeEvidenceInput) (*gen.LifeEvidence, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	b := s.client.LifeEvidence.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetSourceType(in.SourceType).
		SetContent(in.Content).
		SetSourceURL(in.SourceURL).
		SetSummary(in.Summary)
	if in.QuestID != nil {
		b = b.SetQuestID(*in.QuestID)
	}
	row, err := b.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create evidence: %w", err)
	}
	return row, nil
}

// ListEvidenceByQuest returns recent evidence rows for one quest.
func (s *LifeStore) ListEvidenceByQuest(ctx context.Context, profileID, questID int64) ([]*gen.LifeEvidence, error) {
	if !s.ready() {
		return nil, nil
	}
	return s.client.LifeEvidence.Query().
		Where(
			lifeevidence.LifeProfileIDEQ(profileID),
			lifeevidence.QuestIDEQ(questID),
		).
		Order(gen.Desc(lifeevidence.FieldCreatedAt)).
		All(ctx)
}

// CreateAdjudication inserts one suggested quest ruling.
func (s *LifeStore) CreateAdjudication(ctx context.Context, in LifeAdjudicationInput) (*gen.LifeAdjudication, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	row, err := s.client.LifeAdjudication.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetQuestID(in.QuestID).
		SetStatus(in.Status).
		SetVerdict(in.Verdict).
		SetReason(in.Reason).
		SetSuggestedExp(in.SuggestedExp).
		SetSuggestedGold(in.SuggestedGold).
		SetSuggestedNextSteps(in.SuggestedNextSteps).
		SetEvidenceSnapshot(in.EvidenceSnapshot).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create adjudication: %w", err)
	}
	return row, nil
}

// GetLatestAdjudicationByQuest returns the newest ruling for one quest.
func (s *LifeStore) GetLatestAdjudicationByQuest(ctx context.Context, profileID, questID int64) (*gen.LifeAdjudication, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeAdjudication.Query().
		Where(
			lifeadjudication.LifeProfileIDEQ(profileID),
			lifeadjudication.QuestIDEQ(questID),
		).
		Order(gen.Desc(lifeadjudication.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("life: get adjudication: %w", err)
	}
	return row, nil
}

// GetAdjudicationByFlag returns a quest ruling by flag scoped to profile.
func (s *LifeStore) GetAdjudicationByFlag(ctx context.Context, profileID int64, flag string) (*gen.LifeAdjudication, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeAdjudication.Query().
		Where(
			lifeadjudication.LifeProfileIDEQ(profileID),
			lifeadjudication.FlagEQ(flag),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("life: get adjudication by flag: %w", err)
	}
	return row, nil
}

// MarkAdjudicationApplied records that the ruling was accepted.
func (s *LifeStore) MarkAdjudicationApplied(ctx context.Context, id int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	now := time.Now()
	_, err := s.client.LifeAdjudication.UpdateOneID(id).
		SetStatus("applied").
		SetAppliedAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("life: apply adjudication: %w", err)
	}
	return nil
}

// MarkQuestCompleted sets status Completed.
func (s *LifeStore) MarkQuestCompleted(ctx context.Context, id int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	now := time.Now()
	_, err := s.client.LifeQuest.UpdateOneID(id).
		SetStatus("Completed").
		SetCompletedAt(now).
		Save(ctx)
	return err
}

// UpsertEquipment creates equipment if flag missing.
func (s *LifeStore) UpsertEquipment(ctx context.Context, flag, name, rarity, slotType, lore string, buffs, priv map[string]any) (*gen.LifeEquipment, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	existing, err := s.client.LifeEquipment.Query().Where(lifeequipment.FlagEQ(flag)).Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !gen.IsNotFound(err) {
		return nil, err
	}
	if buffs == nil {
		buffs = map[string]any{}
	}
	if priv == nil {
		priv = map[string]any{}
	}
	return s.client.LifeEquipment.Create().
		SetFlag(flag).
		SetName(name).
		SetRarity(rarity).
		SetSlotType(slotType).
		SetStatBuffs(buffs).
		SetAiUnlockedPrivilege(priv).
		SetAiLoreText(lore).
		Save(ctx)
}

// GetEquipmentByFlag returns equipment by flag.
func (s *LifeStore) GetEquipmentByFlag(ctx context.Context, flag string) (*gen.LifeEquipment, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeEquipment.Query().Where(lifeequipment.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetEquipment returns equipment by id.
func (s *LifeStore) GetEquipment(ctx context.Context, id int64) (*gen.LifeEquipment, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeEquipment.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// UpsertLootTable creates or updates a loot table by tier.
func (s *LifeStore) UpsertLootTable(ctx context.Context, tier string, chance float64, pool []string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	existing, err := s.client.LifeLootTable.Query().Where(lifeloottable.DropTierEQ(tier)).Only(ctx)
	if err == nil {
		_, err = s.client.LifeLootTable.UpdateOneID(existing.ID).
			SetBaseDropChance(chance).
			SetItemPoolFlags(pool).
			Save(ctx)
		return err
	}
	if !gen.IsNotFound(err) {
		return err
	}
	_, err = s.client.LifeLootTable.Create().
		SetDropTier(tier).
		SetBaseDropChance(chance).
		SetItemPoolFlags(pool).
		Save(ctx)
	return err
}

// UpsertAchievement inserts or updates a catalog achievement by flag.
func (s *LifeStore) UpsertAchievement(ctx context.Context, in LifeAchievementUpsert) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	kind := in.Kind
	if kind == "" {
		kind = pkglife.AchievementKindFirst
	}
	threshold := max(in.Threshold, 1)
	existing, err := s.client.LifeAchievement.Query().Where(lifeachievement.FlagEQ(in.Flag)).Only(ctx)
	if err == nil {
		_, err = s.client.LifeAchievement.UpdateOneID(existing.ID).
			SetName(in.Name).
			SetDescription(in.Description).
			SetActive(in.Active).
			SetKind(kind).
			SetQuestType(in.QuestType).
			SetDifficulty(in.Difficulty).
			SetThreshold(threshold).
			SetSortOrder(in.SortOrder).
			Save(ctx)
		return err
	}
	if !gen.IsNotFound(err) {
		return err
	}
	_, err = s.client.LifeAchievement.Create().
		SetFlag(in.Flag).
		SetName(in.Name).
		SetDescription(in.Description).
		SetActive(in.Active).
		SetKind(kind).
		SetQuestType(in.QuestType).
		SetDifficulty(in.Difficulty).
		SetThreshold(threshold).
		SetSortOrder(in.SortOrder).
		Save(ctx)
	return err
}

// ListAchievements returns the achievement catalog ordered by sort_order.
func (s *LifeStore) ListAchievements(ctx context.Context) ([]*gen.LifeAchievement, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	return s.client.LifeAchievement.Query().
		Order(lifeachievement.BySortOrder(), lifeachievement.ByFlag()).
		All(ctx)
}

// DeactivateAchievementsNotInFlags sets active=false for catalog rows whose flag is not listed.
func (s *LifeStore) DeactivateAchievementsNotInFlags(ctx context.Context, keepFlags []string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	q := s.client.LifeAchievement.Update().SetActive(false)
	if len(keepFlags) > 0 {
		q = q.Where(lifeachievement.FlagNotIn(keepFlags...))
	}
	_, err := q.Save(ctx)
	return err
}

// ListAchievementProgress returns progress rows for a profile.
func (s *LifeStore) ListAchievementProgress(ctx context.Context, profileID int64) ([]*gen.LifeAchievementProgress, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	return s.client.LifeAchievementProgress.Query().
		Where(lifeachievementprogress.LifeProfileIDEQ(profileID)).
		All(ctx)
}

// ListAchievementUnlocks returns unlock rows for a profile.
func (s *LifeStore) ListAchievementUnlocks(ctx context.Context, profileID int64) ([]*gen.LifeAchievementUnlock, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	return s.client.LifeAchievementUnlock.Query().
		Where(lifeachievementunlock.LifeProfileIDEQ(profileID)).
		All(ctx)
}

// GetLootTable returns loot table by tier.
func (s *LifeStore) GetLootTable(ctx context.Context, tier string) (*gen.LifeLootTable, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeLootTable.Query().Where(lifeloottable.DropTierEQ(tier)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// CreateInventory inserts an inventory instance.
func (s *LifeStore) CreateInventory(ctx context.Context, profileID, equipmentID int64, questID *int64, loreStatus string) (*gen.LifeInventory, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	if loreStatus == "" {
		loreStatus = "none"
	}
	b := s.client.LifeInventory.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(profileID).
		SetEquipmentID(equipmentID).
		SetLoreStatus(loreStatus).
		SetInstanceBuffs(map[string]any{})
	if questID != nil {
		b = b.SetSourceQuestID(*questID)
	}
	row, err := b.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create inventory: %w", err)
	}
	return row, nil
}

// ListInventory lists inventory for a profile.
func (s *LifeStore) ListInventory(ctx context.Context, profileID int64) ([]*gen.LifeInventory, error) {
	rows, _, err := s.ListInventoryPage(ctx, profileID, 0, 0)
	return rows, err
}

// ListInventoryPage returns a page of inventory rows and the total count.
// A non-positive limit returns all matching rows (offset ignored).
func (s *LifeStore) ListInventoryPage(ctx context.Context, profileID int64, limit, offset int) ([]*gen.LifeInventory, int, error) {
	if !s.ready() {
		return nil, 0, nil
	}
	q := s.client.LifeInventory.Query().Where(lifeinventory.LifeProfileIDEQ(profileID))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: count inventory: %w", err)
	}
	q = q.Order(gen.Desc(lifeinventory.FieldAcquiredAt), gen.Desc(lifeinventory.FieldID))
	if lim, off, ok := lifePageBounds(limit, offset); ok {
		q = q.Limit(lim).Offset(off)
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: list inventory page: %w", err)
	}
	return rows, total, nil
}

// GetInventoryByFlag returns inventory by flag scoped to profile.
func (s *LifeStore) GetInventoryByFlag(ctx context.Context, profileID int64, flag string) (*gen.LifeInventory, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeInventory.Query().
		Where(lifeinventory.LifeProfileIDEQ(profileID), lifeinventory.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetInventory returns inventory by id.
func (s *LifeStore) GetInventory(ctx context.Context, id int64) (*gen.LifeInventory, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeInventory.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// UpdateInventoryLore sets instance lore fields and status.
func (s *LifeStore) UpdateInventoryLore(ctx context.Context, id int64, name, lore, status string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeInventory.UpdateOneID(id).
		SetInstanceName(name).
		SetInstanceLore(lore).
		SetLoreStatus(status).
		Save(ctx)
	return err
}

// UpdateInventoryLoreStatus sets only lore_status (keeps instance name/lore).
func (s *LifeStore) UpdateInventoryLoreStatus(ctx context.Context, id int64, status string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeInventory.UpdateOneID(id).
		SetLoreStatus(status).
		Save(ctx)
	return err
}

// CreateActionLog inserts an action log row.
func (s *LifeStore) CreateActionLog(ctx context.Context, profileID, questID int64, exp, gold int, invID *int64, dice *float64) (*gen.LifeActionLog, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	b := s.client.LifeActionLog.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(profileID).
		SetQuestID(questID).
		SetSourceType("quest").
		SetGainedExp(exp).
		SetGainedGold(gold)
	if invID != nil {
		b = b.SetDroppedInventoryID(*invID)
	}
	if dice != nil {
		b = b.SetDiceRollResult(*dice)
	}
	return b.Save(ctx)
}

// ListActionLogs returns recent action logs.
func (s *LifeStore) ListActionLogs(ctx context.Context, profileID int64, limit int) ([]*gen.LifeActionLog, error) {
	rows, _, err := s.ListActionLogsPage(ctx, profileID, limit, 0)
	return rows, err
}

// ListActionLogsPage returns a page of action logs and the total count.
// A non-positive limit returns all matching rows (offset ignored).
func (s *LifeStore) ListActionLogsPage(ctx context.Context, profileID int64, limit, offset int) ([]*gen.LifeActionLog, int, error) {
	if !s.ready() {
		return nil, 0, nil
	}
	q := s.client.LifeActionLog.Query().Where(lifeactionlog.LifeProfileIDEQ(profileID))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: count action logs: %w", err)
	}
	q = q.Order(gen.Desc(lifeactionlog.FieldCreatedAt), gen.Desc(lifeactionlog.FieldID))
	if lim, off, ok := lifePageBounds(limit, offset); ok {
		q = q.Limit(lim).Offset(off)
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: list action logs page: %w", err)
	}
	return rows, total, nil
}

// AppendLoreOutbox writes an unpublished outbox row for lore generation.
func (s *LifeStore) AppendLoreOutbox(ctx context.Context, profileID, inventoryID int64) (string, error) {
	if !s.ready() {
		return "", fmt.Errorf("life: store not available")
	}
	eventID := types.Id()
	payload := map[string]any{
		"event_id":        eventID,
		"type":            LifeLoreRequestedType,
		"life_profile_id": profileID,
		"inventory_id":    inventoryID,
	}
	_, err := s.client.EventOutbox.Create().
		SetEventID(eventID).
		SetPayload(payload).
		SetPublished(false).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("life: append lore outbox: %w", err)
	}
	return eventID, nil
}

// ListPendingLoreOutbox returns unpublished lore outbox rows.
func (s *LifeStore) ListPendingLoreOutbox(ctx context.Context, limit int) ([]*gen.EventOutbox, error) {
	if !s.ready() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.client.EventOutbox.Query().
		Where(eventoutbox.PublishedEQ(false)).
		Order(gen.Asc(eventoutbox.FieldCreatedAt)).
		Limit(limit * 5).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*gen.EventOutbox, 0)
	for _, row := range rows {
		raw, err := sonic.Marshal(row.Payload)
		if err != nil {
			continue
		}
		var payload map[string]any
		if err := sonic.Unmarshal(raw, &payload); err != nil {
			continue
		}
		t, ok := payload["type"].(string)
		if ok && t == LifeLoreRequestedType {
			out = append(out, row)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// MarkOutboxPublished marks an outbox event published.
func (s *LifeStore) MarkOutboxPublished(ctx context.Context, eventID string) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.EventOutbox.Update().
		Where(eventoutbox.EventID(eventID)).
		SetPublished(true).
		Save(ctx)
	return err
}

// GetQuest returns a quest by id.
func (s *LifeStore) GetQuest(ctx context.Context, id int64) (*gen.LifeQuest, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeQuest.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// UpdateGoal updates title, category, and optional Area parent for a goal.
// A nil areaID clears the parent link.
func (s *LifeStore) UpdateGoal(ctx context.Context, id int64, title, category string, areaID *int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	u := s.client.LifeGoal.UpdateOneID(id).SetTitle(title).SetCategory(category)
	if areaID == nil {
		u = u.ClearAreaID()
	} else {
		u = u.SetAreaID(*areaID)
	}
	_, err := u.Save(ctx)
	return err
}

// ClearGoalAreaRefs clears area_id on goals that point at the given Area goal id.
func (s *LifeStore) ClearGoalAreaRefs(ctx context.Context, areaID int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeGoal.Update().
		Where(lifegoal.AreaIDEQ(areaID)).
		ClearAreaID().
		Save(ctx)
	return err
}

// DeleteGoal removes a goal row.
func (s *LifeStore) DeleteGoal(ctx context.Context, id int64) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	return s.client.LifeGoal.DeleteOneID(id).Exec(ctx)
}

// LifeCompletePersist is the write-set for an atomic quest completion.
type LifeCompletePersist struct {
	ProfileID   int64
	QuestID     int64
	SkillID     int64
	CharID      int64
	SkillLevel  int
	SkillExp    int64
	CharLevel   int
	CharExp     int64
	ProfLevel   int
	ProfExp     int64
	ProfGold    int
	Pity        map[string]int
	RustInvIDs  []int64
	DropEquipID int64 // 0 = no drop; set by ResolveLootInTx when enabled
	DropQuestID int64
	LoreStatus  string
	NeedLore    bool
	ActionExp   int
	ActionGold  int
	Dice        float64
	QuestType   string
	Difficulty  string
	// DailyRespawn clones a pending Daily quest after completion when non-nil.
	DailyRespawn *gen.LifeQuest

	// ResolveLootInTx rolls loot from the live profile pity inside the completion transaction.
	ResolveLootInTx  bool
	DropTier         string
	LootBaseChance   float64
	LootPool         []string
	ProfileBonus     float64
	EquippedDropRate float64
}

// LifeCompleteResult is returned from PersistCompleteQuest.
type LifeCompleteResult struct {
	Inventory     *gen.LifeInventory
	Equipment     *gen.LifeEquipment
	NewlyUnlocked []*gen.LifeAchievement
	Dice          float64
	Loot          pkglife.LootResult
}

// LifeAchievementUpsert is the seed write shape for one catalog achievement.
type LifeAchievementUpsert struct {
	Flag        string
	Name        string
	Description string
	Active      bool
	Kind        string
	QuestType   string
	Difficulty  string
	Threshold   int
	SortOrder   int
}

// PersistCompleteQuest applies cascade, loot inventory, action log, and rust clear in one transaction.
func (s *LifeStore) PersistCompleteQuest(ctx context.Context, in LifeCompletePersist) (*LifeCompleteResult, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: begin complete tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	loot, err := resolveCompleteLoot(ctx, tx, &in)
	if err != nil {
		return nil, err
	}
	if err := persistCompleteCascade(ctx, tx, in); err != nil {
		return nil, err
	}
	now := time.Now()
	if _, err := tx.LifeQuest.UpdateOneID(in.QuestID).SetStatus("Completed").SetCompletedAt(now).Save(ctx); err != nil {
		return nil, fmt.Errorf("life: mark completed: %w", err)
	}
	if err := clearCompleteRust(ctx, tx, in); err != nil {
		return nil, err
	}
	out, invID, err := persistCompleteDrop(ctx, tx, in, now)
	if err != nil {
		return nil, err
	}
	if err := persistCompleteActionLog(ctx, tx, in, invID); err != nil {
		return nil, err
	}
	if err := persistDailyRespawn(ctx, tx, in.DailyRespawn); err != nil {
		return nil, err
	}
	unlocked, err := persistCompleteAchievements(ctx, tx, in.ProfileID, in.QuestType, in.Difficulty)
	if err != nil {
		return nil, err
	}
	out.NewlyUnlocked = unlocked
	out.Dice = in.Dice
	out.Loot = loot
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("life: commit complete: %w", err)
	}
	committed = true
	return out, nil
}

func resolveCompleteLoot(ctx context.Context, tx *gen.Tx, in *LifeCompletePersist) (pkglife.LootResult, error) {
	if in == nil {
		return pkglife.LootResult{}, nil
	}
	if !in.ResolveLootInTx {
		return pkglife.LootResult{Roll: in.Dice}, nil
	}
	prof, err := tx.LifeProfile.Get(ctx, in.ProfileID)
	if err != nil {
		return pkglife.LootResult{}, fmt.Errorf("life: load profile for loot: %w", err)
	}
	pity := map[string]int{}
	if prof.PityByTier != nil {
		maps.Copy(pity, prof.PityByTier)
	}
	loot := pkglife.ResolveLoot(pkglife.LootInput{
		BaseDropChance: in.LootBaseChance, ProfileBonus: in.ProfileBonus, EquippedDropRate: in.EquippedDropRate,
		PityCount: pity[in.DropTier], PityThreshold: pkglife.DefaultPityThreshold, Roll: in.Dice, PoolSize: len(in.LootPool),
	})
	pity[in.DropTier] = loot.NextPity
	in.Pity = pity
	in.DropEquipID = 0
	in.NeedLore = false
	in.LoreStatus = "none"
	if !loot.Dropped || len(in.LootPool) == 0 {
		return loot, nil
	}
	eqFlag := in.LootPool[loot.PoolIndex]
	eq, err := tx.LifeEquipment.Query().Where(lifeequipment.FlagEQ(eqFlag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return pkglife.LootResult{}, fmt.Errorf("life: equipment %s missing", eqFlag)
		}
		return pkglife.LootResult{}, fmt.Errorf("life: load equipment %s: %w", eqFlag, err)
	}
	in.DropEquipID = eq.ID
	in.NeedLore = pkglife.NeedsInstanceLore(in.QuestType, in.Difficulty)
	if in.NeedLore {
		in.LoreStatus = "pending"
	}
	return loot, nil
}

// PersistFailQuest marks a quest failed and applies rust in one transaction.
func (s *LifeStore) PersistFailQuest(ctx context.Context, profileID, questID int64, rustInvIDs []int64, until time.Time) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("life: begin fail tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.LifeQuest.UpdateOneID(questID).SetStatus("Failed").Save(ctx); err != nil {
		return fmt.Errorf("life: mark failed: %w", err)
	}
	if _, err := tx.LifeEquippedSlots.Update().Where(lifeequippedslots.LifeProfileIDEQ(profileID)).SetTarnishedUntil(until).Save(ctx); err != nil {
		return fmt.Errorf("life: set slots rust: %w", err)
	}
	for _, id := range rustInvIDs {
		if _, err := tx.LifeInventory.UpdateOneID(id).SetTarnishedUntil(until).Save(ctx); err != nil {
			return fmt.Errorf("life: set inventory rust: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("life: commit fail: %w", err)
	}
	committed = true
	return nil
}

func persistCompleteCascade(ctx context.Context, tx *gen.Tx, in LifeCompletePersist) error {
	if _, err := tx.LifeSkill.UpdateOneID(in.SkillID).SetLevel(in.SkillLevel).SetCurrentExp(in.SkillExp).Save(ctx); err != nil {
		return fmt.Errorf("life: update skill: %w", err)
	}
	if _, err := tx.LifeCharacteristic.UpdateOneID(in.CharID).SetLevel(in.CharLevel).SetCurrentExp(in.CharExp).Save(ctx); err != nil {
		return fmt.Errorf("life: update characteristic: %w", err)
	}
	if _, err := tx.LifeProfile.UpdateOneID(in.ProfileID).
		SetLevel(in.ProfLevel).SetExp(in.ProfExp).SetGold(in.ProfGold).SetPityByTier(in.Pity).Save(ctx); err != nil {
		return fmt.Errorf("life: update profile: %w", err)
	}
	return nil
}

func clearCompleteRust(ctx context.Context, tx *gen.Tx, in LifeCompletePersist) error {
	if _, err := tx.LifeEquippedSlots.Update().Where(lifeequippedslots.LifeProfileIDEQ(in.ProfileID)).ClearTarnishedUntil().Save(ctx); err != nil {
		return fmt.Errorf("life: clear slots rust: %w", err)
	}
	for _, id := range in.RustInvIDs {
		if _, err := tx.LifeInventory.UpdateOneID(id).ClearTarnishedUntil().Save(ctx); err != nil {
			return fmt.Errorf("life: clear inventory rust: %w", err)
		}
	}
	return nil
}

func persistCompleteDrop(ctx context.Context, tx *gen.Tx, in LifeCompletePersist, now time.Time) (*LifeCompleteResult, *int64, error) {
	out := &LifeCompleteResult{}
	if in.DropEquipID <= 0 {
		return out, nil, nil
	}
	loreStatus := in.LoreStatus
	if loreStatus == "" {
		loreStatus = "none"
	}
	inv, err := tx.LifeInventory.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetEquipmentID(in.DropEquipID).
		SetLoreStatus(loreStatus).
		SetInstanceBuffs(map[string]any{}).
		SetSourceQuestID(in.DropQuestID).
		Save(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("life: create inventory: %w", err)
	}
	out.Inventory = inv
	invID := inv.ID
	eq, err := tx.LifeEquipment.Get(ctx, in.DropEquipID)
	if err != nil {
		return nil, nil, fmt.Errorf("life: load equipment: %w", err)
	}
	out.Equipment = eq
	if in.NeedLore {
		eventID := types.Id()
		payload := map[string]any{
			"event_id": eventID, "type": LifeLoreRequestedType,
			"life_profile_id": in.ProfileID, "inventory_id": inv.ID,
		}
		if _, err := tx.EventOutbox.Create().
			SetEventID(eventID).SetPayload(payload).SetPublished(false).SetCreatedAt(now).Save(ctx); err != nil {
			return nil, nil, fmt.Errorf("life: lore outbox: %w", err)
		}
	}
	return out, &invID, nil
}

func persistCompleteActionLog(ctx context.Context, tx *gen.Tx, in LifeCompletePersist, invID *int64) error {
	ab := tx.LifeActionLog.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetQuestID(in.QuestID).
		SetGainedExp(in.ActionExp).
		SetGainedGold(in.ActionGold).
		SetDiceRollResult(in.Dice)
	if invID != nil {
		ab = ab.SetDroppedInventoryID(*invID)
	}
	if _, err := ab.Save(ctx); err != nil {
		return fmt.Errorf("life: action log: %w", err)
	}
	return nil
}

func persistDailyRespawn(ctx context.Context, tx *gen.Tx, dq *gen.LifeQuest) error {
	if dq == nil {
		return nil
	}
	cb := tx.LifeQuest.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(dq.LifeProfileID).
		SetSkillID(dq.SkillID).
		SetTitle(dq.Title).
		SetPrompt(dq.Prompt).
		SetType("Daily").
		SetAiEvaluatedDifficulty(dq.AiEvaluatedDifficulty).
		SetAiEvaluatedFear(dq.AiEvaluatedFear).
		SetBaseExpReward(dq.BaseExpReward).
		SetBaseGoldReward(dq.BaseGoldReward).
		SetDropTier(dq.DropTier).
		SetStatus("Pending")
	if dq.GoalID != nil {
		cb = cb.SetGoalID(*dq.GoalID)
	}
	if _, err := cb.Save(ctx); err != nil {
		return fmt.Errorf("life: daily respawn: %w", err)
	}
	return nil
}

func persistCompleteAchievements(ctx context.Context, tx *gen.Tx, profileID int64, questType, difficulty string) ([]*gen.LifeAchievement, error) {
	catalogRows, err := tx.LifeAchievement.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list achievements: %w", err)
	}
	catalog := make([]pkglife.AchievementDef, 0, len(catalogRows))
	byFlag := make(map[string]*gen.LifeAchievement, len(catalogRows))
	for _, row := range catalogRows {
		byFlag[row.Flag] = row
		catalog = append(catalog, pkglife.AchievementDef{
			Flag: row.Flag, Name: row.Name, Description: row.Description, Active: row.Active,
			Kind: row.Kind, QuestType: row.QuestType, Difficulty: row.Difficulty,
			Threshold: row.Threshold, SortOrder: row.SortOrder,
		})
	}
	keys := pkglife.AchievementKeysForCompletion(questType, difficulty)
	progressRows, err := tx.LifeAchievementProgress.Query().
		Where(
			lifeachievementprogress.LifeProfileIDEQ(profileID),
			lifeachievementprogress.ConditionKeyIn(keys...),
		).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list achievement progress: %w", err)
	}
	progress := make(map[string]int, len(progressRows))
	progressByKey := make(map[string]*gen.LifeAchievementProgress, len(progressRows))
	for _, row := range progressRows {
		progress[row.ConditionKey] = row.CurrentCount
		progressByKey[row.ConditionKey] = row
	}
	unlockRows, err := tx.LifeAchievementUnlock.Query().
		Where(lifeachievementunlock.LifeProfileIDEQ(profileID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list achievement unlocks: %w", err)
	}
	unlocked := make(map[string]struct{}, len(unlockRows))
	for _, row := range unlockRows {
		unlocked[row.AchievementFlag] = struct{}{}
	}
	eval := pkglife.EvaluateAchievements(pkglife.AchievementEvalInput{
		QuestType: questType, Difficulty: difficulty,
		Catalog: catalog, Progress: progress, Unlocked: unlocked,
	})
	for key, count := range eval.ProgressAfter {
		if existing, ok := progressByKey[key]; ok {
			if _, err := tx.LifeAchievementProgress.UpdateOneID(existing.ID).
				SetCurrentCount(count).Save(ctx); err != nil {
				return nil, fmt.Errorf("life: update achievement progress: %w", err)
			}
			continue
		}
		if _, err := tx.LifeAchievementProgress.Create().
			SetFlag(types.Id()).
			SetLifeProfileID(profileID).
			SetConditionKey(key).
			SetCurrentCount(count).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("life: create achievement progress: %w", err)
		}
	}
	out := make([]*gen.LifeAchievement, 0, len(eval.NewlyUnlocked))
	for _, def := range eval.NewlyUnlocked {
		if _, err := tx.LifeAchievementUnlock.Create().
			SetFlag(types.Id()).
			SetLifeProfileID(profileID).
			SetAchievementFlag(def.Flag).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("life: unlock achievement: %w", err)
		}
		if row := byFlag[def.Flag]; row != nil {
			out = append(out, row)
		}
	}
	return out, nil
}

func createPlanNodeWithClient(ctx context.Context, client *gen.Client, in LifeCreatePlanNodeInput) (*gen.LifePlanNode, error) {
	status := in.Status
	if status == "" {
		status = "Active"
	}
	builder := client.LifePlanNode.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetNodeType(in.NodeType).
		SetTitle(in.Title).
		SetDescription(in.Description).
		SetStatus(status).
		SetSortOrder(in.SortOrder)
	if in.ParentID != nil {
		builder = builder.SetParentID(*in.ParentID)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create plan node: %w", err)
	}
	return row, nil
}

func createActionSpecWithClient(ctx context.Context, client *gen.Client, planNodeID int64, in LifePlanActionSpecInput) (*gen.LifeActionSpec, error) {
	taskType := in.TaskType
	if taskType == "" {
		taskType = "todo"
	}
	trackingMode := in.TrackingMode
	if trackingMode == "" {
		trackingMode = "completion"
	}
	repeatTrigger := in.RepeatTrigger
	if repeatTrigger == "" {
		repeatTrigger = "none"
	}
	difficulty := in.Difficulty
	if difficulty == "" {
		difficulty = "C"
	}
	builder := client.LifeActionSpec.Create().
		SetPlanNodeID(planNodeID).
		SetTaskType(taskType).
		SetTrackingMode(trackingMode).
		SetIsRepeatable(in.IsRepeatable).
		SetRepeatTrigger(repeatTrigger).
		SetSuggestedCadence(in.SuggestedCadence).
		SetIsIdentityBuilding(in.IsIdentityBuilding).
		SetReason(in.Reason).
		SetNeedsUserConfirmation(in.NeedsUserConfirmation).
		SetDifficulty(difficulty).
		SetBaseExpReward(in.BaseExpReward).
		SetBaseGoldReward(in.BaseGoldReward)
	if in.ConfirmedAt != nil {
		builder = builder.SetConfirmedAt(*in.ConfirmedAt)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create action spec: %w", err)
	}
	return row, nil
}

func collectPlanDescendantIDs(rows []*gen.LifePlanNode, rootID int64) []int64 {
	if rootID == 0 {
		return nil
	}
	children := map[int64][]int64{}
	for _, row := range rows {
		if row.ParentID == nil {
			continue
		}
		children[*row.ParentID] = append(children[*row.ParentID], row.ID)
	}
	out := make([]int64, 0, 8)
	stack := []int64{rootID}
	seen := map[int64]struct{}{}
	for len(stack) > 0 {
		last := len(stack) - 1
		id := stack[last]
		stack = stack[:last]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		stack = append(stack, children[id]...)
	}
	return out
}
