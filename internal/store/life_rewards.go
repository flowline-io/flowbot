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
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	q := s.client.LifeReward.Query().
		Where(lifereward.LifeProfileIDEQ(profileID)).
		Order(gen.Desc(lifereward.FieldCreatedAt))
	if activeOnly != nil {
		q = q.Where(lifereward.ActiveEQ(*activeOnly))
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list rewards: %w", err)
	}
	return rows, nil
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
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.client.LifeRewardRedemption.Query().
		Where(liferewardredemption.LifeProfileIDEQ(profileID)).
		Order(gen.Desc(liferewardredemption.FieldRedeemedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list reward redemptions: %w", err)
	}
	return rows, nil
}
