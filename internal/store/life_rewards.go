package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifereward"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/liferewardredemption"
	"github.com/flowline-io/flowbot/pkg/types"
)

// LifeRewardCreate is the write-set for inserting a player reward.
type LifeRewardCreate struct {
	Name          string
	Notes         string
	Price         int
	CooldownHours int
}

// CreateReward inserts an active player-defined reward.
func (s *LifeStore) CreateReward(ctx context.Context, profileID int64, in LifeRewardCreate) (*gen.LifeReward, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	row, err := s.client.LifeReward.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(profileID).
		SetName(in.Name).
		SetNotes(in.Notes).
		SetPrice(in.Price).
		SetCooldownHours(in.CooldownHours).
		SetActive(true).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create reward: %w", err)
	}
	return row, nil
}

// GetRewardByFlag returns a reward by flag for a profile.
func (s *LifeStore) GetRewardByFlag(ctx context.Context, profileID int64, flag string) (*gen.LifeReward, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeReward.Query().
		Where(lifereward.LifeProfileIDEQ(profileID), lifereward.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// ListRewards lists rewards for a profile, newest first.
// When activeOnly is non-nil, filters by active status.
func (s *LifeStore) ListRewards(ctx context.Context, profileID int64, activeOnly *bool) ([]*gen.LifeReward, error) {
	rows, _, err := s.ListRewardsPage(ctx, profileID, activeOnly, 0, 0)
	return rows, err
}

// ListRewardsPage returns a page of rewards and the total matching count.
// A non-positive limit returns all matching rows (offset ignored).
func (s *LifeStore) ListRewardsPage(ctx context.Context, profileID int64, activeOnly *bool, limit, offset int) ([]*gen.LifeReward, int, error) {
	if !s.ready() {
		return nil, 0, fmt.Errorf("life: store not available")
	}
	q := s.client.LifeReward.Query().Where(lifereward.LifeProfileIDEQ(profileID))
	if activeOnly != nil {
		q = q.Where(lifereward.ActiveEQ(*activeOnly))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: count rewards: %w", err)
	}
	q = q.Order(gen.Desc(lifereward.FieldCreatedAt), gen.Desc(lifereward.FieldID))
	if lim, off, ok := lifePageBounds(limit, offset); ok {
		q = q.Limit(lim).Offset(off)
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: list rewards page: %w", err)
	}
	return rows, total, nil
}

// UpdateReward updates mutable catalog fields for a reward.
func (s *LifeStore) UpdateReward(ctx context.Context, id int64, in LifeRewardCreate) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeReward.UpdateOneID(id).
		SetName(in.Name).
		SetNotes(in.Notes).
		SetPrice(in.Price).
		SetCooldownHours(in.CooldownHours).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("life: update reward: %w", err)
	}
	return nil
}

// SetRewardActive soft-deletes or restores a reward.
func (s *LifeStore) SetRewardActive(ctx context.Context, id int64, active bool) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeReward.UpdateOneID(id).SetActive(active).Save(ctx)
	if err != nil {
		return fmt.Errorf("life: set reward active: %w", err)
	}
	return nil
}

// SetProfileGold sets absolute gold balance on a profile.
func (s *LifeStore) SetProfileGold(ctx context.Context, id int64, gold int) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeProfile.UpdateOneID(id).SetGold(gold).Save(ctx)
	if err != nil {
		return fmt.Errorf("life: set profile gold: %w", err)
	}
	return nil
}

// MarkRewardRedeemed updates last_redeemed_at after a successful redeem.
func (s *LifeStore) MarkRewardRedeemed(ctx context.Context, id int64, at time.Time) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	_, err := s.client.LifeReward.UpdateOneID(id).SetLastRedeemedAt(at).Save(ctx)
	if err != nil {
		return fmt.Errorf("life: mark reward redeemed: %w", err)
	}
	return nil
}

// CreateRewardRedemption inserts one redemption audit row with price/name snapshots.
func (s *LifeStore) CreateRewardRedemption(ctx context.Context, profileID, rewardID int64, rewardName string, pricePaid int, at time.Time) (*gen.LifeRewardRedemption, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	row, err := s.client.LifeRewardRedemption.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(profileID).
		SetLifeRewardID(rewardID).
		SetRewardName(rewardName).
		SetPricePaid(pricePaid).
		SetRedeemedAt(at).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create reward redemption: %w", err)
	}
	return row, nil
}

// ListRewardRedemptions returns recent redemptions newest first.
func (s *LifeStore) ListRewardRedemptions(ctx context.Context, profileID int64, limit int) ([]*gen.LifeRewardRedemption, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, _, err := s.ListRewardRedemptionsPage(ctx, profileID, limit, 0)
	return rows, err
}

// ListRewardRedemptionsPage returns a page of redemptions and the total count.
// A non-positive limit returns all matching rows (offset ignored).
func (s *LifeStore) ListRewardRedemptionsPage(ctx context.Context, profileID int64, limit, offset int) ([]*gen.LifeRewardRedemption, int, error) {
	if !s.ready() {
		return nil, 0, fmt.Errorf("life: store not available")
	}
	q := s.client.LifeRewardRedemption.Query().Where(liferewardredemption.LifeProfileIDEQ(profileID))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: count reward redemptions: %w", err)
	}
	q = q.Order(gen.Desc(liferewardredemption.FieldRedeemedAt), gen.Desc(liferewardredemption.FieldID))
	if lim, off, ok := lifePageBounds(limit, offset); ok {
		q = q.Limit(lim).Offset(off)
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("life: list reward redemptions page: %w", err)
	}
	return rows, total, nil
}
