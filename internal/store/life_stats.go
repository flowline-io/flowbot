package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeachievementunlock"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactionlog"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifequest"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/liferewardredemption"
)

// ListActionLogsInRange returns action logs with created_at in [since, until).
func (s *LifeStore) ListActionLogsInRange(ctx context.Context, profileID int64, since, until time.Time) ([]*gen.LifeActionLog, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	rows, err := s.client.LifeActionLog.Query().
		Where(
			lifeactionlog.LifeProfileIDEQ(profileID),
			lifeactionlog.CreatedAtGTE(since),
			lifeactionlog.CreatedAtLT(until),
		).
		Order(gen.Asc(lifeactionlog.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list action logs in range: %w", err)
	}
	return rows, nil
}

// ListRewardRedemptionsInRange returns redemptions with redeemed_at in [since, until).
func (s *LifeStore) ListRewardRedemptionsInRange(ctx context.Context, profileID int64, since, until time.Time) ([]*gen.LifeRewardRedemption, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	rows, err := s.client.LifeRewardRedemption.Query().
		Where(
			liferewardredemption.LifeProfileIDEQ(profileID),
			liferewardredemption.RedeemedAtGTE(since),
			liferewardredemption.RedeemedAtLT(until),
		).
		Order(gen.Asc(liferewardredemption.FieldRedeemedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list reward redemptions in range: %w", err)
	}
	return rows, nil
}

// ListAchievementUnlocksInRange returns unlocks with unlocked_at in [since, until).
func (s *LifeStore) ListAchievementUnlocksInRange(ctx context.Context, profileID int64, since, until time.Time) ([]*gen.LifeAchievementUnlock, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	rows, err := s.client.LifeAchievementUnlock.Query().
		Where(
			lifeachievementunlock.LifeProfileIDEQ(profileID),
			lifeachievementunlock.UnlockedAtGTE(since),
			lifeachievementunlock.UnlockedAtLT(until),
		).
		Order(gen.Asc(lifeachievementunlock.FieldUnlockedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list achievement unlocks in range: %w", err)
	}
	return rows, nil
}

// ListQuestsCompletedInRange returns Completed quests with completed_at in [since, until).
func (s *LifeStore) ListQuestsCompletedInRange(ctx context.Context, profileID int64, since, until time.Time) ([]*gen.LifeQuest, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	rows, err := s.client.LifeQuest.Query().
		Where(
			lifequest.LifeProfileIDEQ(profileID),
			lifequest.StatusEQ("Completed"),
			lifequest.CompletedAtNotNil(),
			lifequest.CompletedAtGTE(since),
			lifequest.CompletedAtLT(until),
		).
		Order(gen.Asc(lifequest.FieldCompletedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list completed quests in range: %w", err)
	}
	return rows, nil
}

// ListQuestsByIDs returns quests for the given ids scoped to a profile.
func (s *LifeStore) ListQuestsByIDs(ctx context.Context, profileID int64, ids []int64) ([]*gen.LifeQuest, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.client.LifeQuest.Query().
		Where(lifequest.LifeProfileIDEQ(profileID), lifequest.IDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list quests by ids: %w", err)
	}
	return rows, nil
}
